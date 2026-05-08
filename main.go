package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/arifth/botthie/model"
	"github.com/arifth/botthie/usecase"
	"github.com/arifth/botthie/util"

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

// Global client variable to access in event handlers
var waClient *whatsmeow.Client

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
		usecase.FirstAttempt()
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
		//instantiate new dependency per request
		ctx := context.Background()
		uc := usecase.NewUsecase(ctx, waClient, evt.Info.Chat)
		// Check if it's a JSON file
		if strings.HasSuffix(strings.ToLower(doc.GetFileName()), ".json") {
			handlePostmanCollection(uc, evt.Info.Chat, doc, templ)
		}
	}
}

func handlePostmanCollection(uc *usecase.Usecase, chatJID types.JID, doc *waE2E.DocumentMessage, templ string) {
	ctx := context.Background()
	// Download the document
	data, err := waClient.Download(ctx, doc)
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to download file: %v", err))
		return
	}

	// Parse Postman collection
	var collection model.PostmanCollection
	err = json.Unmarshal(data, &collection)
	if err != nil {
		sendMessage(chatJID, fmt.Sprintf("Failed to parse Postman collection: %v", err))
		return
	}

	valid := util.Validate(collection)
	if !valid {
		errorMsg := "Collection Postman tidak valid, mohon sesuaikan dengan template berikut"
		err := usecase.SendDocumentAndImage(waClient, chatJID, ".env", errorMsg)
		if err != nil {
			return
		}
		//sendMessage(chatJID, fmt.Sprintf("Invalid Postman collection: %v", err))
		return
	}

	_, err = uc.PostBulkToConfluence(collection, templ, uc)
	if err != nil {
		uc.SendMessageAll(uc, "error sending postman collection")
	}
}

// determineType determines the data type from a value
func sendMessage(chatJID types.JID, text string) {
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	_, err := waClient.SendMessage(context.Background(), chatJID, msg)
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
	}
}
