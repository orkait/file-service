package s3

import (
	"context"
	"file-service/config"
	"file-service/pkg/cache"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3 represents the Amazon S3 service.
type S3 struct {
	bucketName string
	svc        *s3.S3
}

// NewS3 creates a new S3 instance with the specified bucket name and AWS session.
func NewClient(config *config.Config) (*S3, error) {
	// Create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region), // Replace with your desired AWS region,
		Credentials: credentials.NewStaticCredentials(
			config.AwsAccessKeyID,     // Replace with your AWS access key ID
			config.AwsSecretAccessKey, // Replace with your AWS secret access key
			"",
		),
	})

	if err != nil {
		return nil, err
	}

	// Create an S3 service client
	svc := s3.New(sess)

	return &S3{
		bucketName: config.BucketName,
		svc:        svc,
	}, nil
}

// CreateFolder creates a folder (empty object) in the specified bucket and folder path
func (s *S3) CreateFolder(folderPath string) error {
	// Add a trailing slash to the folder path if not already present
	if folderPath != "" && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	// Create an empty object with the folder path as the key
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(folderPath),
	}

	_, err := s.svc.PutObject(input)
	if err != nil {
		return err
	}

	return nil
}

// UploadFile uploads a file to the S3 bucket.
func (s *S3) UploadFile(src io.Reader, objectKey string) error {
	// Upload the file to S3
	_, err := s.svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
		Body:   aws.ReadSeekCloser(src),
	})
	if err != nil {
		return err
	}

	return nil
}

