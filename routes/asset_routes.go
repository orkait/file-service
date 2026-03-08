package routes

import (
	"file-service/pkg/cache"
	"file-service/pkg/repository"
	"file-service/pkg/s3"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AssetRoutes struct {
	s3Client    *s3.S3
	assetRepo   *repository.AssetRepository
	projectRepo *repository.ProjectRepository
	memberRepo  *repository.MemberRepository
	urlCache    *cache.URLCache
}

func NewAssetRoutes(s3Client *s3.S3, assetRepo *repository.AssetRepository, projectRepo *repository.ProjectRepository, memberRepo *repository.MemberRepository, urlCache *cache.URLCache) *AssetRoutes {
	return &AssetRoutes{
		s3Client:    s3Client,
		assetRepo:   assetRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		urlCache:    urlCache,
	}
}

func buildS3Key(clientID, projectID, folderPath, assetID, filename string) string {
	if folderPath == "" {
		folderPath = "/"
	}
	return fmt.Sprintf("%s/%s%s%s/%s", clientID, projectID, folderPath, assetID, filename)
}

func normalizeFolderPath(folderPath string) string {
	if folderPath == "" {
		return "/"
	}
	if !strings.HasPrefix(folderPath, "/") {
		folderPath = "/" + folderPath
	}
	if !strings.HasSuffix(folderPath, "/") {
		folderPath = folderPath + "/"
	}
	return folderPath
}

func (ar *AssetRoutes) verifyProjectAccess(projectID, clientID string) error {
	_, err := ar.projectRepo.GetProjectByID(projectID, clientID)
	return err
}

func (ar *AssetRoutes) UploadAsset(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.FormValue("project_id")
	folderPath := normalizeFolderPath(c.FormValue("folder_path"))
	createVersion := c.FormValue("create_version") == "true"
	parentAssetID := c.FormValue("parent_asset_id")

	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id required"})
	}

	if err := ar.verifyProjectAccess(projectID, clientID); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "project not found or access denied"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file required"})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open file"})
	}
	defer src.Close()

	assetID := uuid.New().String()
	s3Key := buildS3Key(clientID, projectID, folderPath, assetID, file.Filename)

	if err := ar.s3Client.UploadFile(src, s3Key); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to upload file"})
	}

	var asset *repository.Asset
	if createVersion && parentAssetID != "" {
		asset, err = ar.assetRepo.CreateAssetVersion(clientID, projectID, folderPath, file.Filename, file.Filename, file.Size, file.Header.Get("Content-Type"), s3Key, parentAssetID)
	} else {
		asset, err = ar.assetRepo.CreateAsset(clientID, projectID, folderPath, file.Filename, file.Filename, file.Size, file.Header.Get("Content-Type"), s3Key)
	}

	if err != nil {
		ar.s3Client.DeleteObject(s3Key)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to record asset"})
	}

	presignedURL, err := ar.s3Client.GenerateDownloadLink(s3Key, ar.urlCache)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate download URL"})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"asset":         asset,
		"presigned_url": presignedURL,
	})
}

func (ar *AssetRoutes) GetAssets(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.QueryParam("project_id")
	folderPath := c.QueryParam("folder_path")

	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id required"})
	}

	if err := ar.verifyProjectAccess(projectID, clientID); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "project not found or access denied"})
	}

	var folderPtr *string
	if folderPath != "" {
		normalized := normalizeFolderPath(folderPath)
		folderPtr = &normalized
	}

	assets, err := ar.assetRepo.GetAssetsByProjectID(projectID, clientID, folderPtr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get assets"})
	}

	for i := range assets {
		if presignedURL, err := ar.s3Client.GenerateDownloadLink(assets[i].S3Key, ar.urlCache); err == nil {
			assets[i].PresignedURL = presignedURL
		}
	}

	return c.JSON(http.StatusOK, assets)
}

