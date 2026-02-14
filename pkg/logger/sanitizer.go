package logger

import (
	"regexp"
	"strings"
)

// Sensitive field patterns to filter from logs
var (
	passwordPattern = regexp.MustCompile(`(?i)(password|passwd|pwd)[\s:=]+[^\s]+`)
	tokenPattern    = regexp.MustCompile(`(?i)(token|jwt|bearer)[\s:=]+[^\s]+`)
	apiKeyPattern   = regexp.MustCompile(`(?i)(api[_-]?key|apikey)[\s:=]+[^\s]+`)
	secretPattern   = regexp.MustCompile(`(?i)(secret|private[_-]?key)[\s:=]+[^\s]+`)
	emailPattern    = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
)

const redactedPlaceholder = "[REDACTED]"

// SanitizeLogMessage removes sensitive information from log messages
func SanitizeLogMessage(message string) string {
	// Redact passwords
	message = passwordPattern.ReplaceAllString(message, "${1}="+redactedPlaceholder)

	// Redact tokens
	message = tokenPattern.ReplaceAllString(message, "${1}="+redactedPlaceholder)

	// Redact API keys
	message = apiKeyPattern.ReplaceAllString(message, "${1}="+redactedPlaceholder)

	// Redact secrets
	message = secretPattern.ReplaceAllString(message, "${1}="+redactedPlaceholder)

	return message
}

// SanitizeMap removes sensitive keys from a map
func SanitizeMap(data map[string]interface{}) map[string]interface{} {
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"token", "jwt", "bearer",
		"api_key", "apikey", "api-key",
		"secret", "private_key", "private-key",
		"password_hash", "passwordhash",
	}

	sanitized := make(map[string]interface{}, len(data))
	for k, v := range data {
		lowerKey := strings.ToLower(k)
		isSensitive := false

		for _, sensitiveKey := range sensitiveKeys {
			if strings.Contains(lowerKey, sensitiveKey) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[k] = redactedPlaceholder
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}
