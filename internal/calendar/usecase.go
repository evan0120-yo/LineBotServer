package calendar

import (
	"context"
	"log"
	"strings"

	"linebot-backend/internal/infra"
)

// UseCase orchestrates calendar operations through Google Calendar.
type UseCase struct {
	service  *Service
	provider infra.GoogleCalendarProvider
	config   Config
}

// NewUseCase creates a new calendar UseCase.
func NewUseCase(service *Service, provider infra.GoogleCalendarProvider, config Config) *UseCase {
	return &UseCase{
		service:  service,
		provider: provider,
		config:   config,
	}
}

// Create creates a Google Calendar event.
func (u *UseCase) Create(ctx context.Context, command CreateCommand) (Event, error) {
	log.Printf("[INFO] calendar create start: summary=%q startAt=%q endAt=%q", command.Summary, command.StartAt, command.EndAt)
	if err := u.ensureProviderConfigured(); err != nil {
		return Event{}, err
	}
	if err := u.service.ValidateCreate(command); err != nil {
		log.Printf("[INFO] calendar create validate failed: err=%v", err)
		return Event{}, err
	}

	event, err := u.provider.CreateEvent(ctx, infra.GoogleCalendarCreateEventCommand{
		CalendarID: u.config.CalendarID,
		Summary:    command.Summary,
		StartAt:    command.StartAt,
		EndAt:      command.EndAt,
		TimeZone:   u.config.TimeZone,
		Location:   command.Location,
	})
	if err != nil {
		log.Printf("[INFO] calendar create provider failed: err=%v", err)
		return Event{}, infra.NewError("GOOGLE_CALENDAR_CREATE_FAILED", "建立行事曆事件失敗", 500)
	}
	log.Printf("[INFO] calendar create completed: eventID=%s summary=%q", event.EventID, event.Summary)

	return Event{
		EventID:  event.EventID,
		Summary:  event.Summary,
		StartAt:  event.StartAt,
		EndAt:    event.EndAt,
		Location: event.Location,
	}, nil
}

// Query queries Google Calendar events by time range and applies overlap filtering.
func (u *UseCase) Query(ctx context.Context, command QueryCommand) ([]Event, error) {
	log.Printf("[INFO] calendar query start: queryStartAt=%q queryEndAt=%q", command.QueryStartAt, command.QueryEndAt)
	if err := u.ensureProviderConfigured(); err != nil {
		return nil, err
	}
	if err := u.service.ValidateQuery(command); err != nil {
		log.Printf("[INFO] calendar query validate failed: err=%v", err)
		return nil, err
	}

	candidates, err := u.provider.ListEvents(ctx, infra.GoogleCalendarListEventsCommand{
		CalendarID:   u.config.CalendarID,
		QueryStartAt: command.QueryStartAt,
		QueryEndAt:   command.QueryEndAt,
		TimeZone:     u.config.TimeZone,
	})
	if err != nil {
		log.Printf("[INFO] calendar query provider failed: err=%v", err)
		return nil, infra.NewError("GOOGLE_CALENDAR_QUERY_FAILED", "查詢行事曆事件失敗", 500)
	}

	events := make([]Event, 0, len(candidates))
	for _, candidate := range candidates {
		events = append(events, Event{
			EventID:  candidate.EventID,
			Summary:  candidate.Summary,
			StartAt:  candidate.StartAt,
			EndAt:    candidate.EndAt,
			Location: candidate.Location,
		})
	}

	filtered, err := u.service.FilterOverlappingEvents(events, command, u.config.TimeZone)
	if err != nil {
		log.Printf("[INFO] calendar query overlap filter failed: err=%v", err)
		return nil, infra.NewError("GOOGLE_CALENDAR_QUERY_FAILED", "查詢行事曆事件失敗", 500)
	}
	log.Printf("[INFO] calendar query completed: candidates=%d matched=%d", len(events), len(filtered))
	return filtered, nil
}

// Delete deletes a Google Calendar event by eventId.
func (u *UseCase) Delete(ctx context.Context, command DeleteCommand) error {
	log.Printf("[INFO] calendar delete start: eventID=%s", command.EventID)
	if err := u.ensureProviderConfigured(); err != nil {
		return err
	}
	if err := u.service.ValidateDelete(command); err != nil {
		log.Printf("[INFO] calendar delete validate failed: err=%v", err)
		return err
	}

	if err := u.provider.DeleteEvent(ctx, infra.GoogleCalendarDeleteEventCommand{
		CalendarID: u.config.CalendarID,
		EventID:    command.EventID,
	}); err != nil {
		log.Printf("[INFO] calendar delete provider failed: err=%v", err)
		return infra.NewError("GOOGLE_CALENDAR_DELETE_FAILED", "刪除行事曆事件失敗", 500)
	}
	log.Printf("[INFO] calendar delete completed: eventID=%s", command.EventID)
	return nil
}

// Update updates the title of a Google Calendar event by eventId.
func (u *UseCase) Update(ctx context.Context, command UpdateCommand) (Event, error) {
	log.Printf("[INFO] calendar update start: eventID=%s summary=%q", command.EventID, command.Summary)
	if err := u.ensureProviderConfigured(); err != nil {
		return Event{}, err
	}
	if err := u.service.ValidateUpdate(command); err != nil {
		log.Printf("[INFO] calendar update validate failed: err=%v", err)
		return Event{}, err
	}

	event, err := u.provider.UpdateEventSummary(ctx, infra.GoogleCalendarUpdateEventCommand{
		CalendarID: u.config.CalendarID,
		EventID:    command.EventID,
		Summary:    command.Summary,
		TimeZone:   u.config.TimeZone,
	})
	if err != nil {
		log.Printf("[INFO] calendar update provider failed: err=%v", err)
		return Event{}, infra.NewError("GOOGLE_CALENDAR_UPDATE_FAILED", "更新行事曆事件失敗", 500)
	}
	log.Printf("[INFO] calendar update completed: eventID=%s summary=%q", event.EventID, event.Summary)

	return Event{
		EventID:  event.EventID,
		Summary:  event.Summary,
		StartAt:  event.StartAt,
		EndAt:    event.EndAt,
		Location: event.Location,
	}, nil
}

func (u *UseCase) ensureProviderConfigured() error {
	if !u.config.Enabled || u.provider == nil || strings.TrimSpace(u.config.CalendarID) == "" {
		return infra.NewError("GOOGLE_CALENDAR_NOT_CONFIGURED", "Google Calendar 未設定完成", 500)
	}
	return nil
}
