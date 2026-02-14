package config

import "fmt"

const (
	errRequiredEnvNotSetFmt = "required environment variable %s is not set"
)

type messageBuilders struct {
	requiredEnvNotSet func(string) string
}

func newMessageBuilders() messageBuilders {
	return messageBuilders{
		requiredEnvNotSet: func(key string) string {
			return fmt.Sprintf(errRequiredEnvNotSetFmt, key)
		},
	}
}

var messages = newMessageBuilders()
