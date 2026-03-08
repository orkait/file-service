package templates

import (
	"file-service/pkg/mailer/registry"
	"net/url"
	"strings"
)

type VerifyEmailContext struct {
	Company         string
	UserName        string
	VerificationURL string
	ExpiryHours     int
}

func VerifyEmailTemplate() (*TypedTemplate[VerifyEmailContext], error) {
	htmlTmpl := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Verify Your Email</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
	<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
		<h2>{{.Company}}</h2>
		<p>{{if .UserName}}Hi {{.UserName}},{{else}}Hi there,{{end}}</p>
		<p>Thank you for signing up! Please verify your email address by clicking the button below:</p>
		<div style="text-align: center; margin: 30px 0;">
			<a href="{{.VerificationURL}}" style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
				Verify Email
			</a>
		</div>
		<p>Or copy and paste this link into your browser:</p>
		<p style="word-break: break-all; color: #007bff;">{{.VerificationURL}}</p>
		<p>This link will expire in {{if .ExpiryHours}}{{.ExpiryHours}}{{else}}24{{end}} hours.</p>
		<p>If you didn't create an account, you can safely ignore this email.</p>
	</div>
</body>
</html>
`

	textTmpl := `
Verify Your Email

{{if .UserName}}Hi {{.UserName}},{{else}}Hi there,{{end}}

Thank you for signing up with {{.Company}}!

Please verify your email address by visiting this link:

{{.VerificationURL}}

This link will expire in {{if .ExpiryHours}}{{.ExpiryHours}}{{else}}24{{end}} hours.

If you didn't create an account, you can safely ignore this email.
`

	parser := func(context VerifyEmailContext) (VerifyEmailContext, error) {
		context.Company = strings.TrimSpace(context.Company)
		context.UserName = strings.TrimSpace(context.UserName)
		context.VerificationURL = strings.TrimSpace(context.VerificationURL)

		if context.Company == "" {
			return context, registry.ErrCompanyRequired
		}
		if context.VerificationURL == "" {
			return context, registry.ErrVerificationURLRequired
		}

		parsed, err := url.Parse(context.VerificationURL)
		if err != nil || !parsed.IsAbs() {
			return context, registry.ErrVerificationURLAbsolute
		}
		if parsed.Scheme != registry.URLSchemeHTTP && parsed.Scheme != registry.URLSchemeHTTPS {
			return context, registry.ErrVerificationURLScheme
		}

		if context.ExpiryHours <= 0 {
			context.ExpiryHours = registry.EmailVerificationExpiryHours
		}

		return context, nil
	}

	return NewTemplate(registry.TemplateNameVerifyEmail, htmlTmpl, textTmpl, parser)
}