func (ar *AssetRoutes) GetAsset(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	assetID := c.Param("id")

	asset, err := ar.assetRepo.GetAssetByID(assetID, clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "asset not found"})
	}

	presignedURL, err := ar.s3Client.GenerateDownloadLink(asset.S3Key, ar.urlCache)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate download URL"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"asset":         asset,
		"presigned_url": presignedURL,
	})
}

func (ar *AssetRoutes) DeleteAsset(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	assetID := c.Param("id")

	asset, err := ar.assetRepo.GetAssetByID(assetID, clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "asset not found"})
	}

	if err = ar.s3Client.DeleteObject(asset.S3Key); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete from storage"})
	}

	if err = ar.assetRepo.DeleteAsset(assetID, clientID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete asset record"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "asset deleted"})
}

func (ar *AssetRoutes) GetUploadURL(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.QueryParam("project_id")
	folderPath := normalizeFolderPath(c.QueryParam("folder_path"))
	filename := c.QueryParam("filename")

	if projectID == "" || filename == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id and filename required"})
	}

	if err := ar.verifyProjectAccess(projectID, clientID); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "project not found or access denied"})
	}

	assetID := uuid.New().String()
	ext := filepath.Ext(filename)
	generatedFilename := assetID + ext
	s3Key := buildS3Key(clientID, projectID, folderPath, assetID, generatedFilename)

	presignedPost, err := ar.s3Client.GeneratePresignedPost(s3Key, 100*1024*1024, 15*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate upload URL"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"upload_url": presignedPost.URL,
		"fields":     presignedPost.Fields,
		"asset_id":   assetID,
		"s3_key":     s3Key,
		"filename":   generatedFilename,
		"expires_in": 900,
	})
}

func (ar *AssetRoutes) ConfirmUpload(c echo.Context) error {
	clientID := c.Get("client_id").(string)

	var req struct {
		ProjectID        string `json:"project_id"`
		FolderPath       string `json:"folder_path"`
		AssetID          string `json:"asset_id"`
		S3Key            string `json:"s3_key"`
		Filename         string `json:"filename"`
		OriginalFilename string `json:"original_filename"`
		FileSize         int64  `json:"file_size"`
		MimeType         string `json:"mime_type"`
		CreateVersion    bool   `json:"create_version"`
		ParentAssetID    string `json:"parent_asset_id"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.ProjectID == "" || req.S3Key == "" || req.Filename == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id, s3_key, and filename required"})
	}

	req.FolderPath = normalizeFolderPath(req.FolderPath)

	if err := ar.verifyProjectAccess(req.ProjectID, clientID); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "project not found or access denied"})
	}

	var asset *repository.Asset
	var err error
	if req.CreateVersion && req.ParentAssetID != "" {
		asset, err = ar.assetRepo.CreateAssetVersion(clientID, req.ProjectID, req.FolderPath, req.Filename, req.OriginalFilename, req.FileSize, req.MimeType, req.S3Key, req.ParentAssetID)
	} else {
		asset, err = ar.assetRepo.CreateAsset(clientID, req.ProjectID, req.FolderPath, req.Filename, req.OriginalFilename, req.FileSize, req.MimeType, req.S3Key)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to record asset"})
	}

	presignedURL, err := ar.s3Client.GenerateDownloadLink(req.S3Key, ar.urlCache)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate download URL"})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"asset":         asset,
		"presigned_url": presignedURL,
	})
}

func (ar *AssetRoutes) GetFolders(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.QueryParam("project_id")

	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id required"})
	}

	if err := ar.verifyProjectAccess(projectID, clientID); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "project not found or access denied"})
	}

	folders, err := ar.assetRepo.GetFoldersByProjectID(projectID, clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get folders"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"folders": folders,
	})
}

func (ar *AssetRoutes) GetAssetVersions(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	assetID := c.Param("id")

	versions, err := ar.assetRepo.GetAssetVersions(assetID, clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "asset not found"})
	}

	for i := range versions {
		if presignedURL, err := ar.s3Client.GenerateDownloadLink(versions[i].S3Key, ar.urlCache); err == nil {
			versions[i].PresignedURL = presignedURL
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"versions": versions,
		"total":    len(versions),
	})
}
