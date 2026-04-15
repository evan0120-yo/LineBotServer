package calendar

import (
	"strings"

	"linebot-backend/internal/infra"
)

// Service handles calendar task business logic.
type Service struct{}

// NewService creates a new calendar Service.
func NewService() *Service {
	return &Service{}
}

// ValidateCreate validates a calendar create command.
// Returns error if required fields (summary, startAt, endAt) are missing.
// location is optional and does not cause validation failure.
func (s *Service) ValidateCreate(command CreateCommand) error {
	var missingFields []string

	if strings.TrimSpace(command.Summary) == "" {
		missingFields = append(missingFields, "summary")
	}

	if strings.TrimSpace(command.StartAt) == "" {
		missingFields = append(missingFields, "startAt")
	}

	if strings.TrimSpace(command.EndAt) == "" {
		missingFields = append(missingFields, "endAt")
	}

	if len(missingFields) > 0 {
		return infra.NewInternalExtractionIncompleteError(missingFields)
	}

	return nil
}
