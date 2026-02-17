package strategies

import (
	"file-service/pkg/mailer/providers"
	"file-service/pkg/mailer/registry"
	"fmt"
	"strings"
)

type FailoverStrategy struct{}

func (s *FailoverStrategy) Send(emailData *providers.EmailData, providerList []providers.EmailProvider) (*providers.EmailResult, error) {
	if len(providerList) == 0 {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrNoProvidersConfigured.Error(),
			Provider: registry.ProviderLabelNone,
		}, registry.ErrNoProvidersConfigured
	}

	var errorMessages []string

	for _, provider := range providerList {
		if provider == nil {
			errorMessages = append(errorMessages, fmt.Sprintf(registry.MsgProviderErrorFmt, registry.UnknownProviderName, registry.ErrProviderCannotBeNil.Error()))
			continue
		}

		result, err := provider.Send(emailData)

		if result != nil && result.Success {
			return result, nil
		}

		errorText := ""
		if result != nil && result.Error != "" {
			errorText = result.Error
		} else if err != nil {
			errorText = err.Error()
		} else {
			errorText = registry.StrategySendFailedText
		}

		errorMsg := fmt.Sprintf(registry.MsgProviderErrorFmt, provider.GetName(), errorText)
		errorMessages = append(errorMessages, errorMsg)
	}

	return &providers.EmailResult{
		Success:  false,
		Error:    fmt.Sprintf(registry.MsgProviderErrorFmt, registry.ErrAllProvidersFailed.Error(), strings.Join(errorMessages, registry.MessageSeparator)),
		Provider: registry.ProviderLabelFailover,
	}, registry.ErrAllProvidersFailed
}
