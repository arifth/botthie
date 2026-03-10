package usecase

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type Usecase struct {
	client *whatsmeow.Client
	jdID   types.JID
	ctx    context.Context
}

func NewUsecase(ctx context.Context, client *whatsmeow.Client, jdID types.JID) *Usecase {
	return &Usecase{
		client: client,
		jdID:   jdID,
		ctx:    ctx,
	}
}
