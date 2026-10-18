package apiresponse

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// newErrorCtx builds an echo context wired to a recorder so a test can drive
// ErrorHandler directly.
func newErrorCtx(t *testing.T, method string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/nope", nil)
	return e.NewContext(req, rec), rec
}

// TestErrorHandlerCoversFrameworkErrors locks in that the errors Echo produces
// on its own are the same envelope shape as the ones handlers produce. Before
// this handler existed, an unknown route came back as {"message": "Not Found"}
// — a second shape the client would have to branch on forever.
func TestErrorHandlerCoversFrameworkErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		want     int
		wantCode string
	}{
		{"unknown route", echo.NewHTTPError(http.StatusNotFound, "Not Found"), http.StatusNotFound, "not_found"},
		{"wrong method", echo.NewHTTPError(http.StatusMethodNotAllowed, "Method Not Allowed"), http.StatusMethodNotAllowed, "method_not_allowed"},
		{"oversized body", echo.NewHTTPError(http.StatusRequestEntityTooLarge, "Request Entity Too Large"), http.StatusRequestEntityTooLarge, "payload_too_large"},
		{"unexpected bare error", errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newErrorCtx(t, http.MethodGet)
			ErrorHandler(tt.err, c)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want, rec.Body.String())
			}
			var got ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not the error envelope: %v (body: %s)", err, rec.Body.String())
			}
			if got.Error.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if got.Error.Message == "" {
				t.Fatal("message must not be empty")
			}
		})
	}
}

// TestErrorHandlerDoesNotLeakInternalDetail ensures a panic's underlying error
// never reaches the client.
func TestErrorHandlerDoesNotLeakInternalDetail(t *testing.T) {
	c, rec := newErrorCtx(t, http.MethodGet)
	ErrorHandler(errors.New("pq: password authentication failed for user \"postgres\" (SQLSTATE 28P01)"), c)

	body := rec.Body.String()
	for _, leak := range []string{"postgres", "28P01", "SQLSTATE"} {
		if strings.Contains(body, leak) {
			t.Fatalf("response body leaked %q: %s", leak, body)
		}
	}
}

// TestErrorHandlerLeavesCommittedResponseAlone guards against writing a second
// body after a handler has already replied.
func TestErrorHandlerLeavesCommittedResponseAlone(t *testing.T) {
	c, rec := newErrorCtx(t, http.MethodGet)
	_ = c.String(http.StatusOK, "already sent")

	ErrorHandler(echo.NewHTTPError(http.StatusNotFound), c)

	if got := rec.Body.String(); got != "already sent" {
		t.Fatalf("committed response was overwritten: %q", got)
	}
}
