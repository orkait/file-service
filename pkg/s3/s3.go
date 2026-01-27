package s3

import (
	"file-management-service/config"
	"file-management-service/pkg/cache"
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

func (s *S3) ListAllFiles(folderPath string) (*ListFilesResponse, error) {
	objects, err := s.ListFiles(folderPath, "", 10, false, &cache.URLCache{})
	nextToken := objects.NextPageToken
	if err != nil {
		return nil, err
	}

	var allObjects []ObjectDetails

	// check if next page token is present
	for nextToken != "" {
		temp, _ := s.ListFiles(folderPath, nextToken, 10, false, &cache.URLCache{})
		allObjects = append(allObjects, *temp.Files...)

		if temp.IsLastPage {
			nextToken = ""
		}
		nextToken = temp.NextPageToken
	}

	// Helper function to recursively fetch objects from subfolders
	var listObjectsRecursively func(path string) error
	listObjectsRecursively = func(path string) error {
		objects, err := s.ListFiles(path, "", 10, false, &cache.URLCache{})
		nextToken := objects.NextPageToken

		// check if next page token is present
		for nextToken != "" {
			t, _ := s.ListFiles(path, nextToken, 10, false, &cache.URLCache{})
			allObjects = append(allObjects, *t.Files...)

			if t.IsLastPage {
				nextToken = ""
			}
			nextToken = t.NextPageToken
		}

		if err != nil {
			return err
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

// BatchUploadFiles uploads multiple files to S3 concurrently
func (s *S3) BatchUploadFiles(files []FileUploadInput) *BatchUploadResponse {
	var wg sync.WaitGroup
	var mu sync.Mutex

	result := &BatchUploadResponse{
		Uploaded: []BatchUploadResult{},
		Failed:   []BatchUploadResult{},
	}

	for _, file := range files {
		wg.Add(1)
		go func(f FileUploadInput) {
			defer wg.Done()

			err := s.UploadFile(f.Reader, f.ObjectKey)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Failed = append(result.Failed, BatchUploadResult{
					FileName: f.FileName,
					Error:    err.Error(),
					Success:  false,
				})
				result.TotalFailed++
			} else {
				result.Uploaded = append(result.Uploaded, BatchUploadResult{
					FileName:  f.FileName,
					ObjectKey: f.ObjectKey,
					Success:   true,
				})
				result.TotalUploaded++
			}
		}(file)
	}

	wg.Wait()
	return result
}

// BatchGenerateDownloadLinks generates download URLs for multiple files concurrently
func (s *S3) BatchGenerateDownloadLinks(paths []string, urlCache *cache.URLCache) *BatchDownloadResponse {
	var wg sync.WaitGroup
	var mu sync.Mutex

	result := &BatchDownloadResponse{
		Files: []BatchDownloadResult{},
	}

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			url, err := s.GenerateDownloadLink(p, urlCache)
			fileName := filepath.Base(p)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Files = append(result.Files, BatchDownloadResult{
					Path:     p,
					FileName: fileName,
					Error:    err.Error(),
					Success:  false,
				})
				result.TotalFailed++
			} else {
				result.Files = append(result.Files, BatchDownloadResult{
					Path:     p,
					FileName: fileName,
					URL:      url,
					Success:  true,
				})
				result.TotalSuccess++
			}
		}(path)
	}

	wg.Wait()
	return result
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

	if err != nil {
		return allObjects
	}

	nextToken := resp.NextContinuationToken

	for nextToken != nil {
		curr, _ := s.svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucketName),
			Prefix:            aws.String(folderPath),
			MaxKeys:           aws.Int64(1000),
			ContinuationToken: nextToken,
		})

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
