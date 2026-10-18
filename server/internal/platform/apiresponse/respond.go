package apiresponse

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Reject writes an envelope error reply from an Echo handler.
func Reject(c echo.Context, status int, code, message string) error {
	return c.JSON(status, ErrorEnvelope{Error: ErrorBody{Code: code, Message: message}})
}

// WriteError writes an envelope error reply to a bare http.ResponseWriter. It
// exists for middleware that runs outside Echo's context — the rate limiter in
// particular, whose over-limit response is written by httprate, not by a
// handler.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorEnvelope{Error: ErrorBody{Code: code, Message: message}})
}
