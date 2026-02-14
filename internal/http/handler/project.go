package handler

import (
	"errors"
	"file-service/internal/auth"
	"file-service/internal/domain/project"
	"file-service/internal/types"
	apperrors "file-service/pkg/errors"
	"file-service/pkg/validator"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ProjectHandler struct {
	projectRepo   ProjectRepository
	memberRepo    ProjectMemberRepository
	userRepo      UserGetter
	s3Client      StorageOperations
	bucketCreator BucketCreator
	bucketRegion  string
	auditLogger   types.AuditLogger
}

func NewProjectHandler(
	projectRepo ProjectRepository,
	memberRepo ProjectMemberRepository,
	userRepo UserGetter,
	s3Client StorageOperations,
	bucketCreator BucketCreator,
	bucketRegion string,
	auditLogger types.AuditLogger,
) *ProjectHandler {
	return &ProjectHandler{
		projectRepo:   projectRepo,
		memberRepo:    memberRepo,
		userRepo:      userRepo,
		s3Client:      s3Client,
		bucketCreator: bucketCreator,
		bucketRegion:  bucketRegion,
		auditLogger:   auditLogger,
	}
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

func (h *ProjectHandler) CreateProject(c echo.Context) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, err.Error())
	}

	clientID, err := auth.GetClientID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, err.Error())
	}

	var req CreateProjectRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}
	req.Name = strings.TrimSpace(req.Name)

	if err := validator.ProjectName(req.Name); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	proj, err := h.projectRepo.Create(c.Request().Context(), project.CreateProjectInput{
		ClientID:  clientID,
		Name:      req.Name,
		IsDefault: false,
	})

	if err != nil {
		c.Logger().Errorf("Failed to create project: %v", err)
		return respondError(c, http.StatusInternalServerError, msgCreateProjectFail)
	}

	if err := h.bucketCreator.CreateBucket(c.Request().Context(), proj.S3BucketName, h.bucketRegion); err != nil {
		if deleteErr := h.projectRepo.Delete(c.Request().Context(), proj.ID); deleteErr != nil {
			c.Logger().Errorf("Failed to rollback project %s after bucket creation failure: %v", proj.ID, deleteErr)
		}
		return respondError(c, http.StatusInternalServerError, msgCreateS3BucketFail)
	}

	if _, err := h.memberRepo.AddMember(c.Request().Context(), project.AddMemberInput{
		ProjectID: proj.ID,
		UserID:    userID,
		Role:      project.RoleAdmin,
		InvitedBy: userID,
	}); err != nil {
		if deleteErr := h.s3Client.DeleteBucket(c.Request().Context(), proj.S3BucketName); deleteErr != nil {
			c.Logger().Errorf("Failed to delete bucket %s during rollback: %v", proj.S3BucketName, deleteErr)
		}
		if deleteErr := h.projectRepo.Delete(c.Request().Context(), proj.ID); deleteErr != nil {
			c.Logger().Errorf("Failed to rollback project %s after member creation failure: %v", proj.ID, deleteErr)
		}
		return respondError(c, http.StatusInternalServerError, msgInitProjectMembersFail)
	}

	return c.JSON(http.StatusCreated, proj)
}

func (h *ProjectHandler) ListProjects(c echo.Context) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, err.Error())
	}

	projects, err := h.projectRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		c.Logger().Errorf("Failed to list projects for user %s: %v", userID, err)
		return respondError(c, http.StatusInternalServerError, msgListProjectsFail)
	}

	return c.JSON(http.StatusOK, projects)
}

func (h *ProjectHandler) GetProject(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectID)
	}

	proj, err := h.projectRepo.GetByID(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}

	// Client isolation is enforced by RBAC middleware
	// which checks project membership before reaching this handler

	return c.JSON(http.StatusOK, proj)
}

func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectID)
	}

	proj, err := h.projectRepo.GetByID(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}

	// Client isolation is enforced by RBAC middleware
	// which checks project membership and admin role before reaching this handler

	if err := h.s3Client.DeleteFolder(c.Request().Context(), proj.S3BucketName, ""); err != nil {
		c.Logger().Errorf("Failed to delete S3 objects in bucket %s: %v", proj.S3BucketName, err)
	}

	if err := h.s3Client.DeleteBucket(c.Request().Context(), proj.S3BucketName); err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "project", &projectID, "delete", err)
		}
		return respondError(c, http.StatusInternalServerError, msgDeleteS3BucketFail)
	}

	if err := h.projectRepo.Delete(c.Request().Context(), projectID); err != nil {
		c.Logger().Errorf("Failed to delete project %s: %v", projectID, err)
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "project", &projectID, "delete", err)
		}
		return respondError(c, http.StatusInternalServerError, msgDeleteProjectFail)
	}

	// Log successful project deletion
	if h.auditLogger != nil {
		metadata := map[string]any{
			"project_id":   projectID.String(),
			"project_name": proj.Name,
			"bucket_name":  proj.S3BucketName,
		}
		_ = h.auditLogger.LogFromContext(c, "project", &projectID, "delete", "success", metadata)
	}

	return respondMessage(c, http.StatusOK, msgProjectDeleted)
}

type AddMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *ProjectHandler) AddMember(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param(paramProjectID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectID)
	}

	invitedBy, err := auth.GetUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, err.Error())
	}

	var req AddMemberRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.TrimSpace(req.Role)
	if err := validator.Email(req.Email); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	role := project.Role(req.Role)
	if err := role.Validate(); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	invitee, err := h.userRepo.GetByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgUserNotFound)
	}

	proj, err := h.projectRepo.GetByID(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgProjectNotFound)
	}

	if invitee.ClientID != proj.ClientID {
		return respondError(c, http.StatusForbidden, msgCrossClientMemberDenied)
	}

	member, err := h.memberRepo.AddMember(c.Request().Context(), project.AddMemberInput{
		ProjectID: projectID,
		UserID:    invitee.ID,
		Role:      role,
		InvitedBy: invitedBy,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			return respondError(c, http.StatusConflict, msgAddMemberFail)
		}
		c.Logger().Errorf("Failed to add member %s to project %s: %v", invitee.ID, projectID, err)
		return respondError(c, http.StatusInternalServerError, msgAddMemberFail)
	}

	return c.JSON(http.StatusCreated, member)
}

func (h *ProjectHandler) ListMembers(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param(paramProjectID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectID)
	}

	members, err := h.memberRepo.GetMembers(c.Request().Context(), projectID)
	if err != nil {
		c.Logger().Errorf("Failed to list members for project %s: %v", projectID, err)
		return respondError(c, http.StatusInternalServerError, msgListMembersFail)
	}

	return c.JSON(http.StatusOK, members)
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (h *ProjectHandler) UpdateMemberRole(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param(paramProjectID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectID)
	}

	userID, err := uuid.Parse(c.Param(paramUserID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidUserID)
	}

	var req UpdateMemberRoleRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}
	req.Role = strings.TrimSpace(req.Role)

	role := project.Role(req.Role)
	if err := role.Validate(); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if role != project.RoleAdmin {
		if err := h.ensureNotLastAdmin(c, projectID, userID); err != nil {
			return err
		}
	}

	if err := h.memberRepo.UpdateMemberRole(c.Request().Context(), project.UpdateMemberRoleInput{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
	}); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return respondError(c, http.StatusNotFound, msgUpdateMemberFail)
		}
		c.Logger().Errorf("Failed to update member role for user %s in project %s: %v", userID, projectID, err)
		return respondError(c, http.StatusInternalServerError, msgUpdateMemberFail)
	}

	return respondMessage(c, http.StatusOK, msgMemberRoleUpdated)
}

func (h *ProjectHandler) RemoveMember(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param(paramProjectID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectID)
	}

	userID, err := uuid.Parse(c.Param(paramUserID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidUserID)
	}

	if err := h.ensureNotLastAdmin(c, projectID, userID); err != nil {
		return err
	}

	if err := h.memberRepo.RemoveMember(c.Request().Context(), projectID, userID); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return respondError(c, http.StatusNotFound, msgRemoveMemberFail)
		}
		c.Logger().Errorf("Failed to remove member %s from project %s: %v", userID, projectID, err)
		return respondError(c, http.StatusInternalServerError, msgRemoveMemberFail)
	}

	return respondMessage(c, http.StatusOK, msgMemberRemoved)
}

func (h *ProjectHandler) ensureNotLastAdmin(c echo.Context, projectID, userID uuid.UUID) error {
	member, err := h.memberRepo.GetMember(c.Request().Context(), projectID, userID)
	if err != nil || member.Role != project.RoleAdmin {
		return nil
	}

	adminCount, err := h.memberRepo.CountAdminsByProject(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgCheckProjectMembersFail)
	}

	if adminCount <= minProjectAdminCount {
		return respondError(c, http.StatusConflict, msgLastAdminConstraint)
	}

	return nil
}
