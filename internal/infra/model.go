package infra

import "time"

// CalendarTaskDoc represents a calendar task document in Firestore.
type CalendarTaskDoc struct {
	TaskID            string    `firestore:"taskId"`
	Source            string    `firestore:"source"`
	RawText           string    `firestore:"rawText"`
	TaskType          string    `firestore:"taskType"`
	Operation         string    `firestore:"operation"`
	Summary           string    `firestore:"summary"`
	StartAt           string    `firestore:"startAt"`
	EndAt             string    `firestore:"endAt"`
	Location          string    `firestore:"location"`
	MissingFields     []string  `firestore:"missingFields"`
	Status            string    `firestore:"status"`
	InternalAppID     string    `firestore:"internalAppId"`
	InternalBuilderID int       `firestore:"internalBuilderId"`
	InternalRequest   string    `firestore:"internalRequest"`
	InternalResponse  string    `firestore:"internalResponse"`
	CreatedAt         time.Time `firestore:"createdAt"`
	UpdatedAt         time.Time `firestore:"updatedAt"`
}
