package gatekeeper

// CreateTaskRequest represents the JSON request body for POST /api/tasks.
type CreateTaskRequest struct {
	Text          string `json:"text"`
	ReferenceTime string `json:"referenceTime"`
	TimeZone      string `json:"timeZone"`
}

// CreateTaskResponseData represents the JSON response data for POST /api/tasks.
type CreateTaskResponseData struct {
	TaskID                 string   `json:"taskId"`
	Operation              string   `json:"operation"`
	Summary                string   `json:"summary"`
	StartAt                string   `json:"startAt"`
	EndAt                  string   `json:"endAt"`
	Location               string   `json:"location"`
	MissingFields          []string `json:"missingFields"`
	CalendarSyncStatus     string   `json:"calendarSyncStatus"`
	GoogleCalendarID       string   `json:"googleCalendarId"`
	GoogleCalendarEventID  string   `json:"googleCalendarEventId"`
	GoogleCalendarHTMLLink string   `json:"googleCalendarHtmlLink"`
	CalendarSyncError      string   `json:"calendarSyncError"`
}
