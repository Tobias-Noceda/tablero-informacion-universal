package mail

import (
	"context"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

const deliveryTimeout = 15 * time.Second

// Message is a rendered mail as a provider takes it.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Sender delivers a rendered mail: Resend in production, the log in
// development.
type Sender interface {
	Deliver(ctx context.Context, message Message) error
}

// Mailer turns what a service asks for into a written mail and hands it to
// the sender.
type Mailer struct {
	templates *Templates
	sender    Sender
}

var _ infrastructure.Mailer = (*Mailer)(nil)

func New(templates *Templates, sender Sender) *Mailer {
	return &Mailer{templates, sender}
}

func (m *Mailer) Send(mail models.Mail) error {
	rendered, err := m.templates.Render(mail)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), deliveryTimeout)
	defer cancel()

	return m.sender.Deliver(ctx, Message{
		To:      mail.To,
		Subject: rendered.Subject,
		Text:    rendered.Text,
		HTML:    rendered.HTML,
	})
}
