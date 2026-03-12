package server

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPErrorHandlerFormatsErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	e := echo.New()
	handler := newHTTPErrorHandler(logger)

	tests := []struct {
		name     string
		err      error
		status   int
		response string
	}{
		{
			name:     "http error",
			err:      echo.NewHTTPError(http.StatusBadRequest, "bad request"),
			status:   http.StatusBadRequest,
			response: "{\"message\":\"bad request\"}\n",
		},
		{
			name:     "unknown error",
			err:      errors.New("boom"),
			status:   http.StatusInternalServerError,
			response: "{\"message\":\"internal server error\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			recorder := httptest.NewRecorder()
			context := e.NewContext(request, recorder)

			handler(tt.err, context)

			require.Equal(t, tt.status, recorder.Code)
			assert.Equal(t, tt.response, recorder.Body.String())
		})
	}
}
