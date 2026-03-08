package mailer

import (
	"file-service/pkg/mailer/providers"
	"file-service/pkg/mailer/registry"
	"file-service/pkg/mailer/strategies"
	"file-service/pkg/mailer/templates"
	"sync"
)

type EmailService struct {
	providers   []providers.EmailProvider
	strategy    strategies.EmailStrategy
	defaultFrom string
	mu          sync.RWMutex
}

type EmailServiceConfig struct {
	Providers   []providers.EmailProvider
	Strategy    strategies.EmailStrategy
	DefaultFrom string
}

func NewEmailService(config EmailServiceConfig) (*EmailService, error) {
	if len(config.Providers) == 0 {
		return nil, registry.ErrAtLeastOneProviderRequired
	}
	providerList := make([]providers.EmailProvider, len(config.Providers))
	copy(providerList, config.Providers)

	for _, provider := range providerList {
		if provider == nil {
			return nil, registry.ErrProviderCannotBeNil
		}
	}

	strategy := config.Strategy
	if strategy == nil {
		strategy = &strategies.SingleProviderStrategy{}
	}

	if config.DefaultFrom != "" {
		if err := ValidateEmail(config.DefaultFrom); err != nil {
			return nil, registry.ErrInvalidDefaultFromEmail
		}
	}

	return &EmailService{
		providers:   providerList,
		strategy:    strategy,
		defaultFrom: config.DefaultFrom,
	}, nil
}

func (s *EmailService) Send(emailData *providers.EmailData) (*providers.EmailResult, error) {
	if emailData == nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrEmailDataRequired.Error(),
			Provider: registry.ProviderLabelValidation,
		}, registry.ErrEmailDataRequired
	}

	s.mu.RLock()
	defaultFrom := s.defaultFrom
	strategy := s.strategy
	providerList := make([]providers.EmailProvider, len(s.providers))
	copy(providerList, s.providers)
	s.mu.RUnlock()

	data := cloneEmailData(emailData)
	if data.From == "" && defaultFrom != "" {
		data.From = defaultFrom
	}

	if err := ValidateEmailData(data); err != nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    err.Error(),
			Provider: registry.ProviderLabelValidation,
		}, err
	}

	return strategy.Send(data, providerList)
}

func (s *EmailService) SendWithTemplate(template templates.EmailTemplate, context any, emailData *providers.EmailData) (*providers.EmailResult, error) {
	if template == nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrEmailTemplateRequired.Error(),
			Provider: registry.ProviderLabelTemplate,
		}, registry.ErrEmailTemplateRequired
	}

	if emailData == nil {
		emailData = &providers.EmailData{}
	}

	html, text, err := template.RenderAny(context)
	if err != nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    err.Error(),
			Provider: registry.ProviderLabelTemplate,
		}, err
	}

	return s.sendWithRenderedContent(html, text, emailData)
}

func SendWithTypedTemplate[T any](service *EmailService, template *templates.TypedTemplate[T], context T, emailData *providers.EmailData) (*providers.EmailResult, error) {
	if service == nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrEmailServiceRequired.Error(),
			Provider: registry.ProviderLabelTemplate,
		}, registry.ErrEmailServiceRequired
	}
	if emailData == nil {
		emailData = &providers.EmailData{}
	}
	if template == nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    registry.ErrEmailTemplateRequired.Error(),
			Provider: registry.ProviderLabelTemplate,
		}, registry.ErrEmailTemplateRequired
	}

	html, text, err := template.Render(context)
	if err != nil {
		return &providers.EmailResult{
			Success:  false,
			Error:    err.Error(),
			Provider: registry.ProviderLabelTemplate,
		}, err
	}

	return service.sendWithRenderedContent(html, text, emailData)
}

func (s *EmailService) AddProvider(provider providers.EmailProvider) {
	if provider == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = append(s.providers, provider)
}

func (s *EmailService) RemoveProvider(providerName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]providers.EmailProvider, 0)
	for _, p := range s.providers {
		if p == nil {
			continue
		}
		if p.GetName() != providerName {
			filtered = append(filtered, p)
		}
	}
	s.providers = filtered
}

func (s *EmailService) GetProviders() []providers.EmailProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providerList := make([]providers.EmailProvider, len(s.providers))
	copy(providerList, s.providers)
	return providerList
}

func (s *EmailService) SetStrategy(strategy strategies.EmailStrategy) {
	if strategy == nil {
		strategy = &strategies.SingleProviderStrategy{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategy = strategy
}

func (s *EmailService) VerifyProviders() (map[string]bool, error) {
	s.mu.RLock()
	providerList := make([]providers.EmailProvider, len(s.providers))
	copy(providerList, s.providers)
	s.mu.RUnlock()

	results := make(map[string]bool)

	for _, provider := range providerList {
		if provider == nil {
			continue
		}
		verified, _ := provider.Verify()
		results[provider.GetName()] = verified
	}

	return results, nil
}

func cloneEmailData(emailData *providers.EmailData) *providers.EmailData {
	clone := *emailData

	if emailData.To != nil {
		clone.To = append([]string(nil), emailData.To...)
	}
	if emailData.CC != nil {
		clone.CC = append([]string(nil), emailData.CC...)
	}
	if emailData.BCC != nil {
		clone.BCC = append([]string(nil), emailData.BCC...)
	}

	return &clone
}

func (s *EmailService) sendWithRenderedContent(html string, text string, emailData *providers.EmailData) (*providers.EmailResult, error) {
	data := cloneEmailData(emailData)
	data.HTML = html
	data.Text = text

	return s.Send(data)
}
