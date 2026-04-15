package infra

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// Store wraps Firestore client for LineBot Backend persistence.
type Store struct {
	client    *firestore.Client
	projectID string
}

// StoreOptions holds options for creating a Store.
type StoreOptions struct {
	ProjectID    string
	EmulatorHost string
}

// NewStoreWithOptions creates a new Store with the given options.
func NewStoreWithOptions(opts StoreOptions) (*Store, error) {
	ctx := context.Background()

	var clientOpts []option.ClientOption
	if opts.EmulatorHost != "" {
		// Set emulator host before creating Firestore client
		os.Setenv("FIRESTORE_EMULATOR_HOST", opts.EmulatorHost)
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	}

	client, err := firestore.NewClient(ctx, opts.ProjectID, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient: %w", err)
	}

	return &Store{
		client:    client,
		projectID: opts.ProjectID,
	}, nil
}

// Close closes the Firestore client.
func (s *Store) Close() error {
	return s.client.Close()
}

// CreateCalendarTask creates a new calendar task document in Firestore.
func (s *Store) CreateCalendarTask(ctx context.Context, task CalendarTaskDoc) error {
	collection := s.client.Collection("calendar_tasks")
	docRef := collection.Doc(task.TaskID)

	_, err := docRef.Set(ctx, task)
	if err != nil {
		return NewFirestoreWriteError(err)
	}

	return nil
}

// UpdateCalendarTaskSyncResult updates Google Calendar sync metadata for a calendar task.
func (s *Store) UpdateCalendarTaskSyncResult(ctx context.Context, taskID string, result CalendarTaskSyncResult) error {
	updates := []firestore.Update{
		{Path: "calendarSyncStatus", Value: result.CalendarSyncStatus},
		{Path: "googleCalendarId", Value: result.GoogleCalendarID},
		{Path: "googleCalendarEventId", Value: result.GoogleCalendarEventID},
		{Path: "googleCalendarHtmlLink", Value: result.GoogleCalendarHTMLLink},
		{Path: "calendarSyncError", Value: result.CalendarSyncError},
		{Path: "updatedAt", Value: time.Now()},
	}
	if result.CalendarSyncedAt != nil {
		updates = append(updates, firestore.Update{Path: "calendarSyncedAt", Value: result.CalendarSyncedAt})
	}

	docRef := s.client.Collection("calendar_tasks").Doc(taskID)
	_, err := docRef.Update(ctx, updates)
	if err != nil {
		return NewFirestoreWriteError(err)
	}
	return nil
}

// GetCalendarTask retrieves a calendar task by ID.
func (s *Store) GetCalendarTask(ctx context.Context, taskID string) (CalendarTaskDoc, error) {
	docRef := s.client.Collection("calendar_tasks").Doc(taskID)

	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return CalendarTaskDoc{}, fmt.Errorf("firestore Get: %w", err)
	}

	var task CalendarTaskDoc
	if err := docSnap.DataTo(&task); err != nil {
		return CalendarTaskDoc{}, fmt.Errorf("firestore DataTo: %w", err)
	}

	return task, nil
}
