package routes

import (
	mailerpkg "file-service/pkg/mailer"
	mailerproviders "file-service/pkg/mailer/providers"
	"file-service/pkg/repository"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type MemberRoutes struct {
	memberRepo  *repository.MemberRepository
	projectRepo *repository.ProjectRepository
	clientRepo  *repository.ClientRepository
	mailer      *mailerpkg.EmailService
	appBaseURL  string
	appName     string
}

func NewMemberRoutes(memberRepo *repository.MemberRepository, projectRepo *repository.ProjectRepository, clientRepo *repository.ClientRepository, mailer *mailerpkg.EmailService, appBaseURL string, appName string) *MemberRoutes {
	return &MemberRoutes{
		memberRepo:  memberRepo,
		projectRepo: projectRepo,
		clientRepo:  clientRepo,
		mailer:      mailer,
		appBaseURL:  strings.TrimRight(appBaseURL, "/"),
		appName:     appName,
	}
}

func (mr *MemberRoutes) sendInviteEmail(inviterName string, inviteeEmail string, projectName string, role string, projectID string) {
	if mr.mailer == nil {
		return
	}

	projectURL := mailerpkg.SanitizeURL(fmt.Sprintf("%s/projects/%s", mr.appBaseURL, projectID))
	if projectURL == "" {
		projectURL = fmt.Sprintf("%s/projects/%s", mr.appBaseURL, projectID)
	}

	html := fmt.Sprintf(`
		<h2>Project invite - %s</h2>
		<p>You were invited by %s to join project <strong>%s</strong> as <strong>%s</strong>.</p>
		<p><a href="%s">Open project</a></p>
	`, mailerpkg.EscapeHTML(mr.appName), mailerpkg.EscapeHTML(inviterName), mailerpkg.EscapeHTML(projectName), mailerpkg.EscapeHTML(role), projectURL)
	text := fmt.Sprintf("%s project invite\n\nYou were invited by %s to join %s as %s.\nOpen project: %s", mr.appName, inviterName, projectName, role, projectURL)

	_, err := mr.mailer.Send(&mailerproviders.EmailData{
		To:      []string{inviteeEmail},
		Subject: fmt.Sprintf("%s invited you to %s", inviterName, projectName),
		HTML:    html,
		Text:    text,
	})
	if err != nil {
		log.Printf("invite email failed for %s: %v", inviteeEmail, err)
	}
}

// InviteMember invites a client to a project
func (mr *MemberRoutes) InviteMember(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.Param("project_id")

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	req.Email = normalizeEmail(req.Email)
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email required"})
	}

	if req.Role == "" {
		req.Role = "viewer"
	}

	if req.Role != "owner" && req.Role != "editor" && req.Role != "viewer" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role (owner, editor, viewer)"})
	}

	hasAccess, role, err := mr.memberRepo.CheckMemberAccess(projectID, clientID)
	if err != nil || !hasAccess {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "project not found or access denied"})
	}

	if role != "owner" {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only project owner can invite members"})
	}

	invitedClient, err := mr.clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found with that email"})
	}
	if invitedClient.Status != "active" {
		return c.JSON(http.StatusConflict, map[string]string{"error": "cannot invite a paused account"})
	}

	member, err := mr.memberRepo.InviteMember(projectID, invitedClient.ID, clientID, req.Role)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "member already exists or failed to invite"})
	}

	inviterName := "Project Owner"
	if inviter, lookupErr := mr.clientRepo.GetClientByID(clientID); lookupErr == nil && strings.TrimSpace(inviter.Name) != "" {
		inviterName = inviter.Name
	}

	projectName := "your project"
	if project, lookupErr := mr.projectRepo.GetProjectByID(projectID, clientID); lookupErr == nil && strings.TrimSpace(project.Name) != "" {
		projectName = project.Name
	}

	mr.sendInviteEmail(inviterName, invitedClient.Email, projectName, req.Role, projectID)

	return c.JSON(http.StatusCreated, member)
}

// GetMembers retrieves all members of a project
func (mr *MemberRoutes) GetMembers(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.Param("project_id")

	hasAccess, _, err := mr.memberRepo.CheckMemberAccess(projectID, clientID)
	if err != nil || !hasAccess {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "project not found or access denied"})
	}

	members, err := mr.memberRepo.GetProjectMembers(projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get members"})
	}

	return c.JSON(http.StatusOK, members)
}

// RemoveMember removes a member from a project
func (mr *MemberRoutes) RemoveMember(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.Param("project_id")
	memberID := c.Param("member_id")

	hasAccess, role, err := mr.memberRepo.CheckMemberAccess(projectID, clientID)
	if err != nil || !hasAccess || role != "owner" {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only project owner can remove members"})
	}

	err = mr.memberRepo.RemoveMember(projectID, memberID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "member removed"})
}
