package utils

import (
	"downloader"
)

func JsonGetString(j interface{}, path string) (string, error) {
	return "", downloader.ErrUnimplemented
}

func JsonGetInt(j interface{}, path string) (int, error) {
	return 0, downloader.ErrUnimplemented
}

func JsonHasKey(j interface{}, path string) bool {
	// TODO implement
	return false
}

func JsonArraySize(j interface{}, path string) int {
	// TODO implement
	return 0
}
