package mail

import (
	"context"
	"log"
)

// LogSender writes every mail to the server log instead of sending it:
// enough for development, where the verification link is copied from the
// console.
type LogSender struct{}

var _ Sender = LogSender{}

func (LogSender) Deliver(_ context.Context, message Message) error {
	log.Printf("mail to %s: %s\n%s", message.To, message.Subject, message.Text)
	return nil
}
