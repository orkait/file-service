package registry

import (
	"errors"
	"fmt"
)

var (
	ErrAtLeastOneProviderRequired = errors.New("at least one provider is required")
	ErrProviderCannotBeNil        = errors.New("provider cannot be nil")
	ErrInvalidDefaultFromEmail    = errors.New("invalid default from email")
	ErrEmailDataRequired          = errors.New("email data is required")
	ErrEmailTemplateRequired      = errors.New("email template is required")
	ErrEmailServiceRequired       = errors.New("email service is required")
	ErrTemplateContextRequired    = errors.New("template context is required")
	ErrNoProvidersConfigured      = errors.New("no email providers configured")
	ErrAllProvidersFailed         = errors.New("all providers failed")
	ErrAllProvidersExhausted      = errors.New("all providers exhausted or failed")
	ErrAPIKeyRequired             = errors.New("api key is required")
	ErrAtLeastOneRecipient        = errors.New("at least one recipient required")
	ErrInvalidFromEmail           = errors.New("invalid 'from' email")
	ErrSubjectRequired            = errors.New("subject is required")
	ErrHTMLContentRequired        = errors.New("html content is required")
	ErrInvalidReplyToEmail        = errors.New("invalid 'replyTo' email")
	ErrCompanyRequired            = errors.New("company is required")
	ErrVerificationURLRequired    = errors.New("verification URL is required")
	ErrResetURLRequired           = errors.New("reset URL is required")
	ErrVerificationURLAbsolute    = errors.New("verification URL must be a valid absolute URL")
	ErrResetURLAbsolute           = errors.New("reset URL must be a valid absolute URL")
	ErrVerificationURLScheme      = errors.New("verification URL must use http or https")
	ErrResetURLScheme             = errors.New("reset URL must use http or https")
)

func ErrInvalidToEmail(email string) error {
	return fmt.Errorf("invalid 'to' email: %s", email)
}

func ErrInvalidCCEmail(email string) error {
	return fmt.Errorf("invalid 'cc' email: %s", email)
}

func ErrInvalidBCCEmail(email string) error {
	return fmt.Errorf("invalid 'bcc' email: %s", email)
}

func ErrInvalidTemplateContextType(name string) error {
	return fmt.Errorf("invalid template context type for %q", name)
}

func ErrProviderNilResult(name string) error {
	return fmt.Errorf("provider %s returned nil result", name)
}

func ErrAPIStatus(statusCode int) error {
	return fmt.Errorf("API error: %d", statusCode)
}
