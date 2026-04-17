package task

// TaskType represents a supported task type.
type TaskType string

const (
	// TaskTypeCalendar represents calendar task type.
	TaskTypeCalendar TaskType = "calendar"
)

// SupportedTaskTypes returns the list of supported task types.
func SupportedTaskTypes() []string {
	return []string{string(TaskTypeCalendar)}
}

// CreateFromTextCommand holds parameters for creating a task from natural language text.
type CreateFromTextCommand struct {
	Source        string
	Text          string
	ReferenceTime string
	TimeZone      string
	ClientIP      string
}

// TaskEvent represents an event returned to REST or LINE reply formatting.
type TaskEvent struct {
	EventID  string
	Summary  string
	StartAt  string
	EndAt    string
	Location string
}

// TaskResult holds the result of a task operation.
type TaskResult struct {
	Operation string
	ReplyText string
	Events    []TaskEvent
}
