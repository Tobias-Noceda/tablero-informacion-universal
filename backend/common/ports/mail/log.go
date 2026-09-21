package mail

import (
	"log"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// Log writes every mail to the server log instead of sending it: enough for
// development, where the verification link is copied from the console.
type Log struct{}

var _ infrastructure.Mailer = Log{}

func (Log) Send(mail models.Mail) error {
	log.Printf("mail to %s: %s\n%s", mail.To, mail.Subject, mail.Text)
	return nil
}
