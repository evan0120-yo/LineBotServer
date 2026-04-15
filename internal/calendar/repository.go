package calendar

import (
	"context"
	"time"

	"github.com/google/uuid"

	"linebot-backend/internal/infra"
)

// Repository handles calendar task persistence.
type Repository struct {
	store *infra.Store
}

// NewRepository creates a new calendar Repository.
func NewRepository(store *infra.Store) *Repository {
	return &Repository{
		store: store,
	}
}

// Create creates a new calendar task in Firestore.
func (r *Repository) Create(ctx context.Context, command CreateCommand) (CalendarTask, error) {
	taskID := uuid.New().String()
	now := time.Now()

	doc := infra.CalendarTaskDoc{
		TaskID:            taskID,
		Source:            command.Source,
		RawText:           command.RawText,
		TaskType:          command.TaskType,
		Operation:         command.Operation,
		Summary:           command.Summary,
		StartAt:           command.StartAt,
		EndAt:             command.EndAt,
		Location:          command.Location,
		MissingFields:     command.MissingFields,
		Status:            "created",
		InternalAppID:     command.InternalAppID,
		InternalBuilderID: command.InternalBuilderID,
		InternalRequest:   command.InternalRequest,
		InternalResponse:  command.InternalResponse,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := r.store.CreateCalendarTask(ctx, doc); err != nil {
		return CalendarTask{}, err
	}

	task := CalendarTask{
		TaskID:        taskID,
		Summary:       command.Summary,
		StartAt:       command.StartAt,
		EndAt:         command.EndAt,
		Location:      command.Location,
		MissingFields: command.MissingFields,
		CreatedAt:     now,
	}

	return task, nil
}
