package calendar

import (
	"context"
	"time"

	"linebot-backend/internal/infra"
)

type repository interface {
	Create(ctx context.Context, command CreateCommand) (CalendarTask, error)
	UpdateSyncResult(ctx context.Context, taskID string, result SyncResult) error
}

// UseCase orchestrates calendar task operations.
type UseCase struct {
	service    *Service
	repository repository
	provider   infra.GoogleCalendarProvider
	syncConfig SyncConfig
}

// NewUseCase creates a new calendar UseCase.
func NewUseCase(
	service *Service,
	repository repository,
	provider infra.GoogleCalendarProvider,
	syncConfig SyncConfig,
) *UseCase {
	return &UseCase{
		service:    service,
		repository: repository,
		provider:   provider,
		syncConfig: syncConfig,
	}
}

// Create creates a new calendar task.
// Validates the command, persists to Firestore, syncs Google Calendar when enabled,
// and returns the created task with sync metadata.
func (u *UseCase) Create(ctx context.Context, command CreateCommand) (CalendarTask, error) {
	if err := u.service.ValidateCreate(command); err != nil {
		return CalendarTask{}, err
	}

	if u.syncConfig.Enabled {
		command.CalendarSyncStatus = CalendarSyncStatusPending
	} else {
		command.CalendarSyncStatus = CalendarSyncStatusNotEnabled
	}

	task, err := u.repository.Create(ctx, command)
	if err != nil {
		return CalendarTask{}, err
	}

	if !u.syncConfig.Enabled {
		return task, nil
	}

	return u.syncGoogleCalendar(ctx, task, command)
}

func (u *UseCase) syncGoogleCalendar(ctx context.Context, task CalendarTask, command CreateCommand) (CalendarTask, error) {
	if u.provider == nil {
		result := SyncResult{
			CalendarSyncStatus: CalendarSyncStatusFailed,
			GoogleCalendarID:   u.syncConfig.CalendarID,
			CalendarSyncError:  "google calendar provider is not configured",
		}
		if err := u.repository.UpdateSyncResult(ctx, task.TaskID, result); err != nil {
			return CalendarTask{}, err
		}
		applySyncResult(&task, result)
		return task, nil
	}

	event, err := u.provider.CreateEvent(ctx, infra.GoogleCalendarCreateEventCommand{
		CalendarID: u.syncConfig.CalendarID,
		Summary:    command.Summary,
		StartAt:    command.StartAt,
		EndAt:      command.EndAt,
		TimeZone:   u.syncConfig.TimeZone,
		Location:   command.Location,
	})
	if err != nil {
		result := SyncResult{
			CalendarSyncStatus: CalendarSyncStatusFailed,
			GoogleCalendarID:   u.syncConfig.CalendarID,
			CalendarSyncError:  err.Error(),
		}
		if updateErr := u.repository.UpdateSyncResult(ctx, task.TaskID, result); updateErr != nil {
			return CalendarTask{}, updateErr
		}
		applySyncResult(&task, result)
		return task, nil
	}

	now := time.Now()
	result := SyncResult{
		CalendarSyncStatus:     CalendarSyncStatusSynced,
		GoogleCalendarID:       event.CalendarID,
		GoogleCalendarEventID:  event.EventID,
		GoogleCalendarHTMLLink: event.HTMLLink,
		CalendarSyncedAt:       &now,
	}
	if err := u.repository.UpdateSyncResult(ctx, task.TaskID, result); err != nil {
		return CalendarTask{}, err
	}

	applySyncResult(&task, result)
	return task, nil
}

func applySyncResult(task *CalendarTask, result SyncResult) {
	task.CalendarSyncStatus = result.CalendarSyncStatus
	task.GoogleCalendarID = result.GoogleCalendarID
	task.GoogleCalendarEventID = result.GoogleCalendarEventID
	task.GoogleCalendarHTMLLink = result.GoogleCalendarHTMLLink
	task.CalendarSyncError = result.CalendarSyncError
	task.CalendarSyncedAt = result.CalendarSyncedAt
}
