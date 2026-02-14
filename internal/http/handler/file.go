package handler

import (
	"context"
	"errors"
	"file-service/internal/auth"
	"file-service/internal/domain/file"
	"file-service/internal/storage/s3"
	"file-service/internal/types"
	apperrors "file-service/pkg/errors"
	"file-service/pkg/validator"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type FileHandler struct {
	fileRepo    FileRepository
	folderRepo  FolderRepository
	projectRepo ProjectGetter
	s3Client    StorageOperations
	auditLogger types.AuditLogger
}

func NewFileHandler(
	fileRepo FileRepository,
	folderRepo FolderRepository,
	projectRepo ProjectGetter,
	s3Client StorageOperations,
	auditLogger types.AuditLogger,
) *FileHandler {
	return &FileHandler{
		fileRepo:    fileRepo,
		folderRepo:  folderRepo,
		projectRepo: projectRepo,
		s3Client:    s3Client,
		auditLogger: auditLogger,
	}
}

type GetUploadURLRequest struct {
	ProjectID   string `json:"project_id"`
	FolderPath  string `json:"folder_path"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type GetUploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
	S3Key     string `json:"s3_key"`
}

func (h *FileHandler) GetUploadURL(c echo.Context) error {
	var req GetUploadURLRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}
	req.FileName = strings.TrimSpace(req.FileName)
	req.FolderPath = strings.Trim(strings.TrimSpace(req.FolderPath), "/")
	req.ContentType = strings.TrimSpace(req.ContentType)

	if err := validator.FileName(req.FileName); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	// Sanitize content-type to prevent injection attacks
	sanitizedContentType, err := validator.SanitizeContentType(req.ContentType)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	req.ContentType = sanitizedContentType

	if err := validator.FileSize(req.SizeBytes); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if err := validator.ValidateFolderPathSecure(req.FolderPath); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	projectID, err := h.resolveProjectID(c, firstNonEmpty(req.ProjectID, c.Param(paramProjectID)))
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	proj, err := h.projectRepo.GetByID(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}
	// API key scope validation is now handled by RBAC middleware

	if req.FolderPath != "" {
		folder, err := h.folderRepo.GetFolderByPath(c.Request().Context(), projectID, req.FolderPath)
		if err != nil || folder.ProjectID != projectID {
			return respondError(c, http.StatusBadRequest, msgFolderNotFound)
		}
	}

	s3Key := s3.BuildObjectKey(req.FolderPath, req.FileName)

	uploadURL, err := h.s3Client.GeneratePresignedUploadURL(c.Request().Context(), proj.S3BucketName, s3Key, req.ContentType)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgUploadURLGenerateFail)
	}

	var uploaderID *uuid.UUID
	if auth.GetAuthType(c) == auth.AuthTypeJWT {
		userID, err := auth.GetUserID(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, err.Error())
		}
		uploaderID = &userID
	}

	fileRecord, err := h.fileRepo.GetByProjectAndS3Key(c.Request().Context(), projectID, s3Key)
	if err == nil {
		if err := h.fileRepo.Update(c.Request().Context(), fileRecord.ID, file.UpdateFileInput{
			Name:      &req.FileName,
			SizeBytes: &req.SizeBytes,
			MimeType:  &req.ContentType,
		}); err != nil {
			return respondError(c, http.StatusInternalServerError, msgUpdateFileMetadataFail)
		}
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return respondError(c, http.StatusInternalServerError, msgCheckExistingFileFail)
	} else {
		fileRecord, err = h.fileRepo.Create(c.Request().Context(), file.CreateFileInput{
			ProjectID:  projectID,
			Name:       req.FileName,
			S3Key:      s3Key,
			SizeBytes:  req.SizeBytes,
			MimeType:   req.ContentType,
			UploadedBy: uploaderID,
		})
		if err != nil {
			if errors.Is(err, apperrors.ErrConflict) {
				fileRecord, err = h.fileRepo.GetByProjectAndS3Key(c.Request().Context(), projectID, s3Key)
				if err == nil {
					return c.JSON(http.StatusOK, GetUploadURLResponse{
						UploadURL: uploadURL,
						FileID:    fileRecord.ID.String(),
						S3Key:     s3Key,
					})
				}
			}
			return respondError(c, http.StatusInternalServerError, msgCreateFileRecordFail)
		}
	}

	// Log presigned URL generation
	if h.auditLogger != nil {
		metadata := map[string]any{
			"file_id":      fileRecord.ID.String(),
			"project_id":   projectID.String(),
			"file_name":    req.FileName,
			"s3_key":       s3Key,
			"content_type": req.ContentType,
			"size_bytes":   req.SizeBytes,
			"operation":    "upload",
		}
		_ = h.auditLogger.LogFromContext(c, "file", &fileRecord.ID, "generate_presigned_url", "success", metadata)
	}

	return c.JSON(http.StatusOK, GetUploadURLResponse{
		UploadURL: uploadURL,
		FileID:    fileRecord.ID.String(),
		S3Key:     s3Key,
	})
}

func (h *FileHandler) GetFile(c echo.Context) error {
	fileID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidFileID)
	}

	fileRecord, err := h.fileRepo.GetByID(c.Request().Context(), fileID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgFileNotFound)
	}
	// API key scope validation is now handled by RBAC middleware

	return c.JSON(http.StatusOK, fileRecord)
}

func (h *FileHandler) GetDownloadURL(c echo.Context) error {
	fileID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidFileID)
	}

	fileRecord, err := h.fileRepo.GetByID(c.Request().Context(), fileID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgFileNotFound)
	}
	// API key scope validation is now handled by RBAC middleware

	proj, err := h.projectRepo.GetByID(c.Request().Context(), fileRecord.ProjectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}

	downloadURL, err := h.s3Client.GeneratePresignedDownloadURL(c.Request().Context(), proj.S3BucketName, fileRecord.S3Key)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgDownloadURLGenerateFail)
	}

	// Log presigned URL generation
	if h.auditLogger != nil {
		metadata := map[string]any{
			"file_id":    fileRecord.ID.String(),
			"project_id": fileRecord.ProjectID.String(),
			"file_name":  fileRecord.Name,
			"s3_key":     fileRecord.S3Key,
			"operation":  "download",
		}
		_ = h.auditLogger.LogFromContext(c, "file", &fileRecord.ID, "generate_presigned_url", "success", metadata)
	}

	return c.JSON(http.StatusOK, map[string]string{
		jsonKeyDownloadURL: downloadURL,
		jsonKeyFileName:    fileRecord.Name,
	})
}

func (h *FileHandler) ListFiles(c echo.Context) error {
	projectID, err := h.resolveProjectID(c, firstNonEmpty(c.QueryParam(queryProjectID), c.Param(paramProjectID)))
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	// API key scope validation is now handled by RBAC middleware

	limit, offset, err := parsePaginationParams(c, defaultFileListLimit, defaultFileListOffset)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	files, err := h.fileRepo.List(c.Request().Context(), file.ListFilesFilter{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})

	if err != nil {
		c.Logger().Errorf("Failed to list files for project %s: %v", projectID, err)
		return respondError(c, http.StatusInternalServerError, msgListFilesFail)
	}

	return c.JSON(http.StatusOK, files)
}

func (h *FileHandler) DeleteFile(c echo.Context) error {
	fileID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidFileID)
	}

	fileRecord, err := h.fileRepo.GetByID(c.Request().Context(), fileID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgFileNotFound)
	}
	// API key scope validation is now handled by RBAC middleware

	proj, err := h.projectRepo.GetByID(c.Request().Context(), fileRecord.ProjectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}

	if err := h.fileRepo.Delete(c.Request().Context(), fileID); err != nil {
		c.Logger().Errorf("Failed to delete file metadata %s: %v", fileID, err)
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "file", &fileID, "delete", err)
		}
		return respondError(c, http.StatusInternalServerError, msgDeleteFileFail)
	}

	if err := h.s3Client.DeleteObject(c.Request().Context(), proj.S3BucketName, fileRecord.S3Key); err != nil {
		c.Logger().Errorf("Failed to delete S3 object %s (orphaned): %v", fileRecord.S3Key, err)
	}

	// Log successful file deletion
	if h.auditLogger != nil {
		metadata := map[string]any{
			"file_id":    fileID.String(),
			"file_name":  fileRecord.Name,
			"project_id": fileRecord.ProjectID.String(),
			"s3_key":     fileRecord.S3Key,
		}
		_ = h.auditLogger.LogFromContext(c, "file", &fileID, "delete", "success", metadata)
	}

	return respondMessage(c, http.StatusOK, msgFileDeleted)
}

type CreateFolderRequest struct {
	ProjectID      string `json:"project_id"`
	ParentFolderID string `json:"parent_folder_id"`
	Name           string `json:"name"`
}

func (h *FileHandler) CreateFolder(c echo.Context) error {
	var req CreateFolderRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}
	req.Name = strings.TrimSpace(req.Name)

	projectID, err := h.resolveProjectID(c, firstNonEmpty(req.ProjectID, c.Param(paramProjectID)))
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	// API key scope validation is now handled by RBAC middleware

	if err := validator.FolderName(req.Name); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	var parentFolderID *uuid.UUID
	var parentPrefix string
	if req.ParentFolderID != "" {
		parentID, err := uuid.Parse(req.ParentFolderID)
		if err != nil {
			return respondError(c, http.StatusBadRequest, msgInvalidParentFolderID)
		}
		parentFolderID = &parentID

		parentFolder, err := h.folderRepo.GetFolder(c.Request().Context(), parentID)
		if err != nil {
			return respondError(c, http.StatusNotFound, msgFolderNotFound)
		}
		if parentFolder.ProjectID != projectID {
			return respondError(c, http.StatusForbidden, msgFolderProjectMismatch)
		}
		parentPrefix = parentFolder.S3Prefix

		if err := h.validateFolderDepth(c.Request().Context(), parentFolderID); err != nil {
			return respondError(c, http.StatusBadRequest, err.Error())
		}
	}

	normalizedPrefix := buildChildFolderPrefix(parentPrefix, req.Name)
	if err := validator.ValidateFolderPathSecure(normalizedPrefix); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	var createdBy *uuid.UUID
	if auth.GetAuthType(c) == auth.AuthTypeJWT {
		userID, err := auth.GetUserID(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, err.Error())
		}
		createdBy = &userID
	}

	folder, err := h.folderRepo.CreateFolder(c.Request().Context(), file.CreateFolderInput{
		ProjectID:      projectID,
		ParentFolderID: parentFolderID,
		Name:           req.Name,
		S3Prefix:       normalizedPrefix,
		CreatedBy:      createdBy,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			return respondError(c, http.StatusConflict, err.Error())
		}
		c.Logger().Errorf("Failed to create folder in project %s: %v", projectID, err)
		return respondError(c, http.StatusInternalServerError, msgCreateFolderFail)
	}

	return c.JSON(http.StatusCreated, folder)
}

func (h *FileHandler) ListFolders(c echo.Context) error {
	projectID, err := h.resolveProjectID(c, firstNonEmpty(c.QueryParam(queryProjectID), c.Param(paramProjectID)))
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	// API key scope validation is now handled by RBAC middleware

	var parentFolderID *uuid.UUID
	if parentIDRaw := c.QueryParam(queryParentID); parentIDRaw != "" {
		parentID, err := uuid.Parse(parentIDRaw)
		if err != nil {
			return respondError(c, http.StatusBadRequest, msgInvalidParentFolderID)
		}
		parentFolderID = &parentID
	}

	folders, err := h.folderRepo.ListFolders(c.Request().Context(), projectID, parentFolderID)
	if err != nil {
		c.Logger().Errorf("Failed to list folders for project %s: %v", projectID, err)
		return respondError(c, http.StatusInternalServerError, msgListFoldersFail)
	}

	return c.JSON(http.StatusOK, folders)
}

func (h *FileHandler) DeleteFolder(c echo.Context) error {
	folderID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidFolderID)
	}

	folder, err := h.folderRepo.GetFolder(c.Request().Context(), folderID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgFolderNotFound)
	}

	if c.Param(paramProjectID) != "" {
		projectID, err := h.resolveProjectID(c, c.Param(paramProjectID))
		if err != nil {
			return respondError(c, http.StatusBadRequest, err.Error())
		}
		if projectID != folder.ProjectID {
			return respondError(c, http.StatusForbidden, msgFolderProjectMismatch)
		}
	}

	// API key scope validation is now handled by RBAC middleware

	proj, err := h.projectRepo.GetByID(c.Request().Context(), folder.ProjectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}

	prefixes, err := h.collectFolderDeletionPrefixes(c.Request().Context(), folder.ProjectID, folder)
	if err != nil {
		c.Logger().Errorf("Failed to collect folder deletion prefixes for folder %s: %v", folderID, err)
		return respondError(c, http.StatusInternalServerError, msgDeleteFolderFail)
	}

	for _, prefix := range prefixes {
		if err := h.deleteFolderPrefixObjects(c.Request().Context(), proj.S3BucketName, prefix); err != nil {
			return respondError(c, http.StatusInternalServerError, msgDeleteFolderObjectsFail)
		}
	}

	for _, prefix := range prefixes {
		if _, err := h.fileRepo.DeleteByProjectAndPrefix(c.Request().Context(), folder.ProjectID, prefix); err != nil {
			return respondError(c, http.StatusInternalServerError, msgDeleteFolderMetadataFail)
		}
	}

	if err := h.folderRepo.DeleteFolder(c.Request().Context(), folderID); err != nil {
		c.Logger().Errorf("Failed to delete folder %s: %v", folderID, err)
		return respondError(c, http.StatusInternalServerError, msgDeleteFolderFail)
	}

	if err := h.verifyFolderDeletion(c.Request().Context(), folder.ProjectID, proj.S3BucketName, folderID, prefixes); err != nil {
		c.Logger().Errorf("Folder deletion verification failed for folder %s: %v", folderID, err)
		return respondError(c, http.StatusInternalServerError, msgDeleteFolderFail)
	}

	return respondMessage(c, http.StatusOK, msgFolderDeleted)
}

func (h *FileHandler) collectFolderDeletionPrefixes(ctx context.Context, projectID uuid.UUID, root *file.Folder) ([]string, error) {
	rootPrefix, err := folderPrefixForDelete(root.S3Prefix)
	if err != nil {
		return nil, err
	}

	prefixes := []string{rootPrefix}
	prefixSeen := map[string]struct{}{rootPrefix: {}}
	folderSeen := map[uuid.UUID]struct{}{root.ID: {}}
	queue := []uuid.UUID{root.ID}

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		children, err := h.folderRepo.ListFolders(ctx, projectID, &currentID)
		if err != nil {
			return nil, fmt.Errorf(msgChildFolderListFail, err)
		}

		for _, child := range children {
			if _, exists := folderSeen[child.ID]; exists {
				continue
			}
			folderSeen[child.ID] = struct{}{}
			queue = append(queue, child.ID)

			childPrefix, err := folderPrefixForDelete(child.S3Prefix)
			if err != nil {
				return nil, err
			}
			if _, exists := prefixSeen[childPrefix]; exists {
				continue
			}
			prefixSeen[childPrefix] = struct{}{}
			prefixes = append(prefixes, childPrefix)
		}
	}

	return prefixes, nil
}

func (h *FileHandler) deleteFolderPrefixObjects(ctx context.Context, bucketName, prefix string) error {
	if err := h.s3Client.DeleteFolder(ctx, bucketName, prefix+"/"); err != nil {
		return err
	}
	if err := h.s3Client.DeleteObject(ctx, bucketName, prefix); err != nil {
		return err
	}

	return nil
}

func (h *FileHandler) verifyFolderDeletion(
	ctx context.Context,
	projectID uuid.UUID,
	bucketName string,
	folderID uuid.UUID,
	prefixes []string,
) error {
	if _, err := h.folderRepo.GetFolder(ctx, folderID); err == nil {
		return folderDeletionVerificationError(msgFolderMetadataStillExist)
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return folderDeletionVerificationError(msgVerifyMetadataRemoval, err)
	}

	for _, prefix := range prefixes {
		remainingRows, err := h.fileRepo.CountByProjectAndPrefix(ctx, projectID, prefix)
		if err != nil {
			return folderDeletionVerificationError(msgVerifyFileMetadata, err)
		}
		if remainingRows > 0 {
			return folderDeletionVerificationError(msgRowsRemainForPrefix, remainingRows, prefix)
		}

		objects, err := h.s3Client.ListObjects(ctx, bucketName, prefix+"/", verificationListObjectLimit)
		if err != nil {
			return folderDeletionVerificationError(msgVerifyS3Cleanup, err)
		}
		if len(objects.Contents) > 0 {
			return folderDeletionVerificationError(msgS3ObjectsRemainPrefix, prefix)
		}
	}

	return nil
}

func folderPrefixForDelete(rawPrefix string) (string, error) {
	prefix := strings.TrimRight(strings.TrimSpace(rawPrefix), "/")
	if prefix == "" {
		return "", fmt.Errorf(msgInvalidFolderPrefix)
	}
	return prefix, nil
}

func folderDeletionVerificationError(format string, args ...interface{}) error {
	return fmt.Errorf(msgFolderDeletionVerifyFail+format, args...)
}

func (h *FileHandler) resolveProjectID(c echo.Context, requestedProjectID string) (uuid.UUID, error) {
	authType := auth.GetAuthType(c)

	if authType == auth.AuthTypeAPIKey {
		keyProjectID, err := auth.GetProjectID(c)
		if err != nil {
			return uuid.Nil, err
		}
		if requestedProjectID == "" {
			return keyProjectID, nil
		}
		requestedID, err := uuid.Parse(requestedProjectID)
		if err != nil {
			return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, msgInvalidProjectID)
		}
		if requestedID != keyProjectID {
			return uuid.Nil, echo.NewHTTPError(http.StatusForbidden, msgAPIKeyScopeDenied)
		}
		return requestedID, nil
	}

	if projectIDPath := c.Param(paramProjectID); projectIDPath != "" {
		pathProjectID, err := uuid.Parse(projectIDPath)
		if err != nil {
			return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, msgInvalidProjectID)
		}

		if requestedProjectID == "" {
			return pathProjectID, nil
		}

		requestedID, err := uuid.Parse(requestedProjectID)
		if err != nil {
			return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, msgInvalidProjectID)
		}
		if requestedID != pathProjectID {
			return uuid.Nil, echo.NewHTTPError(http.StatusForbidden, msgProjectIDMismatchRoute)
		}

		return pathProjectID, nil
	}

	if requestedProjectID != "" {
		projectID, err := uuid.Parse(requestedProjectID)
		if err != nil {
			return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, msgInvalidProjectID)
		}
		return projectID, nil
	}

	return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, msgProjectIDRequired)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func buildChildFolderPrefix(parentPrefix, folderName string) string {
	if parentPrefix == "" {
		return folderName
	}
	return strings.TrimRight(parentPrefix, "/") + "/" + folderName
}

func (h *FileHandler) validateFolderDepth(ctx context.Context, parentID *uuid.UUID) error {
	depth := 0
	currentID := parentID

	for currentID != nil && depth < maxFolderDepth {
		folder, err := h.folderRepo.GetFolder(ctx, *currentID)
		if err != nil {
			return fmt.Errorf(msgFolderNotFound)
		}
		currentID = folder.ParentFolderID
		depth++
	}

	if depth >= maxFolderDepth {
		return fmt.Errorf(msgFolderDepthExceeded, maxFolderDepth)
	}

	return nil
}

func parsePaginationParams(c echo.Context, defaultLimit, defaultOffset int) (limit int, offset int, err error) {
	limit = defaultLimit
	offset = defaultOffset

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		parsedLimit, parseErr := strconv.Atoi(limitStr)
		if parseErr != nil || parsedLimit <= 0 {
			return 0, 0, fmt.Errorf(msgInvalidLimit)
		}

		if parsedLimit > maxPaginationLimit {
			limit = maxPaginationLimit
		} else {
			limit = parsedLimit
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		parsedOffset, parseErr := strconv.Atoi(offsetStr)
		if parseErr != nil || parsedOffset < 0 || parsedOffset > maxPaginationOffset {
			return 0, 0, fmt.Errorf(msgInvalidOffset)
		}

		offset = parsedOffset
	}

	return limit, offset, nil
}
