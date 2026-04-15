package calendar

import (
	"context"
	"errors"
	"testing"

	"linebot-backend/internal/infra"
)

type fakeRepository struct {
	createCommand CreateCommand
	createResult  CalendarTask
	createErr     error

	updateTaskID string
	updateResult SyncResult
	updateErr    error
	updateCalled bool
}

func (r *fakeRepository) Create(_ context.Context, command CreateCommand) (CalendarTask, error) {
	r.createCommand = command
	if r.createErr != nil {
		return CalendarTask{}, r.createErr
	}
	if r.createResult.TaskID == "" {
		r.createResult = CalendarTask{
			TaskID:             "task-1",
			Summary:            command.Summary,
			StartAt:            command.StartAt,
			EndAt:              command.EndAt,
			Location:           command.Location,
			MissingFields:      command.MissingFields,
			CalendarSyncStatus: command.CalendarSyncStatus,
		}
	}
	return r.createResult, nil
}

func (r *fakeRepository) UpdateSyncResult(_ context.Context, taskID string, result SyncResult) error {
	r.updateCalled = true
	r.updateTaskID = taskID
	r.updateResult = result
	return r.updateErr
}

type fakeGoogleCalendarProvider struct {
	command infra.GoogleCalendarCreateEventCommand
	result  infra.GoogleCalendarEventResult
	err     error
	called  bool
}

func (p *fakeGoogleCalendarProvider) CreateEvent(
	_ context.Context,
	command infra.GoogleCalendarCreateEventCommand,
) (infra.GoogleCalendarEventResult, error) {
	p.called = true
	p.command = command
	if p.err != nil {
		return infra.GoogleCalendarEventResult{}, p.err
	}
	return p.result, nil
}

func TestUseCaseCreateWhenGoogleCalendarDisabled(t *testing.T) {
	repo := &fakeRepository{}
	provider := &fakeGoogleCalendarProvider{}
	useCase := NewUseCase(NewService(), repo, provider, SyncConfig{Enabled: false})

	result, err := useCase.Create(context.Background(), validCreateCommand())
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if provider.called {
		t.Fatal("provider should not be called when sync is disabled")
	}
	if repo.updateCalled {
		t.Fatal("sync result should not be updated when sync is disabled")
	}
	if repo.createCommand.CalendarSyncStatus != CalendarSyncStatusNotEnabled {
		t.Fatalf("create sync status = %q, want %q", repo.createCommand.CalendarSyncStatus, CalendarSyncStatusNotEnabled)
	}
	if result.CalendarSyncStatus != CalendarSyncStatusNotEnabled {
		t.Fatalf("result sync status = %q, want %q", result.CalendarSyncStatus, CalendarSyncStatusNotEnabled)
	}
}

func TestUseCaseCreateSyncsGoogleCalendar(t *testing.T) {
	repo := &fakeRepository{}
	provider := &fakeGoogleCalendarProvider{
		result: infra.GoogleCalendarEventResult{
			CalendarID: "calendar-1",
			EventID:    "event-1",
			HTMLLink:   "https://calendar.google.com/event-1",
		},
	}
	useCase := NewUseCase(NewService(), repo, provider, SyncConfig{
		Enabled:    true,
		CalendarID: "calendar-1",
		TimeZone:   "Asia/Taipei",
	})

	result, err := useCase.Create(context.Background(), validCreateCommand())
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if repo.createCommand.CalendarSyncStatus != CalendarSyncStatusPending {
		t.Fatalf("create sync status = %q, want %q", repo.createCommand.CalendarSyncStatus, CalendarSyncStatusPending)
	}
	if !provider.called {
		t.Fatal("provider should be called")
	}
	if provider.command.CalendarID != "calendar-1" {
		t.Fatalf("provider calendar id = %q, want calendar-1", provider.command.CalendarID)
	}
	if provider.command.TimeZone != "Asia/Taipei" {
		t.Fatalf("provider time zone = %q, want Asia/Taipei", provider.command.TimeZone)
	}
	if !repo.updateCalled {
		t.Fatal("sync result should be updated")
	}
	if repo.updateResult.CalendarSyncStatus != CalendarSyncStatusSynced {
		t.Fatalf("update sync status = %q, want %q", repo.updateResult.CalendarSyncStatus, CalendarSyncStatusSynced)
	}
	if repo.updateResult.GoogleCalendarEventID != "event-1" {
		t.Fatalf("event id = %q, want event-1", repo.updateResult.GoogleCalendarEventID)
	}
	if repo.updateResult.CalendarSyncedAt == nil {
		t.Fatal("syncedAt should be set")
	}
	if result.CalendarSyncStatus != CalendarSyncStatusSynced {
		t.Fatalf("result sync status = %q, want %q", result.CalendarSyncStatus, CalendarSyncStatusSynced)
	}
	if result.GoogleCalendarHTMLLink == "" {
		t.Fatal("result html link should be set")
	}
}

func TestUseCaseCreateMarksSyncFailedWithoutDroppingTask(t *testing.T) {
	repo := &fakeRepository{}
	provider := &fakeGoogleCalendarProvider{err: errors.New("google api failed")}
	useCase := NewUseCase(NewService(), repo, provider, SyncConfig{
		Enabled:    true,
		CalendarID: "calendar-1",
		TimeZone:   "Asia/Taipei",
	})

	result, err := useCase.Create(context.Background(), validCreateCommand())
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if !repo.updateCalled {
		t.Fatal("sync result should be updated")
	}
	if repo.updateResult.CalendarSyncStatus != CalendarSyncStatusFailed {
		t.Fatalf("update sync status = %q, want %q", repo.updateResult.CalendarSyncStatus, CalendarSyncStatusFailed)
	}
	if repo.updateResult.CalendarSyncError != "google api failed" {
		t.Fatalf("sync error = %q, want google api failed", repo.updateResult.CalendarSyncError)
	}
	if result.TaskID == "" {
		t.Fatal("task should still be returned")
	}
	if result.CalendarSyncStatus != CalendarSyncStatusFailed {
		t.Fatalf("result sync status = %q, want %q", result.CalendarSyncStatus, CalendarSyncStatusFailed)
	}
}

func TestUseCaseCreateReturnsErrorWhenRequiredTimeMissing(t *testing.T) {
	repo := &fakeRepository{}
	provider := &fakeGoogleCalendarProvider{}
	useCase := NewUseCase(NewService(), repo, provider, SyncConfig{Enabled: true})

	command := validCreateCommand()
	command.StartAt = ""

	_, err := useCase.Create(context.Background(), command)
	if err == nil {
		t.Fatal("Create should return an error")
	}
	if provider.called {
		t.Fatal("provider should not be called when validation fails")
	}
	if repo.createCommand.Summary != "" {
		t.Fatal("repository should not be called when validation fails")
	}
}

func validCreateCommand() CreateCommand {
	return CreateCommand{
		Source:            "rest",
		RawText:           "小傑約明天吃午餐",
		TaskType:          "calendar",
		Operation:         "create",
		Summary:           "小傑約吃午餐",
		StartAt:           "2026-04-16 12:00:00",
		EndAt:             "2026-04-16 12:30:00",
		Location:          "",
		MissingFields:     []string{"location"},
		InternalAppID:     "linebot-app",
		InternalBuilderID: 4,
		InternalRequest:   "{}",
		InternalResponse:  "{}",
	}
}
