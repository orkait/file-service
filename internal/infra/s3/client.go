package s3

import (
	"file-service/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Client represents the S3 client wrapper
type Client struct {
	bucketName string
	svc        *s3.S3
}

// NewClient creates a new S3 client instance
func NewClient(config *config.Config) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(
			config.AwsAccessKeyID,
			config.AwsSecretAccessKey,
			"",
		),
	})

	if err != nil {
		return nil, err
	}

	svc := s3.New(sess)

	return &Client{
		bucketName: config.BucketName,
		svc:        svc,
	}, nil
}
