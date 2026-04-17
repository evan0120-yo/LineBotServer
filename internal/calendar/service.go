package calendar

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"linebot-backend/internal/infra"
)

const dateTimeLayout = "2006-01-02 15:04:05"

var weekdayLabels = []string{"週日", "週一", "週二", "週三", "週四", "週五", "週六"}

// Service handles calendar business logic.
type Service struct{}

// NewService creates a new calendar Service.
func NewService() *Service {
	return &Service{}
}

// ValidateCreate validates a create command.
func (s *Service) ValidateCreate(command CreateCommand) error {
	return s.validateRequiredFields(map[string]string{
		"summary": command.Summary,
		"startAt": command.StartAt,
		"endAt":   command.EndAt,
	})
}

// ValidateQuery validates a query command.
func (s *Service) ValidateQuery(command QueryCommand) error {
	return s.validateRequiredFields(map[string]string{
		"queryStartAt": command.QueryStartAt,
		"queryEndAt":   command.QueryEndAt,
	})
}

// ValidateDelete validates a delete command.
func (s *Service) ValidateDelete(command DeleteCommand) error {
	return s.validateRequiredFields(map[string]string{
		"eventId": command.EventID,
	})
}

// ValidateUpdate validates an update command.
func (s *Service) ValidateUpdate(command UpdateCommand) error {
	return s.validateRequiredFields(map[string]string{
		"eventId": command.EventID,
		"summary": command.Summary,
	})
}

func (s *Service) validateRequiredFields(fields map[string]string) error {
	missingFields := make([]string, 0)
	for key, value := range fields {
		if strings.TrimSpace(value) == "" {
			missingFields = append(missingFields, key)
		}
	}
	if len(missingFields) == 0 {
		return nil
	}
	sort.Strings(missingFields)
	return infra.NewInternalExtractionIncompleteError(missingFields)
}

// FilterOverlappingEvents returns events whose time range overlaps with the query range.
func (s *Service) FilterOverlappingEvents(events []Event, command QueryCommand, timeZone string) ([]Event, error) {
	queryStartAt, err := parseInLocation(command.QueryStartAt, timeZone)
	if err != nil {
		return nil, fmt.Errorf("parse queryStartAt: %w", err)
	}
	queryEndAt, err := parseInLocation(command.QueryEndAt, timeZone)
	if err != nil {
		return nil, fmt.Errorf("parse queryEndAt: %w", err)
	}

	filtered := make([]Event, 0, len(events))
	for _, event := range events {
		eventStartAt, startErr := parseInLocation(event.StartAt, timeZone)
		if startErr != nil {
			return nil, fmt.Errorf("parse event startAt: %w", startErr)
		}
		eventEndAt, endErr := parseInLocation(event.EndAt, timeZone)
		if endErr != nil {
			return nil, fmt.Errorf("parse event endAt: %w", endErr)
		}

		if !eventStartAt.After(queryEndAt) && !eventEndAt.Before(queryStartAt) {
			filtered = append(filtered, event)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].StartAt == filtered[j].StartAt {
			return filtered[i].EventID < filtered[j].EventID
		}
		return filtered[i].StartAt < filtered[j].StartAt
	})

	return filtered, nil
}

// FormatEventsReply renders calendar results into the fixed LINE reply format.
func (s *Service) FormatEventsReply(events []Event, timeZone string) (string, error) {
	if len(events) == 0 {
		return "沒資料", nil
	}

	sections := make([]string, 0, len(events))
	for _, event := range events {
		timeRange, err := s.formatEventTimeRange(event, timeZone)
		if err != nil {
			return "", err
		}

		sections = append(sections, strings.Join([]string{
			strings.TrimSpace(event.EventID),
			strings.TrimSpace(event.Summary),
			timeRange,
		}, "\n"))
	}

	return strings.Join(sections, "\n\n"), nil
}

// FormatDeleteSuccessReply renders a delete success message.
func (s *Service) FormatDeleteSuccessReply() string {
	return "刪除成功"
}

func (s *Service) formatEventTimeRange(event Event, timeZone string) (string, error) {
	startAt, err := parseInLocation(event.StartAt, timeZone)
	if err != nil {
		return "", fmt.Errorf("parse event startAt: %w", err)
	}
	endAt, err := parseInLocation(event.EndAt, timeZone)
	if err != nil {
		return "", fmt.Errorf("parse event endAt: %w", err)
	}

	return fmt.Sprintf(
		"%s (%s) ~ %s (%s)",
		startAt.Format("2006-01-02 15:04"),
		weekdayLabel(startAt.Weekday()),
		endAt.Format("2006-01-02 15:04"),
		weekdayLabel(endAt.Weekday()),
	), nil
}

func parseInLocation(value, timeZone string) (time.Time, error) {
	location, err := time.LoadLocation(normalizeTimeZone(timeZone))
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(dateTimeLayout, strings.TrimSpace(value), location)
}

func normalizeTimeZone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "Asia/Taipei"
	}
	return value
}

func weekdayLabel(weekday time.Weekday) string {
	return weekdayLabels[int(weekday)]
}
