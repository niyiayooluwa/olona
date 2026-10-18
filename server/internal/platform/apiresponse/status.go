package apiresponse

import "net/http"

// CodeForStatus maps a status produced by the framework itself — an unknown
// route, a wrong method, an oversized body, a panic — to a wire code + message.
// Handlers classify their own domain errors; this covers the errors that never
// reach a handler.
func CodeForStatus(status int) (code, message string) {
	switch status {
	case http.StatusNotFound:
		return "not_found", "That item doesn't exist."
	case http.StatusMethodNotAllowed:
		return "method_not_allowed", "That method isn't allowed on this URL."
	case http.StatusRequestEntityTooLarge:
		return "payload_too_large", "That request body is too large."
	case http.StatusTooManyRequests:
		return "rate_limited", "Too many requests. Please slow down."
	case http.StatusInternalServerError:
		return "internal_error", "Something went wrong on our end."
	}
	if status >= http.StatusInternalServerError {
		return "internal_error", "Something went wrong on our end."
	}
	return "invalid_request", "That request couldn't be processed."
}
