package gatekeeper

import (
	"context"

	"linebot-backend/internal/task"
)

type taskCreator interface {
	CreateFromText(ctx context.Context, command task.CreateFromTextCommand) (task.TaskResult, error)
}

// UseCase handles gatekeeper business logic.
type UseCase struct {
	taskUseCase taskCreator
}

// NewUseCase creates a new gatekeeper UseCase.
func NewUseCase(taskUseCase taskCreator) *UseCase {
	return &UseCase{
		taskUseCase: taskUseCase,
	}
}

// CreateTaskCommand holds parameters for creating a task.
type CreateTaskCommand struct {
	Source        string
	Text          string
	ReferenceTime string
	TimeZone      string
	ClientIP      string
}

// CreateTask creates a task from request.
func (u *UseCase) CreateTask(ctx context.Context, command CreateTaskCommand) (task.TaskResult, error) {
	// Build task.CreateFromTextCommand
	taskCommand := task.CreateFromTextCommand{
		Source:        command.Source,
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
