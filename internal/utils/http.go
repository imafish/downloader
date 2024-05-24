package utils

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

type CachedHttpClient struct {
	// TODO: make this cache to LRU cache.
	cache map[string][]byte
}

func NewCachedHttpClient() *CachedHttpClient {
	return &CachedHttpClient{cache: make(map[string][]byte)}
}

// GetBody HTTP response with 'GET' verb
func (c *CachedHttpClient) GetBody(req *http.Request) ([]byte, error) {

	// try the cache
	headers := map[string][]string(req.Header)
	headerArr := make([]string, 0, len(headers))
	for k, v := range headers {
		headerArr = append(headerArr, fmt.Sprintf("%s-%v", k, v))
	}
	slices.Sort(headerArr)
	sb := strings.Builder{}
	sb.WriteString(req.URL.String())
	sb.WriteString(strings.Join(headerArr, "."))
	urlAndHeader := sb.String()

	if data, ok := c.cache[urlAndHeader]; ok {
		return data, nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET request got error: %v", err)
	}
	defer resp.Body.Close()

	var err2 error
	if resp.StatusCode != http.StatusOK {
		err2 = fmt.Errorf("http status code is %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	c.cache[urlAndHeader] = content
	return content, err2
}
