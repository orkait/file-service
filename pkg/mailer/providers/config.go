package providers

import "file-service/pkg/mailer/registry"

func isHTTPSuccess(statusCode int) bool {
	return statusCode >= registry.HTTPStatusSuccessMin && statusCode < registry.HTTPStatusSuccessMax
}
