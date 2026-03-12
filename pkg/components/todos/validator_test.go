package todos

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTodoRequestValidate(t *testing.T) {
	request := &CreateTodoRequest{
		ID:   "  abc  ",
		Name: "  Buy groceries  ",
	}

	require.NoError(t, request.Validate())
	assert.Equal(t, "abc", request.ID)
	assert.Equal(t, "Buy groceries", request.Name)

	err := (&CreateTodoRequest{Name: "   "}).Validate()
	httpError, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, "todo must have a name", httpError.Message)
}

func TestUpdateTodoRequestValidate(t *testing.T) {
	request := &UpdateTodoRequest{Status: " complete "}
	require.NoError(t, request.Validate())
	assert.Equal(t, "complete", request.Status)

	err := (&UpdateTodoRequest{Status: "done"}).Validate()
	httpError, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, "invalid status \"done\"", httpError.Message)
}

func TestSearchTodosRequestValidate(t *testing.T) {
	request := &SearchTodosRequest{Name: "  groceries ", Status: " in progress "}
	require.NoError(t, request.Validate())
	assert.Equal(t, "groceries", request.Name)
	assert.Equal(t, "in progress", request.Status)

	err := (&SearchTodosRequest{Status: "bad"}).Validate()
	httpError, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, "invalid status \"bad\"", httpError.Message)
}

func TestValidateTodoID(t *testing.T) {
	id, err := ValidateTodoID("  abc123  ")
	require.NoError(t, err)
	assert.Equal(t, "abc123", id)

	_, err = ValidateTodoID("   ")
	httpError, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, "id is required", httpError.Message)
}
