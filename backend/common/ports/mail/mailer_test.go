package mail

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

type recordingSender struct {
	delivered []Message
	err       error
}

func (s *recordingSender) Deliver(ctx context.Context, message Message) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("delivery without a deadline")
	}
	s.delivered = append(s.delivered, message)
	return s.err
}

func TestMailer_RendersAndDelivers(t *testing.T) {
	sender := &recordingSender{}
	mailer := New(mustLoad(t), sender)

	err := mailer.Send(models.Mail{To: "ana@example.com", Locale: models.LocaleSpanish, Template: models.MailVerifyEmail, Vars: vars()})
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.delivered) != 1 {
		t.Fatalf("delivered = %d", len(sender.delivered))
	}
	message := sender.delivered[0]
	if message.To != "ana@example.com" || message.Subject == "" || !strings.Contains(message.Text, "Ana") || !strings.Contains(message.HTML, "Ana") {
		t.Errorf("message = %+v", message)
	}
}

func TestMailer_DeliversNothingWhenRenderingFails(t *testing.T) {
	sender := &recordingSender{}
	mailer := New(mustLoad(t), sender)

	if err := mailer.Send(models.Mail{To: "ana@example.com", Template: "nope"}); err == nil {
		t.Error("expected a rendering error")
	}
	if len(sender.delivered) != 0 {
		t.Error("nothing must be delivered")
	}
}

func TestMailer_ReportsDeliveryFailures(t *testing.T) {
	sender := &recordingSender{err: errors.New("provider down")}
	mailer := New(mustLoad(t), sender)

	if err := mailer.Send(models.Mail{To: "ana@example.com", Template: models.MailVerifyEmail, Vars: vars()}); err == nil {
		t.Error("expected the delivery error")
	}
}
