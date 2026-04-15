package infra

import (
	"context"
	"time"
)

// CalendarTaskDoc represents a calendar task document in Firestore.
type CalendarTaskDoc struct {
	TaskID                 string     `firestore:"taskId"`
	Source                 string     `firestore:"source"`
	RawText                string     `firestore:"rawText"`
	TaskType               string     `firestore:"taskType"`
	Operation              string     `firestore:"operation"`
	Summary                string     `firestore:"summary"`
	StartAt                string     `firestore:"startAt"`
	EndAt                  string     `firestore:"endAt"`
	Location               string     `firestore:"location"`
	MissingFields          []string   `firestore:"missingFields"`
	Status                 string     `firestore:"status"`
	CalendarSyncStatus     string     `firestore:"calendarSyncStatus"`
	GoogleCalendarID       string     `firestore:"googleCalendarId"`
	GoogleCalendarEventID  string     `firestore:"googleCalendarEventId"`
	GoogleCalendarHTMLLink string     `firestore:"googleCalendarHtmlLink"`
	CalendarSyncError      string     `firestore:"calendarSyncError"`
	CalendarSyncedAt       *time.Time `firestore:"calendarSyncedAt,omitempty"`
	InternalAppID          string     `firestore:"internalAppId"`
	InternalBuilderID      int        `firestore:"internalBuilderId"`
	InternalRequest        string     `firestore:"internalRequest"`
	InternalResponse       string     `firestore:"internalResponse"`
	CreatedAt              time.Time  `firestore:"createdAt"`
	UpdatedAt              time.Time  `firestore:"updatedAt"`
}

// CalendarTaskSyncResult represents Google Calendar sync metadata persisted to Firestore.
type CalendarTaskSyncResult struct {
	CalendarSyncStatus     string
	GoogleCalendarID       string
	GoogleCalendarEventID  string
	GoogleCalendarHTMLLink string
	CalendarSyncError      string
	CalendarSyncedAt       *time.Time
}

// GoogleCalendarCreateEventCommand holds fields needed to create a Google Calendar event.
type GoogleCalendarCreateEventCommand struct {
	CalendarID string
	Summary    string
	StartAt    string
	EndAt      string
	TimeZone   string
	Location   string
}

// GoogleCalendarEventResult holds the created Google Calendar event metadata.
type GoogleCalendarEventResult struct {
	CalendarID string
	EventID    string
	HTMLLink   string
}

// GoogleCalendarProvider creates events in an external Google Calendar.
type GoogleCalendarProvider interface {
	CreateEvent(ctx context.Context, command GoogleCalendarCreateEventCommand) (GoogleCalendarEventResult, error)
}
