package firebase

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

type Client struct {
	msg *messaging.Client
}

func New(path string) (*Client, error) {
	opt := option.WithCredentialsFile(path)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("firebase error initializing app: %v", err)
	}
	msg, _ := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}
	return &Client{msg: msg}, nil
}

func (s *Client) Send(ctx context.Context, token, title, content string) (string, error) {
	return s.msg.Send(ctx, &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  content,
		},
		Token: token,
	})
}
