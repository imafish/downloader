package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type JsonNode struct {
	data interface{}
}

const (
	JsonTypeInt    = iota
	JsonTypeFloat  = iota
	JsonTypeString = iota
	JsonTypeArray  = iota
	JsonTypeObject = iota
	JsonTypeError  = iota
)

func (n JsonNode) Type() int {
	switch n.data.(type) {
	case int:
		return JsonTypeInt
	case float64:
		return JsonTypeFloat
	case string:
		return JsonTypeString
	case []interface{}:
		return JsonTypeArray
	case map[string]interface{}:
		return JsonTypeObject
	}
	return JsonTypeError
}

func NewJsonNode(data interface{}) *JsonNode {
	return &JsonNode{data: data}
}

func UnmarshalJson(bytes []byte) (*JsonNode, error) {
	var v interface{}
	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return nil, err
	}
	return NewJsonNode(v), nil
}

func (n *JsonNode) HasField(path string) bool {
	paths := strings.Split(path, ".")
	currentNode := n.data
	regex := regexp.MustCompile(`\[(%d+)\]`)
	for _, currentPath := range paths {
		if tmp, ok := currentNode.(*JsonNode); ok {
			currentNode = tmp.data
		}

		if currentPath == "" {
			continue
		} else if matches := regex.FindStringSubmatch(currentPath); matches != nil {
			idx, _ := strconv.Atoi(matches[1])
			if arr, ok := currentNode.([]interface{}); ok {
				if len(arr) > idx {
					currentNode = arr[idx]
				} else {
					return false
				}
			} else {
				return false
			}
		} else {
			if mp, ok := currentNode.(map[string]interface{}); ok {
				if obj, ok2 := mp[currentPath]; ok2 {
					currentNode = obj
				} else {
					return false
				}
			} else {
				return false
			}
		}
	}

	return true
}

func (n *JsonNode) GetSubnode(path string) (*JsonNode, error) {
	paths := strings.Split(path, ".")
	currentNode := n.data
	regex := regexp.MustCompile(`\[(\d+)\]`)

	for _, currentPath := range paths {
		if tmp, ok := currentNode.(*JsonNode); ok {
			currentNode = tmp.data
		}

		if currentPath == "" {
			continue
		} else if matches := regex.FindStringSubmatch(currentPath); matches != nil {
			idx, _ := strconv.Atoi(matches[1])
			if arr, ok := currentNode.([]interface{}); ok {
				if len(arr) > idx {
					currentNode = arr[idx]
				} else {
					return nil, fmt.Errorf("length of field is too small for path %s", currentPath)
				}
			} else {
				return nil, fmt.Errorf("field is not an array")
			}
		} else {
			if mp, ok := currentNode.(map[string]interface{}); ok {
				if obj, ok2 := mp[currentPath]; ok2 {
					currentNode = obj
				} else {
					return nil, fmt.Errorf("expected field %s does not exist", currentPath)
				}
			} else {
				return nil, fmt.Errorf("field is not an object")
			}
		}
	}

	return NewJsonNode(currentNode), nil
}

func (n *JsonNode) GetInt(path string) (int, error) {
	subnode, err := n.GetSubnode(path)
	if err != nil {
		return 0, err
	}
	number, ok := subnode.data.(float64)
	if !ok {
		return 0, fmt.Errorf("the field is not a number")
	}
	integer := int(number)
	small := 0.00000000001
	if float64(integer)-number > small || number-float64(integer) > small {
		return 0, fmt.Errorf("the field is not an integer")
	}

	return integer, nil
}

func (n *JsonNode) GetFloat(path string) (float64, error) {
	subnode, err := n.GetSubnode(path)
	if err != nil {
		return 0, err
	}
	ret, ok := subnode.data.(float64)
	if !ok {
		return 0, fmt.Errorf("cannot convert json node to float")
	}
	return ret, nil
}

func (n *JsonNode) GetString(path string) (string, error) {
	subnode, err := n.GetSubnode(path)
	if err != nil {
		return "", err
	}
	ret, ok := subnode.data.(string)
	if !ok {
		return "", fmt.Errorf("cannot convert json node to string")
	}
	return ret, nil
}

func (n *JsonNode) GetArray(path string) ([]interface{}, error) {
	subnode, err := n.GetSubnode(path)
	if err != nil {
		return nil, err
	}
	ret, ok := subnode.data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot convert json node to array")
	}
	return ret, nil
}

func (n *JsonNode) GetMap(path string) (map[string]interface{}, error) {
	subnode, err := n.GetSubnode(path)
	if err != nil {
		return nil, err
	}
	ret, ok := subnode.data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot convert json node to map")
	}
	return ret, nil
}

func (n *JsonNode) MustAsInt() int {
	return n.data.(int)
}

func (n *JsonNode) MustAsFloat() float64 {
	return n.data.(float64)
}

func (n *JsonNode) MustAsString() string {
	return n.data.(string)
}

func (n *JsonNode) MustAsSlice() []interface{} {
	return n.data.([]interface{})
}

func (n *JsonNode) MustAsMap() map[string]interface{} {
	return n.data.(map[string]interface{})
}
