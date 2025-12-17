package model

// APIRequest represents the request to send to your API
type APIRequest struct {
	HTMLContent string `json:"request"`
}

// TemplateData holds data for HTML template rendering
type TemplateData struct {
	CollectionName string
	Requests       []RequestData
}

type RequestData struct {
	Name       string
	Method     string
	URL        string
	Headers    []PostmanHeader
	Body       string
	BodyFields []BodyField
	BodyMode   string
}

type BodyField struct {
	Field       string
	Type        string
	Mandatory   string
	Description string
	Number      int
}

type Links struct {
	Webui string `json:"webui"`
	Self  string `json:"self"`
}

type Spaces struct {
	ID    int    `json:"id"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Links Links  `json:"_links"`
}

type LinksS struct {
	Links
}

type ConfluenceResponse struct {
	Space Spaces  `json:"space"`
	Links LinksS `json: "_links`
}

type Response struct {
	StatusCode int    `json:"statusCode"`
	Data       Data   `json:"data"`
	Message    string `json:"message"`
	Reason     string `json:"reason"`
}

type Data struct {
	Authorized            bool          `json:"authorized"`
	Valid                 bool          `json:"valid"`
	AllowedInReadOnlyMode bool          `json:"allowedInReadOnlyMode"`
	Errors                []interface{} `json:"errors"`
	Successful            bool          `json:"successful"`
}