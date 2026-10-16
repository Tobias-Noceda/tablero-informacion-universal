package models

// MailTemplate names a mail the platform knows how to write, in every locale.
type MailTemplate string

const (
	MailVerifyEmail   MailTemplate = "verify_email"
	MailResetPassword MailTemplate = "reset_password"
	MailAccountExists MailTemplate = "account_exists"
)

var MailTemplates = []MailTemplate{MailVerifyEmail, MailResetPassword, MailAccountExists}

// Mail is what a service wants sent: to whom, in their language, which
// template and the values it fills in. The mailer owns the wording.
type Mail struct {
	To       string
	Locale   Locale
	Template MailTemplate
	Vars     map[string]string
}
