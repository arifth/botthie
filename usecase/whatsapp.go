package usecase

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func SendDocumentAndImage(client *whatsmeow.Client, jid types.JID, filePath string, text string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	dataType := http.DetectContentType(data)
	fmt.Println(dataType)

	switch dataType {
	case "image/png":
		resp, err := client.Upload(context.Background(), data, whatsmeow.MediaDocument)
		if err != nil {
			return err
		}
		bodyMsg := &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(text),
				Mimetype:      proto.String(http.DetectContentType(data)),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			},
		}
		_, err = client.SendMessage(context.Background(), jid, bodyMsg)
		if err != nil {
			return err
		}
	default:
		resp, err := client.Upload(context.Background(), data, whatsmeow.MediaDocument)
		if err != nil {
			return err
		}

		bodyMsg := &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				FileName:      proto.String(filepath.Base(filePath)),
				Mimetype:      proto.String(http.DetectContentType(data)),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
				Caption:       proto.String(text),
			},
		}
		_, err = client.SendMessage(context.Background(), jid, bodyMsg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (Usecase) SendMessageAll(uc *Usecase, text string) {
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	_, err := uc.client.SendMessage(uc.ctx, uc.jdID, msg)
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
	}
}
