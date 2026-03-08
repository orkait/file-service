package providers

import (
	"bytes"
	"encoding/json"
	"file-service/pkg/mailer/registry"
	"fmt"
	"io"
	"net/http"
)

type SendGridProvider struct {
	BaseProvider
	APIURL string
}

type SendGridConfig struct {
	APIKey string
	APIURL string
}

func NewSendGridProvider(config SendGridConfig) *SendGridProvider {
	apiURL := config.APIURL
	if apiURL == "" {
		apiURL = registry.SendGridAPIURL
	}

	return &SendGridProvider{
		BaseProvider: BaseProvider{
			APIKey:       config.APIKey,
			ProviderName: registry.ProviderSendGrid,
		},
		APIURL: apiURL,
	}
}

func (p *SendGridProvider) Send(emailData *EmailData) (*EmailResult, error) {
	if p.APIKey == "" {
		return &EmailResult{
			Success:  false,
			Error:    registry.ErrAPIKeyRequired.Error(),
			Provider: p.ProviderName,
		}, registry.ErrAPIKeyRequired
	}

	toList := make([]map[string]string, len(emailData.To))
	for i, email := range emailData.To {
		toList[i] = map[string]string{registry.JSONEmail: email}
	}

	personalization := map[string]interface{}{
		registry.JSONTo: toList,
	}

	if len(emailData.CC) > 0 {
		ccList := make([]map[string]string, len(emailData.CC))
		for i, email := range emailData.CC {
			ccList[i] = map[string]string{registry.JSONEmail: email}
		}
		personalization[registry.JSONCC] = ccList
	}

	if len(emailData.BCC) > 0 {
		bccList := make([]map[string]string, len(emailData.BCC))
		for i, email := range emailData.BCC {
			bccList[i] = map[string]string{registry.JSONEmail: email}
		}
		personalization[registry.JSONBCC] = bccList
	}

	content := []map[string]string{
		{registry.JSONType: registry.MIMETextHTML, registry.JSONValue: emailData.HTML},
	}

	if emailData.Text != "" {
		content = append(content, map[string]string{registry.JSONType: registry.MIMETextPlain, registry.JSONValue: emailData.Text})
	}

	payload := map[string]interface{}{
		registry.JSONPersonalizations: []map[string]interface{}{personalization},
		registry.JSONFrom:             map[string]string{registry.JSONEmail: emailData.From},
		registry.JSONSubject:          emailData.Subject,
		registry.JSONContent:          content,
	}

	if emailData.ReplyTo != "" {
		payload[registry.JSONReplyTo] = map[string]string{registry.JSONEmail: emailData.ReplyTo}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return &EmailResult{
			Success:  false,
			Error:    fmt.Sprintf(registry.MsgFailedMarshalPayloadFmt, err),
			Provider: p.ProviderName,
		}, err
	}

	req, err := http.NewRequest(http.MethodPost, p.APIURL+registry.PathSendGridMailSend, bytes.NewBuffer(jsonData))
	if err != nil {
		return &EmailResult{
			Success:  false,
			Error:    fmt.Sprintf(registry.MsgFailedCreateRequestFmt, err),
			Provider: p.ProviderName,
		}, err
	}

	req.Header.Set(registry.HeaderAuthorization, registry.AuthBearerPrefix+p.APIKey)
	req.Header.Set(registry.HeaderContentType, registry.MIMEApplicationJSON)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &EmailResult{
			Success:  false,
			Error:    fmt.Sprintf(registry.MsgRequestFailedFmt, err),
			Provider: p.ProviderName,
		}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if !isHTTPSuccess(resp.StatusCode) {
		return &EmailResult{
			Success:  false,
			Error:    fmt.Sprintf(registry.MsgSendGridAPIErrorFmt, resp.StatusCode, string(body)),
			Provider: p.ProviderName,
		}, registry.ErrAPIStatus(resp.StatusCode)
	}

	messageID := resp.Header.Get(registry.HeaderMessageID)

	return &EmailResult{
		Success:   true,
		MessageID: messageID,
		Provider:  p.ProviderName,
	}, nil
}

func (p *SendGridProvider) Verify() (bool, error) {
	if p.APIKey == "" {
		return false, registry.ErrAPIKeyRequired
	}

	req, err := http.NewRequest(http.MethodGet, p.APIURL+registry.PathSendGridScopes, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set(registry.HeaderAuthorization, registry.AuthBearerPrefix+p.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return isHTTPSuccess(resp.StatusCode), nil
}
