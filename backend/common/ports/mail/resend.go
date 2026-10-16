package mail

import (
	"context"
	"fmt"
	"net/http"

	"github.com/resend/resend-go/v4"
)

// ResendSender delivers through the Resend API from one verified address.
type ResendSender struct {
	client *resend.Client
	from   string
}

var _ Sender = (*ResendSender)(nil)

func NewResendSender(apiKey, from string) *ResendSender {
	client := resend.NewCustomClient(&http.Client{Timeout: deliveryTimeout}, apiKey)
	return &ResendSender{client, from}
}

func (s *ResendSender) Deliver(ctx context.Context, message Message) error {
	_, err := s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{message.To},
		Subject: message.Subject,
		Text:    message.Text,
		Html:    message.HTML,
	})
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	return nil
}
