package handler

import (
	"file-service/internal/auth"
	"file-service/internal/domain/share"
	"file-service/internal/repository"
	"file-service/internal/storage/s3"
	"file-service/pkg/token"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	shareLinkTTL = 7 * 24 * time.Hour
)

var shareTokenPattern = regexp.MustCompile("^[a-f0-9]{64}$")

type ShareHandler struct {
	shareRepo   repository.ShareLinkRepository
	fileRepo    repository.FileRepository
	projectRepo repository.ProjectRepository
	s3Client    *s3.Client
}

func NewShareHandler(
	shareRepo repository.ShareLinkRepository,
	fileRepo repository.FileRepository,
	projectRepo repository.ProjectRepository,
	s3Client *s3.Client,
) *ShareHandler {
	return &ShareHandler{
		shareRepo:   shareRepo,
		fileRepo:    fileRepo,
		projectRepo: projectRepo,
		s3Client:    s3Client,
	}
}

type CreateShareLinkResponse struct {
	ShareToken string `json:"share_token"`
	ShareURL   string `json:"share_url"`
}

func (h *ShareHandler) CreateShareLink(c echo.Context) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, err.Error())
	}

	fileID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidFileID)
	}

	_, err = h.fileRepo.GetByID(c.Request().Context(), fileID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgFileNotFound)
	}

	shareToken, err := token.GenerateShareToken()
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgGenerateShareTokenFail)
	}
	hashedShareToken := auth.HashKey(shareToken)

	_, err = h.shareRepo.Create(c.Request().Context(), repositoryShareInput(fileID, hashedShareToken, userID))
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgCreateShareLinkFail)
	}

	return c.JSON(http.StatusCreated, CreateShareLinkResponse{
		ShareToken: shareToken,
		ShareURL:   publicShareURL(shareToken),
	})
}

func (h *ShareHandler) GetDownloadURLByShareToken(c echo.Context) error {
	shareToken := c.Param(paramToken)
	if !isValidShareToken(shareToken) {
		return respondError(c, http.StatusBadRequest, msgInvalidShareToken)
	}

	link, err := h.shareRepo.GetByToken(c.Request().Context(), auth.HashKey(shareToken))
	if err != nil {
		return respondError(c, http.StatusNotFound, msgShareLinkNotFound)
	}
	if time.Since(link.CreatedAt) > shareLinkTTL {
		return respondError(c, http.StatusGone, msgShareTokenExpired)
	}

	fileRecord, err := h.fileRepo.GetByID(c.Request().Context(), link.FileID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgFileNotFound)
	}

	proj, err := h.projectRepo.GetByID(c.Request().Context(), fileRecord.ProjectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}

	downloadURL, err := h.s3Client.GeneratePresignedDownloadURL(proj.S3BucketName, fileRecord.S3Key)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgDownloadURLGenerateFail)
	}

	return c.JSON(http.StatusOK, map[string]string{
		jsonKeyDownloadURL: downloadURL,
		jsonKeyFileName:    fileRecord.Name,
	})
}

func repositoryShareInput(fileID uuid.UUID, shareToken string, userID uuid.UUID) share.CreateShareLinkInput {
	return share.CreateShareLinkInput{
		FileID:    fileID,
		Token:     shareToken,
		CreatedBy: userID,
	}
}

func publicShareURL(token string) string {
	return "/shares/" + token + "/download-url"
}

func isValidShareToken(token string) bool {
	if len(token) != shareTokenLength {
		return false
	}

	return shareTokenPattern.MatchString(token)
}
