package providers

type EmailProvider interface {
	Send(emailData *EmailData) (*EmailResult, error)
	Verify() (bool, error)
	GetName() string
}

type BaseProvider struct {
	APIKey       string
	ProviderName string
}

func (p *BaseProvider) GetName() string {
	return p.ProviderName
}

type EmailData struct {
	To      []string
	From    string
	Subject string
	HTML    string
	Text    string
	ReplyTo string
	CC      []string
	BCC     []string
}

type EmailResult struct {
	Success   bool
	MessageID string
	Error     string
	Provider  string
}
