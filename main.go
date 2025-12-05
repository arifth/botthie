package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

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
	Mode string `json:"mode"`
	Raw  string `json:"raw,omitempty"`
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
	Name    string
	Method  string
	URL     string
	Headers []PostmanHeader
	Body    string
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.CollectionName}}</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f9f9f9;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #ddd;
            padding-bottom: 10px;
        }
        .request {
            border: 1px solid #ddd;
            padding: 15px;
            margin: 15px 0;
            border-radius: 5px;
            background-color: white;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .request h2 {
            margin-top: 0;
            color: #555;
        }
        .method {
            font-weight: bold;
            color: #fff;
            padding: 5px 10px;
            border-radius: 3px;
            display: inline-block;
            margin-right: 10px;
        }
        .GET { background: #61affe; }
        .POST { background: #49cc90; }
        .PUT { background: #fca130; }
        .DELETE { background: #f93e3e; }
        .PATCH { background: #50e3c2; }
        .section {
            margin: 15px 0;
        }
        .label {
            font-weight: bold;
            color: #333;
            display: block;
            margin-bottom: 5px;
        }
        pre {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 3px;
            overflow-x: auto;
            border-left: 3px solid #61affe;
            margin: 5px 0;
        }
        code {
            background: #f0f0f0;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }
        .header-item {
            padding: 3px 0;
        }
    </style>
</head>
<body>
    <h1>{{.CollectionName}}</h1>
    {{range .Requests}}
    <div class="request">
        <h2>{{.Name}}</h2>
        
        <div class="section">
            <span class="label">Method:</span>
            <span class="method {{.Method}}">{{.Method}}</span>
        </div>
        
        <div class="section">
            <span class="label">URL:</span>
            <code>{{.URL}}</code>
        </div>
        
        {{if .Headers}}
        <div class="section">
            <span class="label">Headers:</span>
            <pre>{{range .Headers}}{{.Key}}: {{.Value}}
{{end}}</pre>
        </div>
        {{end}}
        
        {{if .Body}}
        <div class="section">
            <span class="label">Body:</span>
            <pre>{{.Body}}</pre>
        </div>
        {{end}}
    </div>
    {{end}}
</body>
</html>`

func main() {
	// Setup logging
	dbLog := waLog.Stdout("Database", "INFO", true)

	// Initialize SQLite store - note: no context parameter
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	// Get first device - note: no context parameter
	ctx2 := context.Background()
	deviceStore, err := container.GetFirstDevice(ctx2)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	waClient = whatsmeow.NewClient(deviceStore, clientLog)
	waClient.AddEventHandler(handleEvents)

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

func handleEvents(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		handleMessage(v)
	}
}

func handleMessage(evt *events.Message) {
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
			handlePostmanCollection(evt.Info.Chat, doc)
		}
	}
}

func handlePostmanCollection(chatJID types.JID, doc *waProto.DocumentMessage) {
	// Download the document
	ctx := context.Background()
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
	html := convertToHTML(collection)

	// Prepare API request
	escapedHTML := escapeForJSON(html)
	apiReq := APIRequest{
		HTMLContent: escapedHTML,
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to prepare API request: %v", err))
		return
	}

	// Send to API
	resp, err := http.Post(API_URL, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to send to API: %v", err))
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to read API response: %v", err))
		return
	}

	// Send response back to user
	responseMsg := fmt.Sprintf("✅ *API Response (Status: %d)*\n\n```%s```", resp.StatusCode, string(respBody))
	sendMessage(chatJID, responseMsg)
}

func convertToHTML(collection PostmanCollection) string {
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

		if item.Request.Body != nil && item.Request.Body.Raw != "" {
			reqData.Body = item.Request.Body.Raw
		}

		data.Requests = append(data.Requests, reqData)
	}

	// Parse and execute template
	tmpl, err := template.New("postman").Parse(htmlTemplate)
	if err != nil {
		return fmt.Sprintf("Template parsing error: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Sprintf("Template execution error: %v", err)
	}

	return buf.String()
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
	msg := &waProto.Message{
		Conversation: proto.String(text),
	}

	_, err := waClient.SendMessage(context.Background(), chatJID, msg)
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
	}
}
