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
	syncStatus := command.CalendarSyncStatus
	if syncStatus == "" {
		syncStatus = CalendarSyncStatusNotEnabled
	}

	doc := infra.CalendarTaskDoc{
		TaskID:             taskID,
		Source:             command.Source,
		RawText:            command.RawText,
		TaskType:           command.TaskType,
		Operation:          command.Operation,
		Summary:            command.Summary,
		StartAt:            command.StartAt,
		EndAt:              command.EndAt,
		Location:           command.Location,
		MissingFields:      command.MissingFields,
		Status:             TaskStatusCreated,
		CalendarSyncStatus: syncStatus,
		InternalAppID:      command.InternalAppID,
		InternalBuilderID:  command.InternalBuilderID,
		InternalRequest:    command.InternalRequest,
		InternalResponse:   command.InternalResponse,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := r.store.CreateCalendarTask(ctx, doc); err != nil {
		return CalendarTask{}, err
	}

	task := CalendarTask{
		TaskID:             taskID,
		Summary:            command.Summary,
		StartAt:            command.StartAt,
		EndAt:              command.EndAt,
		Location:           command.Location,
		MissingFields:      command.MissingFields,
		CalendarSyncStatus: syncStatus,
		CreatedAt:          now,
	}

	return task, nil
}

// UpdateSyncResult updates Google Calendar sync metadata for a calendar task.
func (r *Repository) UpdateSyncResult(ctx context.Context, taskID string, result SyncResult) error {
	return r.store.UpdateCalendarTaskSyncResult(ctx, taskID, infra.CalendarTaskSyncResult{
		CalendarSyncStatus:     result.CalendarSyncStatus,
		GoogleCalendarID:       result.GoogleCalendarID,
		GoogleCalendarEventID:  result.GoogleCalendarEventID,
		GoogleCalendarHTMLLink: result.GoogleCalendarHTMLLink,
		CalendarSyncError:      result.CalendarSyncError,
		CalendarSyncedAt:       result.CalendarSyncedAt,
	})
}
