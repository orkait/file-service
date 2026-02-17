package strategies

import "file-service/pkg/mailer/providers"

type EmailStrategy interface {
	Send(emailData *providers.EmailData, providerList []providers.EmailProvider) (*providers.EmailResult, error)
}
