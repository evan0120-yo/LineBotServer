package infra

import (
	"errors"
	"net/http"
)

// BusinessError represents a domain-level error with error code and HTTP status.
type BusinessError struct {
	Code          string
	Message       string
	HTTPStatus    int
	MissingFields []string
}

func (e *BusinessError) Error() string {
	return e.Message
}

// NewError creates a new BusinessError.
func NewError(code, message string, status int) error {
	return &BusinessError{
		Code:       code,
		Message:    message,
		HTTPStatus: status,
	}
}

// NewErrorWithMissingFields creates a BusinessError carrying missing field details.
func NewErrorWithMissingFields(code, message string, status int, missingFields []string) error {
	return &BusinessError{
		Code:          code,
		Message:       message,
		HTTPStatus:    status,
		MissingFields: append([]string(nil), missingFields...),
	}
}

// AsBusinessError attempts to cast an error to BusinessError.
// Returns nil if the error is not a BusinessError.
func AsBusinessError(err error) *BusinessError {
	var bizErr *BusinessError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return nil
}

// Error codes
const (
	ErrCodeTextRequired                 = "TEXT_REQUIRED"
	ErrCodeInternalExtractionIncomplete = "INTERNAL_EXTRACTION_INCOMPLETE"
	ErrCodeTaskTypeUnsupported          = "TASK_TYPE_UNSUPPORTED"
	ErrCodeOperationUnsupported         = "OPERATION_UNSUPPORTED"
	ErrCodeInternalGRPCError            = "INTERNAL_GRPC_ERROR"
	ErrCodeFirestoreWriteError          = "FIRESTORE_WRITE_ERROR"
)

// NewTextRequiredError creates TEXT_REQUIRED error.
func NewTextRequiredError() error {
	return NewError(ErrCodeTextRequired, "text is required", http.StatusBadRequest)
}

// NewInternalExtractionIncompleteError creates INTERNAL_EXTRACTION_INCOMPLETE error.
func NewInternalExtractionIncompleteError(missingFields []string) error {
	message := "Internal extraction did not return required fields"
	return NewErrorWithMissingFields(ErrCodeInternalExtractionIncomplete, message, http.StatusBadRequest, missingFields)
}

// NewTaskTypeUnsupportedError creates TASK_TYPE_UNSUPPORTED error.
func NewTaskTypeUnsupportedError(taskType string) error {
	message := "Task type " + taskType + " is not supported"
	return NewError(ErrCodeTaskTypeUnsupported, message, http.StatusBadRequest)
}

// NewOperationUnsupportedError creates OPERATION_UNSUPPORTED error.
func NewOperationUnsupportedError(operation string) error {
	message := "Operation " + operation + " is not supported in the first version"
	return NewError(ErrCodeOperationUnsupported, message, http.StatusBadRequest)
}

// NewInternalGRPCError creates INTERNAL_GRPC_ERROR error.
func NewInternalGRPCError(err error) error {
	return NewError(ErrCodeInternalGRPCError, "Internal gRPC call failed: "+err.Error(), http.StatusInternalServerError)
}

// NewFirestoreWriteError creates FIRESTORE_WRITE_ERROR error.
func NewFirestoreWriteError(err error) error {
	return NewError(ErrCodeFirestoreWriteError, "Firestore write failed: "+err.Error(), http.StatusInternalServerError)
}
