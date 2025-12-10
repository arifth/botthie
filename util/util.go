package util

import (
	"os"
	"strings"
)

func GetDataFromTemplate(path string) (string, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	cleansed := strings.ReplaceAll(string(data), "\n", "")
	return string(cleansed), nil
}
