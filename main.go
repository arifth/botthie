package main

import (
	"arifthalhah/waBot/model"
	"arifthalhah/waBot/usecase"
	"arifthalhah/waBot/util"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const (
	API_URL = "https://your-api-endpoint.com/process" // Replace with your actual API URL
)

// Global client variable to access in event handlers
var waClient *whatsmeow.Client

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

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	dataTempl, err := util.GetDataFromTemplate("./template/apiBook.html")

	if err != nil {
		fmt.Println(err)
	}

	// Setup logging
	dbLog := waLog.Stdout("Database", "INFO", true)

	ctx := context.Background()
	// Initialize SQLite store - note: no context parameter
	container, err := sqlstore.New(ctx, "sqlite3", "file:whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	ctx2 := context.Background()
	// Get first device - note: no context parameter
	deviceStore, err := container.GetFirstDevice(ctx2)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	waClient = whatsmeow.NewClient(deviceStore, clientLog)
	waClient.AddEventHandler(func(evt interface{}) {
		handleEvents(evt, dataTempl)
	})

	// Handle login (QR code or existing session)
	err = handleLogin(waClient)
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ Bot is running. Press Ctrl+C to stop.")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n🛑 Shutting down...")
	waClient.Disconnect()
}

// handleLogin manages the login process - either via QR code (first time) or existing session
func handleLogin(client *whatsmeow.Client) error {
	// Check if already logged in (has session)
	if client.Store.ID == nil {
		// First time login - need QR code
		fmt.Println("🔐 First time login detected. Please scan the QR code with WhatsApp.")
		fmt.Println("📱 Steps:")
		fmt.Println("   1. Open WhatsApp on your phone")
		fmt.Println("   2. Go to Settings > Linked Devices")
		fmt.Println("   3. Tap 'Link a Device'")
		fmt.Println("   4. Scan the QR code below\n")

		return loginWithQRCode(client)
	} else {
		// Already have session, just connect
		fmt.Println("🔄 Existing session found. Connecting...")
		err := client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect with existing session: %w", err)
		}
		fmt.Println("✅ Connected successfully!")
		return nil
	}
}

// loginWithQRCode handles the QR code generation and scanning process
func loginWithQRCode(client *whatsmeow.Client) error {
	// Get QR code channel
	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	// Connect to WhatsApp
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Listen for QR code events
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			// New QR code received
			fmt.Println("📷 QR Code generated:")
			fmt.Println("----------------------------------------")

			// Display QR in terminal
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)

			fmt.Println("----------------------------------------")

			// Save QR code as PNG file
			err := saveQRCodeImage(evt.Code)
			if err != nil {
				fmt.Printf("⚠️  Warning: Could not save QR code image: %v\n", err)
			} else {
				fmt.Println("💾 QR code also saved to 'qrcode.png'")
			}

			fmt.Println("\n⏳ Waiting for scan...")

		case "success":
			fmt.Println("✅ Successfully logged in!")
			return nil

		case "timeout":
			fmt.Println("⏱️  QR code expired. Generating new one...")

		case "error":
			return fmt.Errorf("QR code error occurred")

		default:
			fmt.Printf("ℹ️  Login event: %s\n", evt.Event)
		}
	}

	return nil
}

// saveQRCodeImage saves the QR code as a PNG file
func saveQRCodeImage(code string) error {
	// Generate QR code with medium error correction and 256x256 size
	err := qrcode.WriteFile(code, qrcode.Medium, 256, "qrcode.png")
	if err != nil {
		return fmt.Errorf("failed to write QR code file: %w", err)
	}
	return nil
}

func handleEvents(evt interface{}, templ string) {
	switch v := evt.(type) {
	case *events.Message:
		handleMessage(v, templ)
	}
}

func handleMessage(evt *events.Message, templ string) {
	// Skip messages sent by bot itself
	if evt.Info.IsFromMe {
		return
	}

	msg := evt.Message

	// Handle text messages
	if msg.GetConversation() != "" || msg.GetExtendedTextMessage() != nil {
		text := msg.GetConversation()
		if text == "" {
			text = msg.GetExtendedTextMessage().GetText()
		}

		if strings.HasPrefix(text, "/generate") {
			sendMessage(evt.Info.Chat, "Please send a Postman collection JSON file.")
		}
		return
	}

	// Handle document messages (JSON files)
	if msg.GetDocumentMessage() != nil {
		doc := msg.GetDocumentMessage()

		// Check if it's a JSON file
		if strings.HasSuffix(strings.ToLower(doc.GetFileName()), ".json") {
			handlePostmanCollection(evt.Info.Chat, doc, templ)
		}
	}
}

