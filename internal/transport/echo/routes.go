package echo

import (
	"errors"
	"file-service/internal/infra/s3"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"
)

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	s.echo.POST("/upload", s.uploadFileHandler)
	s.echo.GET("/download", s.downloadFileHandler)
	s.echo.DELETE("/delete", s.deleteFileHandler)
	s.echo.DELETE("/delete-folder", s.deleteFolderHandler)
	s.echo.GET("/list", s.listFilesHandler)
	s.echo.GET("/list-folders", s.listAllFoldersHandler)
	s.echo.POST("/create-folder", s.createFolderHandler)
	s.echo.POST("/batch-upload", s.batchUploadFileHandler)
	s.echo.POST("/batch-download", s.batchDownloadHandler)
	s.echo.GET("/ping", s.pingHandler)
}

func (s *Server) createFolderHandler(c echo.Context) error {
	folderName := c.QueryParam("path")

	if folderName == "" {
		return c.JSON(http.StatusBadRequest, getFailureResponse(errors.New("folder path is required and should end with /")))
	}

	if string(folderName[len(folderName)-1]) != "/" {
		folderName = folderName + "/"
	}

	if err := s.s3Client.CreateFolder(folderName); err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(errors.New("failed to create folder")))
	}

	return c.JSON(http.StatusOK, getSuccessResponse("Folder created successfully"))
}

func (s *Server) uploadFileHandler(c echo.Context) error {
	folderPath := c.FormValue("path")
	file, err := c.FormFile("file")

	if err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(fmt.Errorf("failed to retrieve uploaded file: %w", err)))
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(fmt.Errorf("failed to open uploaded file: %w", err)))
	}
	defer src.Close()

	objectKey := s3.BuildObjectKey(folderPath, file.Filename)

	if err := s.s3Client.UploadFile(src, objectKey); err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(fmt.Errorf("failed to upload file to S3: %w", err)))
	}

	return c.JSON(http.StatusOK, getSuccessResponse(fmt.Sprintf("File uploaded successfully with object key: %s", objectKey)))
}

func (s *Server) listFilesHandler(c echo.Context) error {
	isFolder, err := strconv.ParseBool(c.QueryParam("isFolder"))
	if err != nil {
		isFolder = false
	}

	folderPath := c.QueryParam("path")
	nextPageToken := c.Request().Header.Get("x-next")

	pageSize, err := strconv.Atoi(c.QueryParam("pageSize"))
	if err != nil {
		pageSize = s.config.PaginationPageSize
	}

	objects, err := s.s3Client.ListFiles(folderPath, nextPageToken, pageSize, isFolder, s.urlCache)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(err))
	}

	return c.JSON(http.StatusOK, getListFolderSuccessResponse(objects))
}

func (s *Server) listAllFoldersHandler(c echo.Context) error {
	folderPath := c.QueryParam("path")
	objects := s.s3Client.ListAllFolders(folderPath)
	return c.JSON(http.StatusOK, objects)
}

func (s *Server) downloadFileHandler(c echo.Context) error {
	key := c.QueryParam("path")

	url, err := s.s3Client.GenerateDownloadLink(key, s.urlCache)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(err))
	}

	fileName := filepath.Base(key)

	if fileName != "" {
		return c.JSON(http.StatusOK, SuccessResponse{
			Status:       "Success",
			ResponseCode: http.StatusOK,
			Data: map[string]string{
				"url":      url,
				"fileName": fileName,
			},
		})
	}

	return c.JSON(http.StatusInternalServerError, getFailureResponse(err))
}

func (s *Server) deleteFileHandler(c echo.Context) error {
	path := c.QueryParam("path")

	if err := s.s3Client.DeleteObject(path); err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(err))
	}

	return c.JSON(http.StatusOK, getSuccessResponse("File deleted successfully"))
}

func (s *Server) deleteFolderHandler(c echo.Context) error {
	folderPath := c.QueryParam("path")

	if err := s.s3Client.DeleteFolder(folderPath); err != nil {
		return c.JSON(http.StatusInternalServerError, getFailureResponse(err))
	}

	return c.JSON(http.StatusOK, getSuccessResponse("Folder deleted successfully"))
}

func (s *Server) pingHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "pong"})
}

func (s *Server) batchUploadFileHandler(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, getFailureResponse(err))
	}

	folderPath := ""
	if paths, ok := form.Value["path"]; ok && len(paths) > 0 {
		folderPath = paths[0]
	}

	files, ok := form.File["files"]
	if !ok || len(files) == 0 {
		return c.JSON(http.StatusBadRequest, getFailureResponse(errors.New(s3.ErrorNoFilesProvided)))
	}

	if len(files) > s3.MaxBatchSize {
		return c.JSON(http.StatusBadRequest, getFailureResponse(fmt.Errorf(s3.ErrorMaxBatchExceeded, s3.MaxBatchSize)))
	}

	var uploadInputs []s3.FileUploadInput
	var openedFiles []multipart.File

	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			continue
		}
		openedFiles = append(openedFiles, src)

		uploadInputs = append(uploadInputs, s3.FileUploadInput{
			Reader:    src,
			FileName:  file.Filename,
			ObjectKey: s3.BuildObjectKey(folderPath, file.Filename),
		})
	}

	if len(uploadInputs) == 0 {
		for _, f := range openedFiles {
			f.Close()
		}
		return c.JSON(http.StatusBadRequest, getFailureResponse(errors.New("failed to open any files for upload")))
	}

	ctx := c.Request().Context()
	result := s.s3Client.BatchUploadFiles(ctx, uploadInputs, s3.DefaultMaxWorkers)

	for _, f := range openedFiles {
		f.Close()
	}

	return c.JSON(http.StatusOK, getSuccessResponseWithData(result))
}

func (s *Server) batchDownloadHandler(c echo.Context) error {
	var req s3.BatchDownloadRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, getFailureResponse(err))
	}

	if len(req.Paths) == 0 {
		return c.JSON(http.StatusBadRequest, getFailureResponse(errors.New(s3.ErrorNoPathsProvided)))
	}

	if len(req.Paths) > s3.MaxBatchSize {
		return c.JSON(http.StatusBadRequest, getFailureResponse(fmt.Errorf(s3.ErrorMaxBatchExceeded, s3.MaxBatchSize)))
	}

	ctx := c.Request().Context()
	result := s.s3Client.BatchGenerateDownloadLinks(ctx, req.Paths, s.urlCache, s3.DefaultMaxWorkers)

	return c.JSON(http.StatusOK, getSuccessResponseWithData(result))
}
