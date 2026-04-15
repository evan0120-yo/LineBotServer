package calendar

import "time"

// CreateCommand holds parameters for creating a calendar task.
type CreateCommand struct {
	Source            string
	RawText           string
	TaskType          string
	Operation         string
	Summary           string
	StartAt           string
	EndAt             string
	Location          string
	MissingFields     []string
	InternalAppID     string
	InternalBuilderID int
	InternalRequest   string
	InternalResponse  string
}

// CalendarTask represents a created calendar task.
type CalendarTask struct {
	TaskID        string
	Summary       string
	StartAt       string
	EndAt         string
	Location      string
	MissingFields []string
	CreatedAt     time.Time
}