func handlePostmanCollection(chatJID types.JID, doc *waE2E.DocumentMessage, templ string) {

	ctx := context.Background()
	// Download the document
	data, err := waClient.Download(ctx, doc)
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to download file: %v", err))
		return
	}

	// Parse Postman collection
	var collection PostmanCollection
	err = json.Unmarshal(data, &collection)
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to parse Postman collection: %v", err))
		return
	}

	// Convert to HTML
	html := convertToHTML(collection, templ)

	// Write HTML to file in current directory
	// filename := sanitizeFilename(collection.Info.Name) + ".html"
	// err = writeHTMLToFile(filename, html)
	// if err != nil {
	// 	sendMessage(chatJID, fmt.Sprintf("Failed to write HTML file: %v", err))
	// 	return
	// }

	// sendMessage(chatJID, fmt.Sprintf("📄 HTML file saved as: %s", filename))

	// Send HTML file back to WhatsApp
	// err = sendHTMLFile(chatJID, filename)
	// if err != nil {
	// 	sendMessage(chatJID, fmt.Sprintf("Failed to send HTML file: %v", err))
	// 	return
	// }

	// Prepare API request
	// TODO : map model to request body
	// escapedHTML := escapeForJSON(html)

	bodyReq := model.ConfluencePage{
		Type:      "page",
		Title:     "Test",
		Ancestors: []model.Ancestor{{ID: os.Getenv("PARENT_ID")}},
		Space:     model.Space{Key: os.Getenv("SPACE_KEY")},
		Body: model.BodyWrapper{
			Storage: model.Storage{
				Value:          string(html),
				Representation: "storage",
			},
		},
	}

	//TODO: map value to struct
	reqBody, err := json.Marshal(bodyReq)
	if err != nil {
		fmt.Println("error when marshalling req body", err)
	}
	resConflu, err := usecase.PostToConfluence(string(reqBody))
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to prepare API request: %v", err))
		return
	}

	if resConflu.IsSuccess() {
		sendMessage(chatJID, "sukses create page to confluence ")
	}

	// // Send to API
	// resp, err := http.Post(API_URL, "application/json", strings.NewReader(string(reqBody)))
	// if err != nil {
	// 	sendMessage(chatJID, fmt.Sprintf("Failed to send to API: %v", err))
	// 	return
	// }
	// defer resp.Body.Close()

	// // Read response
	// respBody, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	sendMessage(chatJID, fmt.Sprintf("Failed to read API response: %v", err))
	// 	return
	// }

	// // Send response back to user
	// responseMsg := fmt.Sprintf("✅ *API Response (Status: %d)*\n\n```%s```", resp.StatusCode, string(respBody))
	// sendMessage(chatJID, responseMsg)
}

func convertToHTML(collection PostmanCollection, dataTempl string) string {
	// Prepare template data
	data := TemplateData{
		CollectionName: collection.Info.Name,
		Requests:       make([]RequestData, 0),
	}

	// Extract request data
	for _, item := range collection.Item {
		reqData := RequestData{
			Name:    item.Name,
			Method:  item.Request.Method,
			URL:     extractURL(item.Request.URL),
			Headers: item.Request.Header,
		}

		// Parse body based on mode
		if item.Request.Body != nil {
			reqData.BodyMode = item.Request.Body.Mode

			// Check if body is JSON with fields
			if item.Request.Body.Mode == "raw" && item.Request.Body.Raw != "" {
				// Try to parse as JSON to extract fields
				bodyFields := parseJSONBodyFields(item.Request.Body.Raw)
				if len(bodyFields) > 0 {
					reqData.BodyFields = bodyFields
				} else {
					// If not valid JSON or no fields, just show raw
					reqData.Body = item.Request.Body.Raw
				}
			} else if item.Request.Body.Mode == "formdata" && len(item.Request.Body.FormData) > 0 {
				// Parse form-data fields
				for idx, field := range item.Request.Body.FormData {
					reqData.BodyFields = append(reqData.BodyFields, BodyField{
						Number:      idx + 1,
						Field:       field.Key,
						Type:        determineType(field.Value),
						Mandatory:   "No",
						Description: makeReadable(field.Key),
					})
				}
			} else if item.Request.Body.Mode == "urlencoded" && len(item.Request.Body.URLEncoded) > 0 {
				// Parse URL-encoded fields
				for idx, field := range item.Request.Body.URLEncoded {
					reqData.BodyFields = append(reqData.BodyFields, BodyField{
						Number:      idx + 1,
						Field:       field.Key,
						Type:        determineType(field.Value),
						Mandatory:   "No",
						Description: makeReadable(field.Key),
					})
				}
			} else if item.Request.Body.Raw != "" {
				reqData.Body = item.Request.Body.Raw
			}
		}

		data.Requests = append(data.Requests, reqData)
	}

	// Parse and execute template
	t := template.Must(template.New("postman").Parse(dataTempl))

	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		return fmt.Sprintf("Template execution error: %v", err)
	}

	return string(buf)
}

