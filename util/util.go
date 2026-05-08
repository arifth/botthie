package util

import (
	"math/rand"
	"os"
	"strings"
	"time"
)

func GetDataFromTemplate(path string) (string, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	cleansed := strings.ReplaceAll(string(data), "\n", "")
	return string(cleansed), nil
}

func GenerateRandomChars() string {
	const charset = "abcdefghijklmnopqrstuvwxyz123456789"

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	result := make([]byte, 4)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
