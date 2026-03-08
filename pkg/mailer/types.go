package mailer

import (
	"file-service/pkg/mailer/providers"
	"file-service/pkg/mailer/registry"
	"net/mail"
)

func ValidateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	return err
}

func ValidateEmailData(data *providers.EmailData) error {
	if data == nil {
		return registry.ErrEmailDataRequired
	}

	if len(data.To) == 0 {
		return registry.ErrAtLeastOneRecipient
	}

	for _, to := range data.To {
		if err := ValidateEmail(to); err != nil {
			return registry.ErrInvalidToEmail(to)
		}
	}

	if err := ValidateEmail(data.From); err != nil {
		return registry.ErrInvalidFromEmail
	}

	if data.Subject == "" {
		return registry.ErrSubjectRequired
	}

	if data.HTML == "" {
		return registry.ErrHTMLContentRequired
	}

	if data.ReplyTo != "" {
		if err := ValidateEmail(data.ReplyTo); err != nil {
			return registry.ErrInvalidReplyToEmail
		}
	}

	for _, cc := range data.CC {
		if err := ValidateEmail(cc); err != nil {
			return registry.ErrInvalidCCEmail(cc)
		}
	}

	for _, bcc := range data.BCC {
		if err := ValidateEmail(bcc); err != nil {
			return registry.ErrInvalidBCCEmail(bcc)
		}
	}

	return nil
}
