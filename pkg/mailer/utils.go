package mailer

import (
	"file-service/pkg/mailer/registry"
	"html"
	"net/url"
	"strings"
)

func EscapeHTML(text string) string {
	return html.EscapeString(text)
}

func SanitizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	if parsed.Scheme != registry.URLSchemeHTTP && parsed.Scheme != registry.URLSchemeHTTPS {
		return ""
	}

	return parsed.String()
}

func EnsureArray(value interface{}) []string {
	switch v := value.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return []string{}
	}
}

func IsNotEmpty(arr []interface{}) bool {
	return len(arr) > 0
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
