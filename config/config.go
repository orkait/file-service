package config

import (
	"fmt"
	"log"
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
}

func LoadConfig() (*Config, error) {
	// Create a new Config instance
	config := &Config{}

	// Retrieve and assign the values from environment variables
	config.BucketName = os.Getenv("BUCKET_NAME")
	config.Region = os.Getenv("REGION")
	config.DownloadURLTimeLimit, _ = strconv.Atoi(os.Getenv("DOWNLOAD_URL_TIME_LIMIT"))
	config.PaginationPageSize, _ = strconv.Atoi(os.Getenv("PAGINATION_PAGE_SIZE"))
	config.AwsAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	config.AwsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")

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

	// AWS keys are optional — ECS Task Role provides credentials automatically
	if config.AwsAccessKeyID == "" || config.AwsSecretAccessKey == "" {
		log.Println("WARNING: AWS credentials not provided via env vars. Using SDK default credential chain (IAM role/instance profile).")
	}

	return config, nil
}
