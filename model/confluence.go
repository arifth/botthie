package model

// The top-level struct matching the main JSON object
type ConfluencePage struct {
	Type      string      `json:"type"`
	Title     string      `json:"title"`
	Ancestors []Ancestor  `json:"ancestors"`
	Space     Space       `json:"space"`
	Body      BodyWrapper `json:"body"`
}

// Struct for the "ancestors" array elements
type Ancestor struct {
	ID string `json:"id"`
}

// Struct for the "space" object
type Space struct {
	Key string `json:"key"`
}

// Struct for the "body" object
type BodyWrapper struct {
	Storage Storage `json:"storage"`
}

// Struct for the "storage" object
type Storage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}
