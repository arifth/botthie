package config

import (
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// Config holds the configuration for the HTTP client
type Config struct {
	BaseURL       string
	Headers       map[string]string
	Timeout       time.Duration
	RetryCount    int
	RetryWaitTime time.Duration
	RetryMaxWait  time.Duration
	Debug         bool
}

// Client wraps the resty client with configuration
type Client struct {
	resty  *resty.Client
	config *Config
}

// NewClient creates a new HTTP client with the provided configuration
func NewClient(cfg *Config) *Client {
	// Set defaults if not provided
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.RetryCount == 0 {
		cfg.RetryCount = 3
	}
	if cfg.RetryWaitTime == 0 {
		cfg.RetryWaitTime = 1 * time.Second
	}
	if cfg.RetryMaxWait == 0 {
		cfg.RetryMaxWait = 5 * time.Second
	}

	// Create resty client
	client := resty.New()
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	client.SetBasicAuth(username, password)
	client.SetHeader("Content-Type", "application/json")
	client.SetDebug(true)

	// Configure base URL
	if cfg.BaseURL != "" {
		client.SetBaseURL(cfg.BaseURL)
	}

	// Configure headers
	if cfg.Headers != nil {
		client.SetHeaders(cfg.Headers)
	}

	// Configure timeout and retries
	client.
		SetTimeout(cfg.Timeout).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(cfg.RetryWaitTime).
		SetRetryMaxWaitTime(cfg.RetryMaxWait)

	// Enable debug mode if requested
	if cfg.Debug {
		client.SetDebug(true)
	}

	return &Client{
		resty:  client,
		config: cfg,
	}
}

// GetRestyClient returns the underlying resty client for advanced usage
func (c *Client) GetRestyClient() *resty.Client {
	return c.resty
}

// Get performs a GET request
func (c *Client) Get(url string) (*resty.Response, error) {
	return c.resty.R().Get(url)
}

// Post performs a POST request with a body
func (c *Client) Post(url string, body interface{}) (*resty.Response, error) {
	return c.resty.R().SetBody(body).Post(url)
}

// Put performs a PUT request with a body
func (c *Client) Put(url string, body interface{}) (*resty.Response, error) {
	return c.resty.R().SetBody(body).Put(url)
}

// Delete performs a DELETE request
func (c *Client) Delete(url string) (*resty.Response, error) {
	return c.resty.R().Delete(url)
}

// Patch performs a PATCH request with a body
func (c *Client) Patch(url string, body interface{}) (*resty.Response, error) {
	return c.resty.R().SetBody(body).Patch(url)
}

// NewRequest creates a new request with the configured client
func (c *Client) NewRequest() *resty.Request {
	return c.resty.R()
}

// SetAuthToken sets an authorization bearer token
func (c *Client) SetAuthToken(token string) *Client {
	c.resty.SetAuthToken(token)
	return c
}

// SetHeader sets a custom header for all requests
func (c *Client) SetHeader(key, value string) *Client {
	c.resty.SetHeader(key, value)
	return c
}

// Example usage
/*
func main() {
	// Create client configuration
	config := &httpclient.Config{
		BaseURL: "https://api.example.com",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"User-Agent":   "MyApp/1.0",
		},
		Timeout:    10 * time.Second,
		RetryCount: 3,
		Debug:      true,
	}

	// Instantiate the client
	client := httpclient.NewClient(config)

	// Set auth token if needed
	client.SetAuthToken("your-auth-token")

	// Perform a GET request
	resp, err := client.Get("/users")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Status:", resp.Status())
	fmt.Println("Body:", string(resp.Body()))

	// Perform a POST request
	userData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}
	resp, err = client.Post("/users", userData)
	if err != nil {
		log.Fatal(err)
	}

	// For advanced usage, get the underlying resty client
	restyClient := client.GetRestyClient()
	resp, err = restyClient.R().
		SetQueryParam("page", "1").
		SetHeader("X-Custom-Header", "value").
		Get("/users")
}
*/
