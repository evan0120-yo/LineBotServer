package infra

import (
	"encoding/json"
	"io"
	"net/http"
)

// APIError represents an error in API response.
type APIError struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	MissingFields []string `json:"missingFields,omitempty"`
}

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// WriteJSON writes a successful JSON response.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := APIResponse{
		Success: true,
		Data:    data,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// WriteError writes an error JSON response.
func WriteError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	var status int
	var apiError *APIError

	if bizErr := AsBusinessError(err); bizErr != nil {
		status = bizErr.HTTPStatus
		apiError = &APIError{
			Code:          bizErr.Code,
			Message:       bizErr.Message,
			MissingFields: bizErr.MissingFields,
		}
	} else {
		status = http.StatusInternalServerError
		apiError = &APIError{
			Code:    "INTERNAL_ERROR",
			Message: "An internal error occurred",
		}
	}

	w.WriteHeader(status)

	response := APIResponse{
		Success: false,
		Error:   apiError,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// DecodeJSONStrict decodes JSON request body with strict validation.
// Returns error if body exceeds maxBytes or contains unknown fields.
func DecodeJSONStrict(w http.ResponseWriter, r *http.Request, target any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return NewError("INVALID_JSON", "Invalid JSON request: "+err.Error(), http.StatusBadRequest)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return NewError("INVALID_JSON", "Request body contains multiple JSON objects", http.StatusBadRequest)
	}

	return nil
}
