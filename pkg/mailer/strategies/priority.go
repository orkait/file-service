package strategies

import (
	"file-service/pkg/mailer/providers"
	"file-service/pkg/mailer/registry"
	"math"
	"sync"
)

type PriorityStrategy struct {
	providerUsage  map[string]int
	providerLimits map[string]int
	mu             sync.Mutex
}

func NewPriorityStrategy(limits map[string]int) *PriorityStrategy {
	if limits == nil {
		limits = make(map[string]int)
	}
	limitCopy := make(map[string]int, len(limits))
	for k, v := range limits {
		limitCopy[k] = v
	}

	return &PriorityStrategy{
		providerUsage:  make(map[string]int),
		providerLimits: limitCopy,
	}
}

func (s *PriorityStrategy) Send(emailData *providers.EmailData, providerList []providers.EmailProvider) (*providers.EmailResult, error) {
	if len(providerList) == 0 {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrNoProvidersConfigured.Error(),
			Provider: registry.ProviderLabelNone,
		}, registry.ErrNoProvidersConfigured
	}

	for _, provider := range providerList {
		if provider == nil {
			continue
		}

		providerName := provider.GetName()
		s.mu.Lock()
		usage := s.providerUsage[providerName]
		limit, hasLimit := s.providerLimits[providerName]

		if !hasLimit {
			limit = math.MaxInt
		}
		if usage >= limit {
			s.mu.Unlock()
			continue
		}
		// Reserve capacity before sending so concurrent calls cannot exceed limits.
		s.providerUsage[providerName] = usage + 1
		s.mu.Unlock()

		result, err := provider.Send(emailData)
		if result != nil && result.Success {
			return result, nil
		}
		_ = err

		s.mu.Lock()
		currentUsage := s.providerUsage[providerName]
		if currentUsage <= 1 {
			delete(s.providerUsage, providerName)
		} else {
			s.providerUsage[providerName] = currentUsage - 1
		}
		s.mu.Unlock()
	}

	return &providers.EmailResult{
		Success:  false,
		Error:    registry.ErrAllProvidersExhausted.Error(),
		Provider: registry.ProviderLabelPriority,
	}, registry.ErrAllProvidersExhausted
}

func (s *PriorityStrategy) ResetUsage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerUsage = make(map[string]int)
}

func (s *PriorityStrategy) GetUsage() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	usage := make(map[string]int)
	for k, v := range s.providerUsage {
		usage[k] = v
	}
	return usage
}
