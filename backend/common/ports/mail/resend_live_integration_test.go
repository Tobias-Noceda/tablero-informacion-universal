//go:build integration

package mail

import (
	"os"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

// TestLiveResend sends a real verification mail. It needs RESEND_API_KEY and
// RESEND_TEST_TO; without a verified domain use MAIL_FROM=onboarding@resend.dev
// and RESEND_TEST_TO=delivered@resend.dev.
func TestLiveResend(t *testing.T) {
	key, to := os.Getenv("RESEND_API_KEY"), os.Getenv("RESEND_TEST_TO")
	if key == "" || to == "" {
		t.Skip("RESEND_API_KEY and RESEND_TEST_TO are not set")
	}
	from := os.Getenv("MAIL_FROM")
	if from == "" {
		from = "onboarding@resend.dev"
	}

	mailer := New(mustLoad(t), NewResendSender(key, from))

	for _, locale := range models.Locales {
		err := mailer.Send(models.Mail{
			To:       to,
			Locale:   locale,
			Template: models.MailVerifyEmail,
			Vars:     map[string]string{"Name": "Integration test", "Link": "https://example.com/verify?token=live-test"},
		})
		if err != nil {
			t.Fatalf("%s: %v", locale, err)
		}
	}
}