// ListObjects lists all the objects within a folder in the S3 bucket.
func (s *S3) ListFiles(folderPath string, nextPageToken string, pageSize int, isFolder bool, cache *cache.URLCache) (*ListFilesResponse, error) {

	// If the folder path does not end with a slash, add it
	if (folderPath != "") && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucketName),
		Prefix:    aws.String(folderPath),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(int64(pageSize + 1)),
	}

	if nextPageToken != "" {
		input.ContinuationToken = aws.String(nextPageToken)
	}

	resp, err := s.svc.ListObjectsV2(input)

	// send all file details
	var objects []ObjectDetails

	for _, obj := range resp.CommonPrefixes {
		objects = append(objects, ObjectDetails{
			Name:         *obj.Prefix,
			IsFolder:     true,
			Size:         0,
			LastModified: time.Now().UTC().Truncate(time.Second),
		})
	}

	if err != nil {
		return nil, err
	}

	var fileCount int32 = 0

	if !isFolder {
		for _, obj := range resp.Contents {
			if *obj.Key == folderPath {
				continue // skip the folder itself
			}

			fileCount++
			objects = append(objects, ObjectDetails{
				Name:         *obj.Key,
				IsFolder:     *obj.Size == 0,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})

			// generate a signed download URL for the object
			downloadURL, err := s.GenerateDownloadLink(*obj.Key, cache)

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

func (s *S3) ListAllFiles(folderPath string, urlCache *cache.URLCache) (*ListFilesResponse, error) {
	objects, err := s.ListFiles(folderPath, "", 10, false, urlCache)
	if err != nil {
		return nil, err
	}

	nextToken := objects.NextPageToken
	var allObjects []ObjectDetails

	// check if next page token is present
	for nextToken != "" {
		temp, err := s.ListFiles(folderPath, nextToken, 10, false, urlCache)
		if err != nil {
			return nil, err
		}
		allObjects = append(allObjects, *temp.Files...)

		if temp.IsLastPage {
			nextToken = ""
		} else {
			nextToken = temp.NextPageToken
		}
	}

	// Helper function to recursively fetch objects from subfolders
	var listObjectsRecursively func(path string) error
	listObjectsRecursively = func(path string) error {
		objects, err := s.ListFiles(path, "", 10, false, urlCache)
		if err != nil {
			return err
		}

		nextToken := objects.NextPageToken

		// check if next page token is present
		for nextToken != "" {
			t, err := s.ListFiles(path, nextToken, 10, false, urlCache)
			if err != nil {
				return err
			}
			allObjects = append(allObjects, *t.Files...)

			if t.IsLastPage {
				nextToken = ""
			} else {
				nextToken = t.NextPageToken
			}
		}

		// Add the objects from the current folder to the result
		allObjects = append(allObjects, *objects.Files...)

		// Recursively fetch objects from subfolders
		for _, subfolder := range *objects.Files {
			if subfolder.IsFolder {
				err := listObjectsRecursively(subfolder.Name)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	// Recursively fetch objects from subfolders
	for _, folder := range *objects.Files {
		if folder.IsFolder {
			err := listObjectsRecursively(folder.Name)
			if err != nil {
				return nil, err
			}
		}
	}

	// Combine the initial folder's objects with the recursively fetched objects
	allObjects = append(*objects.Files, allObjects...)

	return &ListFilesResponse{
		Files:               &allObjects,
		NextPageToken:       objects.NextPageToken,
		IsLastPage:          objects.IsLastPage,
		NoOfRecordsReturned: int32(len(allObjects)),
		FilesCount:          objects.FilesCount,
		FoldersCount:        objects.FoldersCount,
	}, nil
}

// GetFile retrieves a file from the specified bucket and key in S3.
func (s *S3) GetFile(bucket, key string) (io.Reader, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.svc.GetObject(input)

	if err != nil {
		return nil, err
	}

	return result.Body, nil
}

// Function to generate a signed download URL for the object
func (s *S3) GenerateDownloadLink(objectKey string, cache *cache.URLCache) (string, error) {
	url, found := cache.Get(objectKey)

	// Check if the URL is already in the cache and valid
	if found {
		return url, nil
	}

	expiryTime := 15 * time.Minute

	req, _ := s.svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket:              aws.String(s.bucketName),
		Key:                 aws.String(objectKey),
		ResponseContentType: aws.String("image/png"),
	})

	downloadURL, err := req.Presign(expiryTime) // Set the validity period of the signed URL
	if err != nil {
		return "", err
	}

	// Cache the URL with its expiration time
	cache.Set(objectKey, downloadURL, time.Now().Add(expiryTime))

	return downloadURL, nil
}

// DeleteObject deletes an object from the S3 bucket.
func (s *S3) DeleteObject(objectKey string) error {
	_, err := s.svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return err
	}

	return nil
}

// BuildObjectKey constructs the S3 object key by combining folder path and filename
func BuildObjectKey(folderPath, filename string) string {
	if folderPath == "" {
		return filename
	}
	if strings.HasSuffix(folderPath, "/") {
		return folderPath + filename
	}
	return folderPath + "/" + filename
}

// buildUploadResult creates a BatchUploadResult from file and error
func buildUploadResult(file FileUploadInput, err error) BatchUploadResult {
	if err != nil {
		return BatchUploadResult{
			FileName: file.FileName,
			Error:    err.Error(),
			Success:  false,
		}
	}
	return BatchUploadResult{
		FileName:  file.FileName,
		ObjectKey: file.ObjectKey,
		Success:   true,
	}
}

func buildDownloadResult(path string, url string, err error) BatchDownloadResult {
	fileName := filepath.Base(path)
	if err != nil {
		return BatchDownloadResult{
			Path:     path,
			FileName: fileName,
			Error:    err.Error(),
			Success:  false,
		}
	}
	return BatchDownloadResult{
		Path:     path,
		FileName: fileName,
		URL:      url,
		Success:  true,
	}
}

/*
BatchUploadFiles uploads multiple files to S3 concurrently using a worker pool pattern.

Internal Working:
1. Worker Pool Setup: Creates a fixed number of worker goroutines to process uploads concurrently
2. Channels: Uses two buffered channels:
  - jobs: feeds FileUploadInput to workers (buffer size = total files)
  - results: collects BatchUploadResult from workers (buffer size = total files)

3. Workers: Each worker goroutine:
  - Reads files from jobs channel
  - Checks for context cancellation (client disconnect)
  - Uploads to S3 with panic recovery
  - Sends result to results channel

4. Job Distribution: A separate goroutine pushes all files to jobs channel and closes it
5. Result Collection: Main goroutine collects results and categorizes them as uploaded/failed
6. Cleanup: Uses WaitGroup to ensure all workers finish before closing results channel

This pattern provides controlled concurrency, prevents resource exhaustion, and handles failures gracefully.
*/
func (s *S3) BatchUploadFiles(ctx context.Context, files []FileUploadInput, maxWorkers int) *BatchUploadResponse {
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
								Error:    formatPanicError(r),
								Success:  false,
							}
						}
					}()

					err := s.UploadFile(file.Reader, file.ObjectKey)
					results <- buildUploadResult(file, err)
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

