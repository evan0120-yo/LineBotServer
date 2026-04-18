package infra

import (
	"context"
)

// GoogleCalendarCreateEventCommand holds fields needed to create a Google Calendar event.
type GoogleCalendarCreateEventCommand struct {
	CalendarID string
	Summary    string
	StartAt    string
	EndAt      string
	TimeZone   string
	Location   string
}

// GoogleCalendarListEventsCommand holds fields needed to query Google Calendar events.
type GoogleCalendarListEventsCommand struct {
	CalendarID   string
	QueryStartAt string
	QueryEndAt   string
	TimeZone     string
}

// GoogleCalendarDeleteEventCommand holds fields needed to delete a Google Calendar event.
type GoogleCalendarDeleteEventCommand struct {
	CalendarID string
	EventID    string
}

// GoogleCalendarUpdateEventCommand holds fields needed to update a Google Calendar event.
type GoogleCalendarUpdateEventCommand struct {
	CalendarID string
	EventID    string
	Summary    string
	TimeZone   string
	Location   string
}

// GoogleCalendarEventResult holds Google Calendar event metadata.
type GoogleCalendarEventResult struct {
	CalendarID string
	EventID    string
	Summary    string
	StartAt    string
	EndAt      string
	Location   string
	HTMLLink   string
}

// GoogleCalendarProvider performs CRUD-like operations against Google Calendar.
type GoogleCalendarProvider interface {
	CreateEvent(ctx context.Context, command GoogleCalendarCreateEventCommand) (GoogleCalendarEventResult, error)
	ListEvents(ctx context.Context, command GoogleCalendarListEventsCommand) ([]GoogleCalendarEventResult, error)
	DeleteEvent(ctx context.Context, command GoogleCalendarDeleteEventCommand) error
	UpdateEventSummary(ctx context.Context, command GoogleCalendarUpdateEventCommand) (GoogleCalendarEventResult, error)
}

// LineReplyCommand holds fields needed to send a LINE reply message.
type LineReplyCommand struct {
	ReplyToken string
	Text       string
}

// LineReplyProvider sends LINE reply messages.
type LineReplyProvider interface {
	ReplyText(ctx context.Context, command LineReplyCommand) error
}
