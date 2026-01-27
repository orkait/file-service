package routes

import (
	"errors"
	"file-management-service/config"
	"file-management-service/pkg/cache"
	"file-management-service/pkg/s3"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers all the routes for the application
func RegisterRoutes(e *echo.Echo, config *config.Config, cache *cache.URLCache) {
	// Define route for uploading images
	e.POST("/upload", func(c echo.Context) error {
		return uploadFileHandler(c, config)
	})

	// Define route for serving files
	e.GET("/download", func(c echo.Context) error {
		return downloadFileHandler(c, config, cache)
	})

	// Delete File
	e.DELETE("/delete", func(c echo.Context) error {
		return deleteFileHandler(c, config, cache)
	})

	// Delete File
	e.DELETE("/delete-folder", func(c echo.Context) error {
		return deleteFolderHandler(c, config)
	})

	// List files within current folder
	e.GET("/list", func(c echo.Context) error {
		return listFilesHandler(c, config, cache)
	})

	// list all folders within current folder
	e.GET("/list-folders", func(c echo.Context) error {
		return listAllFoldersHandler(c, config)
	})

	e.POST("/create-folder", func(c echo.Context) error {
		return createFolderHandler(c, config)
	})

	// Batch upload multiple files
	e.POST("/batch-upload", func(c echo.Context) error {
		return batchUploadFileHandler(c, config)
	})

	// Batch download - get multiple download URLs
	e.POST("/batch-download", func(c echo.Context) error {
		return batchDownloadHandler(c, config, cache)
	})

	// Define route for testing the server
	e.GET("/ping", ping)
}

// Handler to create folder
// createFolderHandler is a handler function for creating a folder in S3
func createFolderHandler(c echo.Context, config *config.Config) error {

	folderName := c.QueryParam("path")

	if folderName == "" {
		response := s3.GetFailureResponse(errors.New("folder path is required and should end with /"))
		return c.JSON(http.StatusBadRequest, response)
	}

	if string(folderName[len(folderName)-1]) != "/" {
		folderName = folderName + "/"
	}

	// Create a new S3 client using your desired bucket name and region
	client, err := s3.NewClient(config)
	if err != nil {
		// Handle error creating S3 client
		response := s3.GetFailureResponse(errors.New("failed to create S3 client"))
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Call the CreateFolder function to create the folder
	err = client.CreateFolder(folderName)
	if err != nil {
		// Handle error creating folder
		response := s3.GetFailureResponse(errors.New("failed to create folder"))
		return c.JSON(http.StatusInternalServerError, response)
	}
	response := s3.GetSuccessResponse("Folder created successfully")
	return c.JSON(http.StatusOK, response)
}

// Handler for image upload
func uploadFileHandler(c echo.Context, config *config.Config) error {
	folderPath := c.FormValue("path")
	file, err := c.FormFile("file")

	if err != nil {
		// Handle the error and return an error response
		errorMessage := fmt.Sprintf("Failed to retrieve uploaded file: %s", err.Error())
		response := s3.GetFailureResponse(errors.New(errorMessage))
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		// Handle the error and return an error response
		errorMessage := fmt.Sprintf("Failed to open uploaded file: %s", err.Error())
		response := s3.GetFailureResponse(errors.New(errorMessage))
		return c.JSON(http.StatusInternalServerError, response)
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			// Handle the error (optional)
			fmt.Println("Failed to close uploaded file:", closeErr)
		}
	}()

	// Create a new S3 client
	client, err := s3.NewClient(config)
	if err != nil {
		// Handle the error and return an error response
		errorMessage := fmt.Sprintf("Failed to create S3 client: %s", err.Error())
		response := s3.GetFailureResponse(errors.New(errorMessage))
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Use the file name as it is as the object key
	objectKey := file.Filename
	// Add the folder details
	if folderPath != "" {
		if string(folderPath[len(folderPath)-1]) == "/" {
			objectKey = folderPath + objectKey
		} else {
			objectKey = folderPath + "/" + objectKey
		}
	}

	// Upload the file to S3
	err = client.UploadFile(src, objectKey)
	if err != nil {
		// Handle the error and return an error response
		errorMessage := fmt.Sprintf("Failed to upload file to S3: %s", err.Error())
		response := s3.GetFailureResponse(errors.New(errorMessage))
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Return a success response
	successMessage := fmt.Sprintf("File uploaded successfully with object key: %s", objectKey)
	response := s3.GetSuccessResponse(successMessage)
	// Return the array of file and folder information as JSON response
	return c.JSON(http.StatusOK, response)
}

// List all files and folders within a folder
func listFilesHandler(c echo.Context, config *config.Config, cache *cache.URLCache) error {

	// bool
	isFolder, err := strconv.ParseBool(c.QueryParam("isFolder"))
	if err != nil {
		isFolder = false
	}

	folderPath := c.QueryParam("path")

	// Next page token for pagination
	nextPageToken := c.Request().Header.Get("x-next")

	// Page size for pagination
	pageSize, err := strconv.Atoi(c.QueryParam("pageSize"))
	if err != nil {
		pageSize = config.PaginationPageSize
	}

	// Create a new S3 client
	client, err := s3.NewClient(config) // Update with your desired region

	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	// List all the files and folders within the nested folder
	objects, err := client.ListFiles(folderPath, nextPageToken, pageSize, isFolder, cache)

	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	response := s3.GetListFolderSuccessResponse(objects)
	return c.JSON(http.StatusOK, response)
}

func listAllFilesHandler(c echo.Context, config *config.Config) error {
	// Create a new S3 client
	client, err := s3.NewClient(config) // Update with your desired region

	folderPath := c.QueryParam("path")

	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	// List all the files and folders within the nested folder
	objects, err := client.ListAllFiles(folderPath)

	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	return c.JSON(http.StatusOK, objects)
}

func listAllFoldersHandler(c echo.Context, config *config.Config) error {
	// Create a new S3 client
	client, err := s3.NewClient(config) // Update with your desired region
	folderPath := c.QueryParam("path")

	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	// List all the files and folders within the nested folder
	objects := client.ListAllFolders(folderPath)

	return c.JSON(http.StatusOK, objects)
}

// Handler for downloading a file
func downloadFileHandler(c echo.Context, config *config.Config, cache *cache.URLCache) error {
	key := c.QueryParam("path")

	// Create a new S3 client
	client, err := s3.NewClient(config) // Update with your desired region
	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	url, err := client.GenerateDownloadLink(key, cache)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, s3.GetFailureResponse(err))
	}

	// Get the fileName, ignoring folders in prefix.
	fileName := filepath.Base(key)

	if fileName != "" {
		return c.JSON(http.StatusOK,
			s3.SuccessResponse{
				Status:       "Success",
				ResponseCode: http.StatusOK,
				Data: map[string]string{
					"url":      url,
					"fileName": fileName,
				},
			})
	}

	return c.JSON(http.StatusInternalServerError, s3.GetFailureResponse(err))
}

func deleteFileHandler(c echo.Context, config *config.Config, cache *cache.URLCache) error {
	// bucket := c.QueryParam("bucket")
	path := c.QueryParam("path")

	// Create a new S3 client
	client, err := s3.NewClient(config) // Update with your desired region
	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Delete the file or folder from the S3 bucket
	err = client.DeleteObject(path)
	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Return a success response
	response := s3.GetSuccessResponse("File deleted successfully")
	return c.JSON(http.StatusOK, response)
}

func deleteFolderHandler(c echo.Context, config *config.Config) error {
	// bucket := c.QueryParam("bucket")
	folderPath := c.QueryParam("path")

	// Create a new S3 client
	client, err := s3.NewClient(config) // Update with your desired region
	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Delete the file or folder from the S3 bucket
	err = client.DeleteFolder(folderPath)
	if err != nil {
		response := s3.GetFailureResponse(err)
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Return a success response
	response := s3.GetSuccessResponse("Folder deleted successfully")
	return c.JSON(http.StatusOK, response)
}

// ping is a simple handler to test the server
func ping(c echo.Context) error {
	response := map[string]string{"message": "pong"}
	return c.JSON(http.StatusOK, response)
}

// batchUploadFileHandler handles uploading multiple files in a single request
func batchUploadFileHandler(c echo.Context, config *config.Config) error {
	folderPath := c.FormValue("path")

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		response := s3.GetFailureResponse(errors.New("failed to parse multipart form"))
		return c.JSON(http.StatusBadRequest, response)
	}

	files := form.File["files"]
	if len(files) == 0 {
		response := s3.GetFailureResponse(errors.New("no files provided"))
		return c.JSON(http.StatusBadRequest, response)
	}

	// Limit batch size to 100 files
	if len(files) > 100 {
		response := s3.GetFailureResponse(errors.New("maximum 100 files allowed per batch"))
		return c.JSON(http.StatusBadRequest, response)
	}

	// Create S3 client
	client, err := s3.NewClient(config)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to create S3 client: %s", err.Error())
		response := s3.GetFailureResponse(errors.New(errorMessage))
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Prepare files for batch upload
	var uploadInputs []s3.FileUploadInput
	var openedFiles []multipart.File // Track opened files for cleanup

	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			continue // Skip files that can't be opened
		}
		openedFiles = append(openedFiles, src)

		// Build object key with folder path
		objectKey := file.Filename
		if folderPath != "" {
			if strings.HasSuffix(folderPath, "/") {
				objectKey = folderPath + objectKey
			} else {
				objectKey = folderPath + "/" + objectKey
			}
		}

		uploadInputs = append(uploadInputs, s3.FileUploadInput{
			Reader:    src,
			FileName:  file.Filename,
			ObjectKey: objectKey,
		})
	}

	// Perform batch upload
	result := client.BatchUploadFiles(uploadInputs)

	// Close all opened files after upload completes
	for _, f := range openedFiles {
		f.Close()
	}

	response := s3.GetSuccessResponseWithData(result)
	return c.JSON(http.StatusOK, response)
}

// batchDownloadHandler handles generating download URLs for multiple files
func batchDownloadHandler(c echo.Context, config *config.Config, urlCache *cache.URLCache) error {
	var req s3.BatchDownloadRequest
	if err := c.Bind(&req); err != nil {
		response := s3.GetFailureResponse(errors.New("invalid request body"))
		return c.JSON(http.StatusBadRequest, response)
	}

	if len(req.Paths) == 0 {
		response := s3.GetFailureResponse(errors.New("no paths provided"))
		return c.JSON(http.StatusBadRequest, response)
	}

	// Limit batch size to 100 files
	if len(req.Paths) > 100 {
		response := s3.GetFailureResponse(errors.New("maximum 100 files allowed per batch"))
		return c.JSON(http.StatusBadRequest, response)
	}

	// Create S3 client
	client, err := s3.NewClient(config)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to create S3 client: %s", err.Error())
		response := s3.GetFailureResponse(errors.New(errorMessage))
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Generate download URLs for all paths
	result := client.BatchGenerateDownloadLinks(req.Paths, urlCache)

	response := s3.GetSuccessResponseWithData(result)
	return c.JSON(http.StatusOK, response)
}
