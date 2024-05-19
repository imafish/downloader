package utils

import (
	"fmt"
	"io"
	"net/http"
)

type CachedHttpClient struct {
	// TODO: make this cache to LRU cache.
	cache map[string][]byte
}

func NewCachedHttpClient() *CachedHttpClient {
	return &CachedHttpClient{cache: make(map[string][]byte)}
}

// Get HTTP response with 'GET' verb
func (c *CachedHttpClient) Get(req *http.Request) ([]byte, error) {

	// try the cache
	if data, ok := c.cache[req.URL.String()]; ok {
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

	c.cache[req.URL.String()] = content
	return content, err2
}
