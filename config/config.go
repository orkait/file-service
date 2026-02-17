package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	BucketName           string `json:"bucketName"`
	Region               string `json:"region"`
	DownloadURLTimeLimit int    `json:"downloadURLTimeLimit"`
	PaginationPageSize   int    `json:"paginationPageSize"`
	AwsAccessKeyID       string `json:"awsAccessKeyId"`
	AwsSecretAccessKey   string `json:"awsSecretAccessKey"`
	DatabaseURL          string `json:"databaseUrl"`
	JWTSecret            string `json:"jwtSecret"`
	AppBaseURL           string `json:"appBaseUrl"`
	AppName              string `json:"appName"`
	MailProviders        string `json:"mailProviders"`
	ResendAPIKey         string `json:"resendApiKey"`
	ResendAPIURL         string `json:"resendApiUrl"`
	SendGridAPIKey       string `json:"sendGridApiKey"`
	SendGridAPIURL       string `json:"sendGridApiUrl"`
	MailFrom             string `json:"mailFrom"`
}

func LoadConfig() (*Config, error) {
	// Create a new Config instance
	config := &Config{}

	// Retrieve and assign the values from environment variables
	config.BucketName = os.Getenv("BUCKET_NAME")
	config.Region = os.Getenv("REGION")

	if val := os.Getenv("DOWNLOAD_URL_TIME_LIMIT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.DownloadURLTimeLimit = parsed
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Invalid DOWNLOAD_URL_TIME_LIMIT value '%s', using default\n", val)
		}
	}

	if val := os.Getenv("PAGINATION_PAGE_SIZE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.PaginationPageSize = parsed
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Invalid PAGINATION_PAGE_SIZE value '%s', using default\n", val)
		}
	}

	config.AwsAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	config.AwsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	config.DatabaseURL = os.Getenv("DATABASE_URL")
	config.JWTSecret = os.Getenv("JWT_SECRET")
	config.AppBaseURL = os.Getenv("APP_BASE_URL")
	config.AppName = os.Getenv("APP_NAME")
	config.MailProviders = os.Getenv("MAIL_PROVIDERS")
	config.ResendAPIKey = os.Getenv("RESEND_API_KEY")
	config.ResendAPIURL = os.Getenv("RESEND_API_URL")
	config.SendGridAPIKey = os.Getenv("SENDGRID_API_KEY")
	config.SendGridAPIURL = os.Getenv("SENDGRID_API_URL")
	config.MailFrom = os.Getenv("MAIL_FROM")

	if config.BucketName == "" {
		return nil, fmt.Errorf("BUCKET_NAME must be set")
	}

	if config.Region == "" {
		return nil, fmt.Errorf("REGION must be set")
	}

	if config.DownloadURLTimeLimit == 0 {
		config.DownloadURLTimeLimit = 15
	}

	if config.PaginationPageSize == 0 {
		config.PaginationPageSize = 100
	}

	if config.AwsAccessKeyID == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID must be set")
	}

	if config.AwsSecretAccessKey == "" {
		return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY must be set")
	}

	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must be set")
	}

	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set")
	}

	if config.AppBaseURL == "" {
		config.AppBaseURL = "http://localhost:3000"
	}

	if config.AppName == "" {
		config.AppName = "Orka File Service"
	}

	return config, nil
}
