package util

import "github.com/arifth/botthie/model"

func Validate(collection model.PostmanCollection) bool {
	// valid template only consist of one level
	for _, item := range collection.Item {
		if item.Name == "" || item.Request.Method == "" || item.Request.URL == nil || item.Request.Header == nil {
			return false
		}
	}
	return true
}
