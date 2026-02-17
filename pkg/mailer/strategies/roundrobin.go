package strategies

import (
	"errors"
	"file-service/pkg/mailer/providers"
	"file-service/pkg/mailer/registry"
	"sync"
)

type RoundRobinStrategy struct {
	currentIndex int
	mu           sync.Mutex
}

func (s *RoundRobinStrategy) Send(emailData *providers.EmailData, providerList []providers.EmailProvider) (*providers.EmailResult, error) {
	if len(providerList) == 0 {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrNoProvidersConfigured.Error(),
			Provider: registry.ProviderLabelNone,
		}, registry.ErrNoProvidersConfigured
	}

	s.mu.Lock()
	startIndex := s.currentIndex % len(providerList)
	s.currentIndex = (startIndex + 1) % len(providerList)
	s.mu.Unlock()

	lastErr := registry.ErrNoProvidersConfigured
	lastProvider := registry.ProviderLabelNone
	for i := 0; i < len(providerList); i++ {
		idx := (startIndex + i) % len(providerList)
		provider := providerList[idx]
		if provider == nil {
			lastErr = registry.ErrNoProvidersConfigured
			lastProvider = registry.ProviderLabelNone
			continue
		}

		result, err := provider.Send(emailData)
		if result != nil && result.Success {
			return result, nil
		}

		lastProvider = provider.GetName()
		if err != nil {
			lastErr = err
			continue
		}
		if result != nil && result.Error != "" {
			lastErr = errors.New(result.Error)
			continue
		}
		lastErr = errors.New(registry.StrategySendFailedText)
	}

	return &providers.EmailResult{
		Success:  false,
		Error:    lastErr.Error(),
		Provider: lastProvider,
	}, lastErr
}
