package calendar

import (
	"context"
	"errors"
	"testing"

	"linebot-backend/internal/infra"
)

type fakeGoogleCalendarProvider struct {
	createCommand infra.GoogleCalendarCreateEventCommand
	createResult  infra.GoogleCalendarEventResult
	createErr     error
	createCalled  bool

	listCommand infra.GoogleCalendarListEventsCommand
	listResult  []infra.GoogleCalendarEventResult
	listErr     error
	listCalled  bool

	deleteCommand infra.GoogleCalendarDeleteEventCommand
	deleteErr     error
	deleteCalled  bool

	updateCommand infra.GoogleCalendarUpdateEventCommand
	updateResult  infra.GoogleCalendarEventResult
	updateErr     error
	updateCalled  bool
}

func (p *fakeGoogleCalendarProvider) CreateEvent(_ context.Context, command infra.GoogleCalendarCreateEventCommand) (infra.GoogleCalendarEventResult, error) {
	p.createCalled = true
	p.createCommand = command
	if p.createErr != nil {
		return infra.GoogleCalendarEventResult{}, p.createErr
	}
	return p.createResult, nil
}

func (p *fakeGoogleCalendarProvider) ListEvents(_ context.Context, command infra.GoogleCalendarListEventsCommand) ([]infra.GoogleCalendarEventResult, error) {
	p.listCalled = true
	p.listCommand = command
	if p.listErr != nil {
		return nil, p.listErr
	}
	return p.listResult, nil
}

func (p *fakeGoogleCalendarProvider) DeleteEvent(_ context.Context, command infra.GoogleCalendarDeleteEventCommand) error {
	p.deleteCalled = true
	p.deleteCommand = command
	return p.deleteErr
}

func (p *fakeGoogleCalendarProvider) UpdateEventSummary(_ context.Context, command infra.GoogleCalendarUpdateEventCommand) (infra.GoogleCalendarEventResult, error) {
	p.updateCalled = true
	p.updateCommand = command
	if p.updateErr != nil {
		return infra.GoogleCalendarEventResult{}, p.updateErr
	}
	return p.updateResult, nil
}

func TestUseCaseCreate(t *testing.T) {
	provider := &fakeGoogleCalendarProvider{
		createResult: infra.GoogleCalendarEventResult{
			CalendarID: "calendar-1",
			EventID:    "event-1",
			Summary:    "小傑約明天吃晚餐",
			StartAt:    "2026-04-18 12:00:00",
			EndAt:      "2026-04-18 12:30:00",
		},
	}
	useCase := NewUseCase(NewService(), provider, Config{
		Enabled:    true,
		CalendarID: "calendar-1",
		TimeZone:   "Asia/Taipei",
	})

	event, err := useCase.Create(context.Background(), CreateCommand{
		Summary: "小傑約明天吃晚餐",
		StartAt: "2026-04-18 12:00:00",
		EndAt:   "2026-04-18 12:30:00",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if !provider.createCalled {
		t.Fatal("expected provider create call")
	}
	if event.EventID != "event-1" {
		t.Fatalf("eventID = %q, want event-1", event.EventID)
	}
}

func TestUseCaseQueryAppliesOverlapRule(t *testing.T) {
	provider := &fakeGoogleCalendarProvider{
		listResult: []infra.GoogleCalendarEventResult{
			{
				EventID: "event-1",
				Summary: "午餐",
				StartAt: "2026-04-18 12:00:00",
				EndAt:   "2026-04-18 15:00:00",
			},
			{
				EventID: "event-2",
				Summary: "晚餐",
				StartAt: "2026-04-18 18:00:00",
				EndAt:   "2026-04-18 19:00:00",
			},
		},
	}
	useCase := NewUseCase(NewService(), provider, Config{
		Enabled:    true,
		CalendarID: "calendar-1",
		TimeZone:   "Asia/Taipei",
	})

	events, err := useCase.Query(context.Background(), QueryCommand{
		QueryStartAt: "2026-04-18 14:30:00",
		QueryEndAt:   "2026-04-18 15:30:00",
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if !provider.listCalled {
		t.Fatal("expected provider list call")
	}
	if len(events) != 1 || events[0].EventID != "event-1" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestUseCaseDelete(t *testing.T) {
	provider := &fakeGoogleCalendarProvider{}
	useCase := NewUseCase(NewService(), provider, Config{
		Enabled:    true,
		CalendarID: "calendar-1",
		TimeZone:   "Asia/Taipei",
	})

	if err := useCase.Delete(context.Background(), DeleteCommand{EventID: "event-1"}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !provider.deleteCalled {
		t.Fatal("expected provider delete call")
	}
}

func TestUseCaseUpdate(t *testing.T) {
	provider := &fakeGoogleCalendarProvider{
		updateResult: infra.GoogleCalendarEventResult{
			EventID: "event-1",
			Summary: "新標題",
			StartAt: "2026-04-18 12:00:00",
			EndAt:   "2026-04-18 12:30:00",
		},
	}
	useCase := NewUseCase(NewService(), provider, Config{
		Enabled:    true,
		CalendarID: "calendar-1",
		TimeZone:   "Asia/Taipei",
	})

	event, err := useCase.Update(context.Background(), UpdateCommand{
		EventID: "event-1",
		Summary: "新標題",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !provider.updateCalled {
		t.Fatal("expected provider update call")
	}
	if event.Summary != "新標題" {
		t.Fatalf("summary = %q, want 新標題", event.Summary)
	}
}

func TestUseCaseQueryProviderFailure(t *testing.T) {
	provider := &fakeGoogleCalendarProvider{listErr: errors.New("google api failed")}
	useCase := NewUseCase(NewService(), provider, Config{
		Enabled:    true,
		CalendarID: "calendar-1",
		TimeZone:   "Asia/Taipei",
	})

	if _, err := useCase.Query(context.Background(), QueryCommand{
		QueryStartAt: "2026-04-18 12:00:00",
		QueryEndAt:   "2026-04-18 12:30:00",
	}); err == nil {
		t.Fatal("expected query error")
	}
}
