package gatekeeper

// CreateTaskRequest represents the JSON request body for POST /api/tasks.
type CreateTaskRequest struct {
	Text          string `json:"text"`
	ReferenceTime string `json:"referenceTime"`
	TimeZone      string `json:"timeZone"`
}

// CreateTaskResponseData represents the JSON response data for POST /api/tasks.
type CreateTaskResponseData struct {
	Operation string                    `json:"operation"`
	ReplyText string                    `json:"replyText"`
	Events    []CreateTaskResponseEvent `json:"events"`
}

// CreateTaskResponseEvent represents one calendar event in REST response.
type CreateTaskResponseEvent struct {
	EventID  string `json:"eventId"`
	Summary  string `json:"summary"`
	StartAt  string `json:"startAt"`
	EndAt    string `json:"endAt"`
	Location string `json:"location"`
}
