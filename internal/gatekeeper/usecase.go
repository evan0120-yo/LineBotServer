package gatekeeper

import (
	"context"

	"linebot-backend/internal/task"
)

// UseCase handles gatekeeper business logic.
type UseCase struct {
	taskUseCase *task.UseCase
}

// NewUseCase creates a new gatekeeper UseCase.
func NewUseCase(taskUseCase *task.UseCase) *UseCase {
	return &UseCase{
		taskUseCase: taskUseCase,
	}
}

// CreateTaskCommand holds parameters for creating a task from the REST API.
type CreateTaskCommand struct {
	Text          string
	ReferenceTime string
	TimeZone      string
	ClientIP      string
}

// CreateTask creates a task from REST API request.
func (u *UseCase) CreateTask(ctx context.Context, command CreateTaskCommand) (task.TaskResult, error) {
	// Build task.CreateFromTextCommand
	taskCommand := task.CreateFromTextCommand{
		Source:        "rest",
		Text:          command.Text,
		ReferenceTime: command.ReferenceTime,
		TimeZone:      command.TimeZone,
		ClientIP:      command.ClientIP,
	}

	// Call task usecase
	result, err := u.taskUseCase.CreateFromText(ctx, taskCommand)
	if err != nil {
		return task.TaskResult{}, err
	}

	return result, nil
}
