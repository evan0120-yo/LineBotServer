package task

import "linebot-backend/internal/infra"

// Service handles task dispatch validation.
type Service struct{}

// NewService creates a new task Service.
func NewService() *Service {
	return &Service{}
}

// ValidateTaskType validates if the taskType is supported.
func (s *Service) ValidateTaskType(taskType string) error {
	for _, supported := range SupportedTaskTypes() {
		if taskType == supported {
			return nil
		}
	}
	return infra.NewTaskTypeUnsupportedError(taskType)
}

// ValidateOperation validates if the operation is supported.
func (s *Service) ValidateOperation(operation string) error {
	switch operation {
	case "create", "query", "delete", "update":
		return nil
	default:
		return infra.NewOperationUnsupportedError(operation)
	}
}
