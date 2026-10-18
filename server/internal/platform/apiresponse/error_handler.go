package apiresponse

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorHandler is Echo's HTTPErrorHandler. It exists so that every failure —
// including the ones Echo answers on its own (unknown route, wrong method,
// oversized body) and the ones Recover turns a panic into — is the same
// envelope shape as the errors handlers produce. Without it, those would come
// back as Echo's default {"message": "..."} and the client would have to handle
// a second shape forever.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	status := http.StatusInternalServerError
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		status = httpErr.Code
	}
	code, message := CodeForStatus(status)

	// A 5xx here is an error we did not anticipate — a panic, or a handler
	// returning a bare error. Log it; otherwise the only trace of it is a user
	// seeing "something went wrong".
	if status >= http.StatusInternalServerError {
		slog.Error("unhandled request error",
			"status", status, "method", c.Request().Method,
			"path", c.Request().URL.Path, "error", err)
	}

	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(status)
		return
	}
	_ = c.JSON(status, ErrorEnvelope{Error: ErrorBody{Code: code, Message: message}})
}
