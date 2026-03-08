package providers

import (
	"bytes"
	"encoding/json"
	"file-service/pkg/mailer/registry"
	"fmt"
	"io"
	"net/http"
)

type ResendProvider struct {
	BaseProvider
	APIURL string
}

type ResendConfig struct {
	APIKey string
	APIURL string
}

func NewResendProvider(config ResendConfig) *ResendProvider {
	apiURL := config.APIURL
	if apiURL == "" {
		apiURL = registry.ResendAPIURL
	}

	return &ResendProvider{
		BaseProvider: BaseProvider{
			APIKey:       config.APIKey,
			ProviderName: registry.ProviderResend,
		},
		APIURL: apiURL,
	}
}

func (p *ResendProvider) Send(emailData *EmailData) (*EmailResult, error) {
	if p.APIKey == "" {
		return &EmailResult{
			Success:  false,
			Error:    registry.ErrAPIKeyRequired.Error(),
			Provider: p.ProviderName,
		}, registry.ErrAPIKeyRequired
	}

	payload := map[string]interface{}{
		registry.JSONFrom:    emailData.From,
		registry.JSONTo:      emailData.To,
		registry.JSONSubject: emailData.Subject,
		registry.JSONHTML:    emailData.HTML,
	}

	if emailData.Text != "" {
		payload[registry.JSONText] = emailData.Text
	}

	if emailData.ReplyTo != "" {
		payload[registry.JSONReplyTo] = emailData.ReplyTo
	}

	if len(emailData.CC) > 0 {
		payload[registry.JSONCC] = emailData.CC
	}

	if len(emailData.BCC) > 0 {
		payload[registry.JSONBCC] = emailData.BCC
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return &EmailResult{
			Success:  false,
			Error:    fmt.Sprintf(registry.MsgFailedMarshalPayloadFmt, err),
			Provider: p.ProviderName,
		}, err
	}

	req, err := http.NewRequest(http.MethodPost, p.APIURL+registry.PathResendEmails, bytes.NewBuffer(jsonData))
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
			Error:    fmt.Sprintf(registry.MsgResendAPIErrorFmt, resp.StatusCode, string(body)),
			Provider: p.ProviderName,
		}, registry.ErrAPIStatus(resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return &EmailResult{
			Success:  false,
			Error:    fmt.Sprintf(registry.MsgFailedParseResponseFmt, err),
			Provider: p.ProviderName,
		}, err
	}

	return &EmailResult{
		Success:   true,
		MessageID: result.ID,
		Provider:  p.ProviderName,
	}, nil
}

func (p *ResendProvider) Verify() (bool, error) {
	if p.APIKey == "" {
		return false, registry.ErrAPIKeyRequired
	}

	req, err := http.NewRequest(http.MethodGet, p.APIURL+registry.PathResendAPIKeys, nil)
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
