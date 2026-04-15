package app

import (
	"context"
	"fmt"
	"net/http"

	"linebot-backend/internal/calendar"
	"linebot-backend/internal/gatekeeper"
	"linebot-backend/internal/infra"
	"linebot-backend/internal/internalclient"
	"linebot-backend/internal/task"
)

// App represents the LineBot Backend application.
type App struct {
	handler        http.Handler
	store          *infra.Store
	internalClient *internalclient.Service
}

// New creates and wires up the LineBot Backend application.
func New(cfg infra.Config) (*App, error) {
	// 1. Create Firestore store
	store, err := infra.NewStoreWithOptions(infra.StoreOptions{
		ProjectID:    cfg.FirestoreProjectID,
		EmulatorHost: cfg.FirestoreEmulatorHost,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore store: %w", err)
	}

	// 2. Create Internal AI Copilot gRPC client
	internalClient, err := internalclient.NewService(cfg.InternalGRPCAddr)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to create Internal gRPC client: %w", err)
	}

	// 3. Create calendar module
	var googleCalendarProvider infra.GoogleCalendarProvider
	if cfg.GoogleCalendarEnabled {
		googleCalendarProvider, err = infra.NewGoogleCalendarClient(context.Background(), infra.GoogleCalendarClientOptions{
			CredentialsFile: cfg.GoogleOAuthCredentialsFile,
			TokenFile:       cfg.GoogleOAuthTokenFile,
			CalendarID:      cfg.GoogleCalendarID,
			TimeZone:        cfg.GoogleCalendarTimeZone,
		})
		if err != nil {
			internalClient.Close()
			store.Close()
			return nil, fmt.Errorf("failed to create Google Calendar client: %w", err)
		}
	}

	calendarService := calendar.NewService()
	calendarRepository := calendar.NewRepository(store)
	calendarUseCase := calendar.NewUseCase(
		calendarService,
		calendarRepository,
		googleCalendarProvider,
		calendar.SyncConfig{
			Enabled:    cfg.GoogleCalendarEnabled,
			CalendarID: cfg.GoogleCalendarID,
			TimeZone:   cfg.GoogleCalendarTimeZone,
		},
	)

	// 4. Create task module
	taskService := task.NewService()
	taskUseCase := task.NewUseCase(taskService, internalClient, calendarUseCase, cfg)

	// 5. Create gatekeeper module
	gatekeeperUseCase := gatekeeper.NewUseCase(taskUseCase)
	gatekeeperHandler := gatekeeper.NewHandler(gatekeeperUseCase)

	// 6. Create HTTP router
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tasks", gatekeeperHandler.CreateTask)

	return &App{
		handler:        mux,
		store:          store,
		internalClient: internalClient,
	}, nil
}

// Handler returns the HTTP handler for the application.
func (a *App) Handler() http.Handler {
	return a.handler
}

// Close closes all resources.
func (a *App) Close() error {
	var errs []error

	if err := a.internalClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close internal client: %w", err))
	}

	if err := a.store.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close store: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}
