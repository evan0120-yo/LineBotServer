package calendar

import "time"

const (
	// TaskStatusCreated means the calendar task document was created in Firestore.
	TaskStatusCreated = "created"

	// CalendarSyncStatusNotEnabled means Google Calendar sync is disabled by config.
	CalendarSyncStatusNotEnabled = "not_enabled"

	// CalendarSyncStatusPending means the task was saved and external sync is in progress.
	CalendarSyncStatusPending = "calendar_sync_pending"

	// CalendarSyncStatusSynced means the Google Calendar event was created.
	CalendarSyncStatusSynced = "calendar_synced"

	// CalendarSyncStatusFailed means Google Calendar sync failed after Firestore persistence.
	CalendarSyncStatusFailed = "calendar_sync_failed"
)

// CreateCommand holds parameters for creating a calendar task.
type CreateCommand struct {
	Source             string
	RawText            string
	TaskType           string
	Operation          string
	Summary            string
	StartAt            string
	EndAt              string
	Location           string
	MissingFields      []string
	InternalAppID      string
	InternalBuilderID  int
	InternalRequest    string
	InternalResponse   string
	CalendarSyncStatus string
}

// CalendarTask represents a created calendar task.
type CalendarTask struct {
	TaskID                 string
	Summary                string
	StartAt                string
	EndAt                  string
	Location               string
	MissingFields          []string
	CalendarSyncStatus     string
	GoogleCalendarID       string
	GoogleCalendarEventID  string
	GoogleCalendarHTMLLink string
	CalendarSyncError      string
	CalendarSyncedAt       *time.Time
	CreatedAt              time.Time
}

// SyncConfig controls Google Calendar sync behavior.
type SyncConfig struct {
	Enabled    bool
	CalendarID string
	TimeZone   string
}

// SyncResult holds Google Calendar sync metadata for a task.
type SyncResult struct {
	CalendarSyncStatus     string
	GoogleCalendarID       string
	GoogleCalendarEventID  string
	GoogleCalendarHTMLLink string
	CalendarSyncError      string
	CalendarSyncedAt       *time.Time
}
