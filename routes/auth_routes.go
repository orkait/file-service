package routes

import (
	"file-service/pkg/auth"
	mailerpkg "file-service/pkg/mailer"
	mailerproviders "file-service/pkg/mailer/providers"
	"file-service/pkg/repository"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

type AuthRoutes struct {
	clientRepo *repository.ClientRepository
	jwtSecret  string
	mailer     *mailerpkg.EmailService
	appBaseURL string
	appName    string
}

func NewAuthRoutes(clientRepo *repository.ClientRepository, jwtSecret string, mailer *mailerpkg.EmailService, appBaseURL string, appName string) *AuthRoutes {
	return &AuthRoutes{
		clientRepo: clientRepo,
		jwtSecret:  jwtSecret,
		mailer:     mailer,
		appBaseURL: strings.TrimRight(appBaseURL, "/"),
		appName:    appName,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func accountPausedResponse(client *repository.ClientRepository, clientID string) map[string]any {
	response := map[string]any{
		"error": "account is paused and scheduled for deletion",
	}
	if c, err := client.GetClientByID(clientID); err == nil && c.ScheduledDeletionAt != nil {
		response["scheduled_deletion_at"] = c.ScheduledDeletionAt.UTC()
	}
	return response
}

func (ar *AuthRoutes) sendWelcomeEmail(name, email string) {
	if ar.mailer == nil {
		return
	}

	safeName := mailerpkg.EscapeHTML(strings.TrimSpace(name))
	safeApp := mailerpkg.EscapeHTML(strings.TrimSpace(ar.appName))
	loginURL := mailerpkg.SanitizeURL(ar.appBaseURL + "/login")
	if loginURL == "" {
		loginURL = ar.appBaseURL + "/login"
	}

	html := fmt.Sprintf(`
		<h2>Welcome to %s</h2>
		<p>Hi %s,</p>
		<p>Your account is ready. You can now log in and manage your assets.</p>
		<p><a href="%s">Go to login</a></p>
	`, safeApp, safeName, loginURL)
	text := fmt.Sprintf("Welcome to %s\n\nHi %s,\nYour account is ready.\nLogin: %s", ar.appName, strings.TrimSpace(name), loginURL)

	_, err := ar.mailer.Send(&mailerproviders.EmailData{
		To:      []string{email},
		Subject: fmt.Sprintf("Welcome to %s", ar.appName),
		HTML:    html,
		Text:    text,
	})
	if err != nil {
		log.Printf("signup email failed for %s: %v", email, err)
	}
}

// Register handles client signup
func (ar *AuthRoutes) Register(c echo.Context) error {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = normalizeEmail(req.Email)
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name, email and password required"})
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process password"})
	}

	client, err := ar.clientRepo.CreateClient(req.Name, req.Email, passwordHash)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "email already exists"})
	}

	accessToken, _ := auth.GenerateAccessToken(client.ID, client.Email, ar.jwtSecret)
	refreshToken, _ := auth.GenerateRefreshToken(client.ID, client.Email, ar.jwtSecret)

	ar.sendWelcomeEmail(client.Name, client.Email)

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"client":        client,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// CreateClient is an alias for registration.
func (ar *AuthRoutes) CreateClient(c echo.Context) error {
	return ar.Register(c)
}

// Login handles client authentication
func (ar *AuthRoutes) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	req.Email = normalizeEmail(req.Email)
	client, err := ar.clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}
	if client.Status != "active" {
		return c.JSON(http.StatusForbidden, accountPausedResponse(ar.clientRepo, client.ID))
	}

	if !auth.VerifyPassword(req.Password, client.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	accessToken, _ := auth.GenerateAccessToken(client.ID, client.Email, ar.jwtSecret)
	refreshToken, _ := auth.GenerateRefreshToken(client.ID, client.Email, ar.jwtSecret)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"client":        client,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// RefreshToken generates new access token from refresh token
func (ar *AuthRoutes) RefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	claims, err := auth.ValidateToken(req.RefreshToken, ar.jwtSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
	}

	client, err := ar.clientRepo.GetClientByID(claims.ClientID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
	}
	if client.Status != "active" {
		return c.JSON(http.StatusForbidden, accountPausedResponse(ar.clientRepo, client.ID))
	}

	email := claims.Email
	if email == "" {
		email = client.Email
	}

	accessToken, _ := auth.GenerateAccessToken(claims.ClientID, email, ar.jwtSecret)

	return c.JSON(http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

// ForgotPassword sends password reset email with a short-lived token.
func (ar *AuthRoutes) ForgotPassword(c echo.Context) error {
	var req struct {
		Email string `json:"email"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	req.Email = normalizeEmail(req.Email)
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email required"})
	}

	if ar.mailer == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "email service not configured"})
	}

	client, err := ar.clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]string{"message": "if this email exists, password reset instructions were sent"})
	}
	if client.Status != "active" {
		return c.JSON(http.StatusOK, map[string]string{"message": "if this email exists, password reset instructions were sent"})
	}

	token, err := auth.GeneratePasswordResetToken(client.ID, client.Email, ar.jwtSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create reset token"})
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", ar.appBaseURL, url.QueryEscape(token))
	safeResetURL := mailerpkg.SanitizeURL(resetURL)
	if safeResetURL == "" {
		safeResetURL = resetURL
	}
	html := fmt.Sprintf(`
		<h2>%s password reset</h2>
		<p>Hi %s,</p>
		<p>We got a request to reset your password.</p>
		<p><a href="%s">Reset Password</a></p>
		<p>This link expires in 1 hour.</p>
	`, mailerpkg.EscapeHTML(ar.appName), mailerpkg.EscapeHTML(client.Name), safeResetURL)
	text := fmt.Sprintf("%s password reset\n\nHi %s,\nReset your password: %s\nThis link expires in 1 hour.", ar.appName, client.Name, resetURL)

	_, err = ar.mailer.Send(&mailerproviders.EmailData{
		To:      []string{client.Email},
		Subject: fmt.Sprintf("%s password reset", ar.appName),
		HTML:    html,
		Text:    text,
	})
	if err != nil {
		log.Printf("forgot password email failed for %s: %v", client.Email, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send reset email"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "if this email exists, password reset instructions were sent"})
}

// ResetPassword resets account password using a valid reset token.
func (ar *AuthRoutes) ResetPassword(c echo.Context) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if strings.TrimSpace(req.Token) == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token and new_password required"})
	}

	claims, err := auth.ValidatePasswordResetToken(req.Token, ar.jwtSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
	}

	client, err := ar.clientRepo.GetClientByID(claims.ClientID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "account not found"})
	}
	if client.Status != "active" {
		return c.JSON(http.StatusForbidden, accountPausedResponse(ar.clientRepo, client.ID))
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process password"})
	}

	if err := ar.clientRepo.UpdateClientPassword(claims.ClientID, passwordHash); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to update password"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "password updated successfully"})
}
