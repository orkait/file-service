package s3

import (
	"context"
	"file-service/internal/infra/cache"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	MaxBatchSize           = 100
	DefaultMaxWorkers      = 10
	ErrorUploadCancelled   = "upload cancelled"
	ErrorDownloadCancelled = "download cancelled"
	ErrorMaxBatchExceeded  = "maximum %d files allowed per batch"
	ErrorNoFilesProvided   = "no files provided. Use 'files' as the form-data field name"
	ErrorNoPathsProvided   = "no paths provided"
)

// CreateFolder creates a folder (empty object) in S3
func (c *Client) CreateFolder(folderPath string) error {
	if folderPath != "" && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(folderPath),
	}

	_, err := c.svc.PutObject(input)
	return err
}

// UploadFile uploads a file to S3
func (c *Client) UploadFile(src io.Reader, objectKey string) error {
	_, err := c.svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectKey),
		Body:   aws.ReadSeekCloser(src),
	})
	return err
}

// ListFiles lists all objects within a folder in S3
func (c *Client) ListFiles(folderPath string, nextPageToken string, pageSize int, isFolder bool, urlCache *cache.URLCache) (*ListFilesResponse, error) {
	if (folderPath != "") && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucketName),
		Prefix:    aws.String(folderPath),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(int64(pageSize + 1)),
	}

	if nextPageToken != "" {
		input.ContinuationToken = aws.String(nextPageToken)
	}

	resp, err := c.svc.ListObjectsV2(input)
	if err != nil {
		return nil, err
	}

	var objects []ObjectDetails

	for _, obj := range resp.CommonPrefixes {
		objects = append(objects, ObjectDetails{
			Name:         *obj.Prefix,
			IsFolder:     true,
			Size:         0,
			LastModified: time.Now().UTC().Truncate(time.Second),
		})
	}

	var fileCount int32 = 0

	if !isFolder {
		for _, obj := range resp.Contents {
			if *obj.Key == folderPath {
				continue
			}

			fileCount++
			objects = append(objects, ObjectDetails{
				Name:         *obj.Key,
				IsFolder:     *obj.Size == 0,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})

			downloadURL, err := c.GenerateDownloadLink(*obj.Key, urlCache)
			if err != nil {
				return nil, err
			}

			objects[len(objects)-1].DownloadLink = downloadURL
		}
	}

	nextToken := ""
	if resp.NextContinuationToken != nil {
		nextToken = *resp.NextContinuationToken
	}

	response := &ListFilesResponse{
		Files:               &objects,
		NextPageToken:       nextToken,
		IsLastPage:          !*resp.IsTruncated,
		NoOfRecordsReturned: int32(len(objects)),
		FilesCount:          fileCount,
		FoldersCount:        int32(len(resp.CommonPrefixes)),
	}

	return response, nil
}

// GenerateDownloadLink generates a presigned download URL
func (c *Client) GenerateDownloadLink(objectKey string, urlCache *cache.URLCache) (string, error) {
	url, found := urlCache.Get(objectKey)
	if found {
		return url, nil
	}

	expiryTime := 15 * time.Minute

	req, _ := c.svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket:              aws.String(c.bucketName),
		Key:                 aws.String(objectKey),
		ResponseContentType: aws.String("image/png"),
	})

	downloadURL, err := req.Presign(expiryTime)
	if err != nil {
		return "", err
	}

	urlCache.Set(objectKey, downloadURL, time.Now().Add(expiryTime))

	return downloadURL, nil
}

// DeleteObject deletes an object from S3
func (c *Client) DeleteObject(objectKey string) error {
	_, err := c.svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectKey),
	})
	return err
}

// DeleteFolder deletes a folder and its contents recursively
func (c *Client) DeleteFolder(folderPath string) error {
	if folderPath != "" && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	allObjects := []ObjectDetails{}

	resp, err := c.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(c.bucketName),
		Prefix:  aws.String(folderPath),
		MaxKeys: aws.Int64(2),
	})

	if err != nil {
		return err
	}

	for _, obj := range resp.Contents {
		if *obj.Key == folderPath {
			continue
		}

		allObjects = append(allObjects, ObjectDetails{
			Name:         *obj.Key,
			IsFolder:     *obj.Size == 0,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
		})
	}

	nextToken := resp.NextContinuationToken

	for nextToken != nil {
		curr, err := c.svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(c.bucketName),
			Prefix:            aws.String(folderPath),
			MaxKeys:           aws.Int64(1000),
			ContinuationToken: nextToken,
		})

		if err != nil {
			return err
		}

		for _, obj := range curr.Contents {
			if *obj.Key == folderPath {
				continue
			}

			allObjects = append(allObjects, ObjectDetails{
				Name:         *obj.Key,
				IsFolder:     *obj.Size == 0,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})

			nextToken = curr.NextContinuationToken
			if nextToken == nil {
				break
			}
		}
	}

	for _, obj := range allObjects {
		if err := c.DeleteObject(obj.Name); err != nil {
			return err
		}
	}

	return c.DeleteObject(folderPath)
}

