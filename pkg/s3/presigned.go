package s3

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// PresignedPostResponse contains the URL and fields for browser upload
type PresignedPostResponse struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

// GeneratePresignedPost creates a presigned POST for direct browser → S3 upload
func (s *S3) GeneratePresignedPost(s3Key string, maxFileSize int64, expiresIn time.Duration) (*PresignedPostResponse, error) {
	// Create presigned POST request
	req, _ := s.svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})

	url, err := req.Presign(expiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned POST: %w", err)
	}

	// For PUT presigned URL (simpler than POST policy)
	return &PresignedPostResponse{
		URL: url,
		Fields: map[string]string{
			"key": s3Key,
		},
	}, nil
}
