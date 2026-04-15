package task

// TaskType represents a supported task type.
type TaskType string

const (
	// TaskTypeCalendar represents calendar task type.
	TaskTypeCalendar TaskType = "calendar"
)

// SupportedTaskTypes returns the list of supported task types for the first version.
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

// TaskResult holds the result of task creation.
type TaskResult struct {
	TaskID        string
	Operation     string
	Summary       string
	StartAt       string
	EndAt         string
	Location      string
	MissingFields []string
}
