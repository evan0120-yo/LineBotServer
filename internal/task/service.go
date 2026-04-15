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

// ValidateOperation validates if the operation is supported in the first version.
// First version only supports "create" operation.
func (s *Service) ValidateOperation(operation string) error {
	if operation != "create" {
		return infra.NewOperationUnsupportedError(operation)
	}

	return nil
}
