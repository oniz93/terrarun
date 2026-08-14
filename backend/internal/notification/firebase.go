package notification

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

type FirebaseClient struct {
	enabled bool
}

func NewFirebaseClient() *FirebaseClient {
	return &FirebaseClient{enabled: false}
}

type PushNotification struct {
	Token string            `json:"token"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
}

func (f *FirebaseClient) Send(ctx context.Context, notif PushNotification) error {
	if !f.enabled {
		log.Debug().Msg("firebase push disabled, skipping")
		return nil
	}
	return fmt.Errorf("firebase not configured")
}

func (f *FirebaseClient) SendToDevice(ctx context.Context, deviceToken, title, body string, data map[string]string) error {
	return f.Send(ctx, PushNotification{
		Token: deviceToken,
		Title: title,
		Body:  body,
		Data:  data,
	})
}
