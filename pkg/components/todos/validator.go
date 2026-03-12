package todos

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type CreateTodoRequest struct {
	ID   string `json:"id" form:"id"`
	Name string `json:"name" form:"name"`
}

func (r *CreateTodoRequest) Validate() error {
	r.ID = strings.TrimSpace(r.ID)
	r.Name = strings.TrimSpace(r.Name)

	if r.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "todo must have a name")
	}

	return nil
}

type UpdateTodoRequest struct {
	Status string `json:"status" form:"status"`
}

func (r *UpdateTodoRequest) Validate() error {
	r.Status = strings.TrimSpace(r.Status)
	if r.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "status is required")
	}

	if _, ok := TodoStatusMap[r.Status]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid status %q", r.Status))
	}

	return nil
}

type SearchTodosRequest struct {
	Name   string
	Status string
}

func (r *SearchTodosRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Status = strings.TrimSpace(r.Status)

	if r.Status == "" {
		return nil
	}

	if _, ok := TodoStatusMap[r.Status]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid status %q", r.Status))
	}

	return nil
}

func ValidateTodoID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}

	return id, nil
}