// parseJSONBodyFields parses JSON body and extracts field names with their types
func parseJSONBodyFields(rawBody string) []BodyField {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(rawBody), &jsonData)
	if err != nil {
		return nil
	}

	var fields []BodyField
	index := 1
	for key, value := range jsonData {
		fields = append(fields, BodyField{
			Number:      index,
			Field:       key,
			Type:        determineType(value),
			Mandatory:   "No", // Default to No, can be customized
			Description: generateDescription(key, value),
		})
		index++
	}

	return fields
}

// generateDescription generates a description based on field name and value
func generateDescription(fieldName string, value interface{}) string {
	// Convert field name from camelCase/snake_case to readable format
	readable := makeReadable(fieldName)

	// Generate smart description based on type
	valueType := determineType(value)

	switch valueType {
	case "string":
		if strVal, ok := value.(string); ok && strVal != "" {
			return fmt.Sprintf("%s (example: %s)", readable, strVal)
		}
		return readable
	case "integer", "number":
		return fmt.Sprintf("%s value", readable)
	case "boolean":
		return fmt.Sprintf("%s flag", readable)
	case "array":
		return fmt.Sprintf("List of %s", readable)
	case "object":
		return fmt.Sprintf("%s object details", readable)
	default:
		return readable
	}
}

// makeReadable converts field names to readable format
func makeReadable(fieldName string) string {
	// Replace underscores with spaces
	result := strings.ReplaceAll(fieldName, "_", " ")

	// Add space before capital letters (camelCase)
	var readable strings.Builder
	for i, r := range result {
		if i > 0 && r >= 'A' && r <= 'Z' {
			readable.WriteRune(' ')
		}
		if i == 0 {
			readable.WriteRune(r)
		} else {
			readable.WriteRune(r)
		}
	}

	// Capitalize first letter
	finalResult := readable.String()
	if len(finalResult) > 0 {
		finalResult = strings.ToUpper(string(finalResult[0])) + finalResult[1:]
	}

	return finalResult
}

// determineType determines the data type from a value
func determineType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64:
		return "integer"
	case float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func extractURL(urlInterface interface{}) string {
	switch v := urlInterface.(type) {
	case string:
		return v
	case map[string]interface{}:
		if raw, ok := v["raw"].(string); ok {
			return raw
		}
	}
	return ""
}

func escapeForJSON(s string) string {
	// Escape double quotes and backslashes for JSON string
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func sendMessage(chatJID types.JID, text string) {
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	_, err := waClient.SendMessage(context.Background(), chatJID, msg)
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
	}
}

// writeHTMLToFile writes the HTML content to a file in the current directory
func writeHTMLToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	fmt.Printf("✅ HTML written to file: %s\n", filename)
	return nil
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Remove leading/trailing spaces
	result = strings.TrimSpace(result)

	// If name is empty after sanitization, use default
	if result == "" {
		result = "postman_collection"
	}

	return result
}

// sendHTMLFile sends the HTML file back to WhatsApp
func sendHTMLFile(chatJID types.JID, filename string) error {
	// Read the file
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}

	// Upload the file to WhatsApp servers
	uploaded, err := waClient.Upload(context.Background(), fileData, whatsmeow.MediaDocument)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Create document message
	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("text/html"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(fileData))),
			FileName:      proto.String(filename),
			Caption:       proto.String("📄 Generated HTML Documentation"),
		},
	}

	// Send the message
	_, err = waClient.SendMessage(context.Background(), chatJID, msg)
	if err != nil {
		return fmt.Errorf("failed to send document message: %w", err)
	}

	fmt.Printf("✅ HTML file sent to WhatsApp: %s\n", filename)
	return nil
}
