package internalclient

// LineTaskConsultCommand holds parameters for calling Internal AI Copilot LineTaskConsult.
type LineTaskConsultCommand struct {
	AppID              string
	BuilderID          int
	MessageText        string
	ReferenceTime      string
	TimeZone           string
	SupportedTaskTypes []string
	ClientIP           string
}

// LineTaskConsultResult holds the result from Internal AI Copilot LineTaskConsult.
type LineTaskConsultResult struct {
	TaskType      string
	Operation     string
	EventID       string
	Summary       string
	StartAt       string
	EndAt         string
	QueryStartAt  string
	QueryEndAt    string
	Location      string
	MissingFields []string
}
