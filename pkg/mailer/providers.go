package mailer

import (
	"file-service/pkg/mailer/providers"
	"file-service/pkg/mailer/templates"
)

func NewResendProvider(config providers.ResendConfig) *providers.ResendProvider {
	return providers.NewResendProvider(config)
}

func NewSendGridProvider(config providers.SendGridConfig) *providers.SendGridProvider {
	return providers.NewSendGridProvider(config)
}

func NewTemplate(name string, htmlTmpl string, textTmpl string) (*templates.TypedTemplate[templates.TemplateContext], error) {
	return templates.NewRawTemplate(name, htmlTmpl, textTmpl)
}

func VerifyEmailTemplate() (*templates.TypedTemplate[templates.VerifyEmailContext], error) {
	return templates.VerifyEmailTemplate()
}

func PasswordResetTemplate() (*templates.TypedTemplate[templates.PasswordResetContext], error) {
	return templates.PasswordResetTemplate()
}
