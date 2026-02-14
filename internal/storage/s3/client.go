package s3

import (
	"context"
	"file-service/internal/config"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	emptyAWSSessionToken                     = ""
	defaultS3Region                          = "us-east-1"
	deleteFolderBatchSize                    = 1000
	pathSeparator                            = '/'
	errFailedCreateAWSSessionFmt             = "failed to create AWS session: %w"
	errFailedGeneratePresignedUploadURLFmt   = "failed to generate presigned upload URL: %w"
	errFailedGeneratePresignedDownloadURLFmt = "failed to generate presigned download URL: %w"
	errFailedDeleteObjectFmt                 = "failed to delete object: %w"
	errFailedCreateBucketFmt                 = "failed to create bucket: %w"
	errFailedWaitBucketExistsFmt             = "failed to wait for bucket to exist: %w"
	errFailedDeleteBucketFmt                 = "failed to delete bucket: %w"
	errFailedListObjectsFmt                  = "failed to list objects: %w"
	errFailedDeleteFolderObjectsFmt          = "failed to delete folder objects: %w"
)

type Client struct {
	svc                *s3.S3
	presignedURLExpiry time.Duration
}

func NewClient(cfg *config.AWSConfig, presignedURLExpiry time.Duration) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			emptyAWSSessionToken,
		),
	})

	if err != nil {
		return nil, fmt.Errorf(errFailedCreateAWSSessionFmt, err)
	}

	return &Client{
		svc:                s3.New(sess),
		presignedURLExpiry: presignedURLExpiry,
	}, nil
}

func (c *Client) GeneratePresignedUploadURL(ctx context.Context, bucketName, objectKey string, contentType string) (string, error) {
	req, _ := c.svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	})
	req.SetContext(ctx)

	url, err := req.Presign(c.presignedURLExpiry)
	if err != nil {
		return "", fmt.Errorf(errFailedGeneratePresignedUploadURLFmt, err)
	}

	return url, nil
}

func (c *Client) GeneratePresignedDownloadURL(ctx context.Context, bucketName, objectKey string) (string, error) {
	req, _ := c.svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	req.SetContext(ctx)

	url, err := req.Presign(c.presignedURLExpiry)
	if err != nil {
		return "", fmt.Errorf(errFailedGeneratePresignedDownloadURLFmt, err)
	}

	return url, nil
}

func (c *Client) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	_, err := c.svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return fmt.Errorf(errFailedDeleteObjectFmt, err)
	}

	return nil
}

func (c *Client) CreateBucket(ctx context.Context, bucketName, region string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	if region != defaultS3Region {
		input.CreateBucketConfiguration = &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		}
	}

	_, err := c.svc.CreateBucketWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf(errFailedCreateBucketFmt, err)
	}

	if err := c.svc.WaitUntilBucketExistsWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		return fmt.Errorf(errFailedWaitBucketExistsFmt, err)
	}

	return nil
}

func (c *Client) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := c.svc.DeleteBucketWithContext(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		return fmt.Errorf(errFailedDeleteBucketFmt, err)
	}

	return nil
}

func (c *Client) ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) (*s3.ListObjectsV2Output, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(int64(maxKeys)),
	}

	result, err := c.svc.ListObjectsV2WithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf(errFailedListObjectsFmt, err)
	}

	return result, nil
}

func (c *Client) DeleteFolder(ctx context.Context, bucketName, prefix string) error {
	result, err := c.ListObjects(ctx, bucketName, prefix, deleteFolderBatchSize)
	if err != nil {
		return err
	}

	if len(result.Contents) == 0 {
		return nil
	}

	var objectsToDelete []*s3.ObjectIdentifier
	for _, obj := range result.Contents {
		objectsToDelete = append(objectsToDelete, &s3.ObjectIdentifier{
			Key: obj.Key,
		})
	}

	_, err = c.svc.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &s3.Delete{
			Objects: objectsToDelete,
			Quiet:   aws.Bool(true),
		},
	})

	if err != nil {
		return fmt.Errorf(errFailedDeleteFolderObjectsFmt, err)
	}

	if aws.BoolValue(result.IsTruncated) {
		return c.DeleteFolder(ctx, bucketName, prefix)
	}

	return nil
}

func BuildObjectKey(folderPath, filename string) string {
	if folderPath == "" {
		return filename
	}

	if folderPath[len(folderPath)-1] != pathSeparator {
		folderPath += "/"
	}

	return folderPath + filename
}