/*
BatchGenerateDownloadLinks generates presigned URLs for multiple files concurrently using a worker pool pattern.

Internal Working:
1. Worker Pool Setup: Creates a fixed number of worker goroutines to generate URLs concurrently
2. Channels: Uses two buffered channels:
  - jobs: feeds file paths (strings) to workers (buffer size = total paths)
  - results: collects BatchDownloadResult from workers (buffer size = total paths)

3. Workers: Each worker goroutine:
  - Reads paths from jobs channel
  - Checks for context cancellation (client disconnect)
  - Generates presigned URL with panic recovery and cache support
  - Sends result to results channel

4. Job Distribution: A separate goroutine pushes all paths to jobs channel and closes it
5. Result Collection: Main goroutine collects all results into a single response
6. Cleanup: Uses WaitGroup to ensure all workers finish before closing results channel

This pattern provides controlled concurrency and leverages URL caching for better performance.
*/
func (s *S3) BatchGenerateDownloadLinks(ctx context.Context, paths []string, urlCache *cache.URLCache, maxWorkers int) *BatchDownloadResponse {
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
								Error:    formatPanicError(r),
								Success:  false,
							}
						}
					}()

					url, err := s.GenerateDownloadLink(path, urlCache)
					results <- buildDownloadResult(path, url, err)
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

	// Close results channel after all workers finish
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

// DeleteFolder deletes a folder and its contents recursively from the S3 bucket.
func (s *S3) DeleteFolder(folderPath string) error {

	// add a trailing slash to the folder path if not already present
	if folderPath != "" && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	allObjects := []ObjectDetails{}

	resp, err := s.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucketName),
		Prefix:  aws.String(folderPath),
		MaxKeys: aws.Int64(2),
	})

	for _, obj := range resp.Contents {

		if *obj.Key == folderPath {
			continue // skip the folder itself
		}

		allObjects = append(allObjects, ObjectDetails{
			Name:         *obj.Key,
			IsFolder:     *obj.Size == 0,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
		})
	}

	if err != nil {
		return err
	}

	nextToken := resp.NextContinuationToken

	for nextToken != nil {

		curr, err := s.svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucketName),
			Prefix:            aws.String(folderPath),
			MaxKeys:           aws.Int64(1000),
			ContinuationToken: nextToken,
		})

		if err != nil {
			return err
		}

		for _, obj := range curr.Contents {

			if *obj.Key == folderPath {
				continue // skip the folder itself
			}

			allObjects = append(allObjects, ObjectDetails{
				Name:         *obj.Key,
				IsFolder:     *obj.Size == 0,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})

			// update the next token
			nextToken = curr.NextContinuationToken

			if nextToken == nil {
				break
			}
		}

	}

	for _, obj := range allObjects {
		err := s.DeleteObject(obj.Name)
		if err != nil {
			return err
		}
	}

	// delete the folder itself
	err = s.DeleteObject(folderPath)

	if err != nil {
		return err
	}

	return nil
}

// ListAllFolders lists all the folders within a folder in the S3 bucket.
func (s *S3) ListAllFolders(folderPath string) []ObjectDetails {
	// add a trailing slash to the folder path if not already present
	if folderPath != "" && !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	allObjects := []ObjectDetails{}

	resp, err := s.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucketName),
		Prefix:  aws.String(folderPath),
		MaxKeys: aws.Int64(1000),
	})

	if err != nil {
		return allObjects
	}

	for _, obj := range resp.Contents {

		if *obj.Key == folderPath {
			continue // skip the folder itself
		}

		if *obj.Size == 0 {
			allObjects = append(allObjects, ObjectDetails{
				Name:         *obj.Key,
				IsFolder:     *obj.Size == 0,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})
		}
	}

	nextToken := resp.NextContinuationToken

	for nextToken != nil {
		curr, err := s.svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucketName),
			Prefix:            aws.String(folderPath),
			MaxKeys:           aws.Int64(1000),
			ContinuationToken: nextToken,
		})

		if err != nil {
			break
		}

		for _, obj := range curr.Contents {

			if *obj.Key == folderPath {
				continue // skip the folder itself
			}

			if *obj.Size == 0 {
				allObjects = append(allObjects, ObjectDetails{
					Name:         *obj.Key,
					IsFolder:     *obj.Size == 0,
					Size:         *obj.Size,
					LastModified: *obj.LastModified,
				})
			}

			// update the next token
			nextToken = curr.NextContinuationToken
			if nextToken == nil {
				break
			}
		}

	}

	return allObjects
}
