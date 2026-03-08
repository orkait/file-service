package routes

import (
	"file-service/pkg/repository"
	"net/http"

	"github.com/labstack/echo/v4"
)

type ProjectRoutes struct {
	projectRepo *repository.ProjectRepository
}

func NewProjectRoutes(projectRepo *repository.ProjectRepository) *ProjectRoutes {
	return &ProjectRoutes{projectRepo: projectRepo}
}

// CreateProject creates a new project
func (pr *ProjectRoutes) CreateProject(c echo.Context) error {
	clientID := c.Get("client_id").(string)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project name required"})
	}

	project, err := pr.projectRepo.CreateProject(clientID, req.Name, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create project"})
	}

	return c.JSON(http.StatusCreated, project)
}

// GetProjects retrieves all projects for authenticated client
func (pr *ProjectRoutes) GetProjects(c echo.Context) error {
	clientID := c.Get("client_id").(string)

	projects, err := pr.projectRepo.GetProjectsByClientID(clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get projects"})
	}

	return c.JSON(http.StatusOK, projects)
}

// GetProject retrieves a specific project
func (pr *ProjectRoutes) GetProject(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	projectID := c.Param("id")

	project, err := pr.projectRepo.GetProjectByID(projectID, clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "project not found"})
	}

	return c.JSON(http.StatusOK, project)
}