// ListAllFolders lists all folders within a folder
func (c *Client) ListAllFolders(folderPath string) []ObjectDetails {
	if folderPath != "" && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	allObjects := []ObjectDetails{}

	resp, err := c.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(c.bucketName),
		Prefix:  aws.String(folderPath),
		MaxKeys: aws.Int64(1000),
	})

	if err != nil {
		return allObjects
	}

	for _, obj := range resp.Contents {
		if *obj.Key == folderPath {
			continue
		}

		if *obj.Size == 0 {
			allObjects = append(allObjects, ObjectDetails{
				Name:         *obj.Key,
				IsFolder:     true,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})
		}
	}

	nextToken := resp.NextContinuationToken

	for nextToken != nil {
		curr, _ := c.svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(c.bucketName),
			Prefix:            aws.String(folderPath),
			MaxKeys:           aws.Int64(1000),
			ContinuationToken: nextToken,
		})

		for _, obj := range curr.Contents {
			if *obj.Key == folderPath {
				continue
			}

			if *obj.Size == 0 {
				allObjects = append(allObjects, ObjectDetails{
					Name:         *obj.Key,
					IsFolder:     true,
					Size:         *obj.Size,
					LastModified: *obj.LastModified,
				})
			}

			nextToken = curr.NextContinuationToken
			if nextToken == nil {
				break
			}
		}
	}

	return allObjects
}

// BuildObjectKey constructs the S3 object key
func BuildObjectKey(folderPath, filename string) string {
	if folderPath == "" {
		return filename
	}
	if strings.HasSuffix(folderPath, "/") {
		return folderPath + filename
	}
	return folderPath + "/" + filename
}

// BatchUploadFiles uploads multiple files concurrently
func (c *Client) BatchUploadFiles(ctx context.Context, files []FileUploadInput, maxWorkers int) *BatchUploadResponse {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}
	if maxWorkers > len(files) {
		maxWorkers = len(files)
	}

	jobs := make(chan FileUploadInput, len(files))
	results := make(chan BatchUploadResult, len(files))

	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				select {
				case <-ctx.Done():
					results <- BatchUploadResult{
						FileName: file.FileName,
						Error:    ErrorUploadCancelled,
						Success:  false,
					}
					continue
				default:
				}

				func() {
					defer func() {
						if r := recover(); r != nil {
							results <- BatchUploadResult{
								FileName: file.FileName,
								Error:    fmt.Sprintf("panic: %v", r),
								Success:  false,
							}
						}
					}()

					err := c.UploadFile(file.Reader, file.ObjectKey)
					if err != nil {
						results <- BatchUploadResult{
							FileName: file.FileName,
							Error:    err.Error(),
							Success:  false,
						}
					} else {
						results <- BatchUploadResult{
							FileName:  file.FileName,
							ObjectKey: file.ObjectKey,
							Success:   true,
						}
					}
				}()
			}
		}()
	}

	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	response := &BatchUploadResponse{
		Uploaded: []BatchUploadResult{},
		Failed:   []BatchUploadResult{},
	}

	for res := range results {
		if res.Success {
			response.Uploaded = append(response.Uploaded, res)
			response.TotalUploaded++
		} else {
			response.Failed = append(response.Failed, res)
			response.TotalFailed++
		}
	}

	return response
}

// BatchGenerateDownloadLinks generates presigned URLs for multiple files concurrently
func (c *Client) BatchGenerateDownloadLinks(ctx context.Context, paths []string, urlCache *cache.URLCache, maxWorkers int) *BatchDownloadResponse {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}
	if maxWorkers > len(paths) {
		maxWorkers = len(paths)
	}

	jobs := make(chan string, len(paths))
	results := make(chan BatchDownloadResult, len(paths))

	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				select {
				case <-ctx.Done():
					results <- BatchDownloadResult{
						Path:     path,
						FileName: filepath.Base(path),
						Error:    ErrorDownloadCancelled,
						Success:  false,
					}
					continue
				default:
				}

				func() {
					defer func() {
						if r := recover(); r != nil {
							results <- BatchDownloadResult{
								Path:     path,
								FileName: filepath.Base(path),
								Error:    fmt.Sprintf("panic: %v", r),
								Success:  false,
							}
						}
					}()

					url, err := c.GenerateDownloadLink(path, urlCache)
					fileName := filepath.Base(path)
					if err != nil {
						results <- BatchDownloadResult{
							Path:     path,
							FileName: fileName,
							Error:    err.Error(),
							Success:  false,
						}
					} else {
						results <- BatchDownloadResult{
							Path:     path,
							FileName: fileName,
							URL:      url,
							Success:  true,
						}
					}
				}()
			}
		}()
	}

	go func() {
		for _, path := range paths {
			jobs <- path
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	response := &BatchDownloadResponse{
		Files: []BatchDownloadResult{},
	}

	for res := range results {
		if res.Success {
			response.TotalSuccess++
		} else {
			response.TotalFailed++
		}
		response.Files = append(response.Files, res)
	}

	return response
}
