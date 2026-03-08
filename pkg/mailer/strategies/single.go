package strategies

import (
	"file-service/pkg/mailer/providers"
	"file-service/pkg/mailer/registry"
)

type SingleProviderStrategy struct{}

func (s *SingleProviderStrategy) Send(emailData *providers.EmailData, providerList []providers.EmailProvider) (*providers.EmailResult, error) {
	if len(providerList) == 0 {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrNoProvidersConfigured.Error(),
			Provider: registry.ProviderLabelNone,
		}, registry.ErrNoProvidersConfigured
	}

	if providerList[0] == nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrNoProvidersConfigured.Error(),
			Provider: registry.ProviderLabelNone,
		}, registry.ErrNoProvidersConfigured
	}

	return providerList[0].Send(emailData)
}
