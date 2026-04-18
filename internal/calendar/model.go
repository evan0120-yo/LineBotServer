package calendar

// CreateCommand holds parameters for creating a calendar event.
type CreateCommand struct {
	Summary  string
	StartAt  string
	EndAt    string
	Location string
}

// QueryCommand holds parameters for querying calendar events.
type QueryCommand struct {
	QueryStartAt string
	QueryEndAt   string
}

// DeleteCommand holds parameters for deleting a calendar event.
type DeleteCommand struct {
	EventID string
}

// UpdateCommand holds parameters for updating a calendar event.
type UpdateCommand struct {
	EventID  string
	Summary  string
	Location string
}

// Event represents a calendar event in LineBot Backend.
type Event struct {
	EventID  string
	Summary  string
	StartAt  string
	EndAt    string
	Location string
}

// Config controls Google Calendar behavior.
type Config struct {
	Enabled    bool
	CalendarID string
	TimeZone   string
}
