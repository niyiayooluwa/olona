// Package apiresponse holds the API response envelope shared by every domain.
//
// docs/server/HTTP Response Contract.md is the specification. Every reply is
// either a success envelope — a domain type with a "data" field — or
// ErrorEnvelope, which carries "error" and no "data". Because success and
// failure are different Go types, "exactly one of data / error" is enforced by
// the compiler rather than by convention.
package apiresponse

// FieldError is one field-specific validation failure inside details.
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// ErrorBody is the machine-readable error payload. Details is omitted when
// nothing is field-specific, and is never set on an auth failure: a field-level
// "not_found" on an email would confirm whether that address has an account.
type ErrorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// ErrorEnvelope is the reply shape for every failure. It deliberately has no
// data field.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}
