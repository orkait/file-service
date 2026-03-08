package templates

import (
	"file-service/pkg/mailer/registry"
	"net/url"
	"strings"
)

type PasswordResetContext struct {
	Company     string
	UserName    string
	ResetURL    string
	ExpiryHours int
}

func PasswordResetTemplate() (*TypedTemplate[PasswordResetContext], error) {
	htmlTmpl := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Reset Your Password</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
	<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
		<h2>{{.Company}}</h2>
		<p>{{if .UserName}}Hi {{.UserName}},{{else}}Hi there,{{end}}</p>
		<p>We received a request to reset your password. Click the button below to create a new password:</p>
		<div style="text-align: center; margin: 30px 0;">
			<a href="{{.ResetURL}}" style="background-color: #dc3545; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
				Reset Password
			</a>
		</div>
		<p>Or copy and paste this link into your browser:</p>
		<p style="word-break: break-all; color: #dc3545;">{{.ResetURL}}</p>
		<p>This link will expire in {{if .ExpiryHours}}{{.ExpiryHours}}{{else}}1{{end}} hour(s).</p>
		<p>If you didn't request a password reset, you can safely ignore this email.</p>
	</div>
</body>
</html>
`

	textTmpl := `
Reset Your Password

{{if .UserName}}Hi {{.UserName}},{{else}}Hi there,{{end}}

We received a request to reset your password for your {{.Company}} account.

Visit this link to create a new password:
{{.ResetURL}}

This link will expire in {{if .ExpiryHours}}{{.ExpiryHours}}{{else}}1{{end}} hour(s).

If you didn't request a password reset, you can safely ignore this email.
`

	parser := func(context PasswordResetContext) (PasswordResetContext, error) {
		context.Company = strings.TrimSpace(context.Company)
		context.UserName = strings.TrimSpace(context.UserName)
		context.ResetURL = strings.TrimSpace(context.ResetURL)

		if context.Company == "" {
			return context, registry.ErrCompanyRequired
		}
		if context.ResetURL == "" {
			return context, registry.ErrResetURLRequired
		}

		parsed, err := url.Parse(context.ResetURL)
		if err != nil || !parsed.IsAbs() {
			return context, registry.ErrResetURLAbsolute
		}
		if parsed.Scheme != registry.URLSchemeHTTP && parsed.Scheme != registry.URLSchemeHTTPS {
			return context, registry.ErrResetURLScheme
		}

		if context.ExpiryHours <= 0 {
			context.ExpiryHours = registry.PasswordResetExpiryHours
		}

		return context, nil
	}

	return NewTemplate(registry.TemplateNamePasswordReset, htmlTmpl, textTmpl, parser)
}
