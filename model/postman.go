package model

// PostmanCollection represents the structure of a Postman collection
type PostmanCollection struct {
	Info struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"info"`
	Item []PostmanItem `json:"item"`
}

type PostmanItem struct {
	Name    string         `json:"name"`
	Request PostmanRequest `json:"request"`
}

type PostmanRequest struct {
	Method string          `json:"method"`
	Header []PostmanHeader `json:"header"`
	Body   *PostmanBody    `json:"body,omitempty"`
	URL    interface{}     `json:"url"`
}

type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type PostmanBody struct {
	Mode       string                `json:"mode"`
	Raw        string                `json:"raw,omitempty"`
	FormData   []PostmanFormDataItem `json:"formdata,omitempty"`
	URLEncoded []PostmanFormDataItem `json:"urlencoded,omitempty"`
}

type PostmanFormDataItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type PostmanURL struct {
	Raw  string   `json:"raw"`
	Host []string `json:"host,omitempty"`
	Path []string `json:"path,omitempty"`
}
