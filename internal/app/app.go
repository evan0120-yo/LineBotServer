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
	internalClient *internalclient.Service
}

// New creates and wires up the LineBot Backend application.
func New(cfg infra.Config) (*App, error) {
	// 1. Create Internal AI Copilot gRPC client
	internalClient, err := internalclient.NewService(cfg.InternalGRPCAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create Internal gRPC client: %w", err)
	}

	// 2. Create calendar module
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
			return nil, fmt.Errorf("failed to create Google Calendar client: %w", err)
		}
	}

	var lineReplyClient infra.LineReplyProvider
	if cfg.LineChannelAccessToken != "" {
		lineReplyClient = infra.NewLineMessagingClient(cfg.LineChannelAccessToken)
	}

	calendarService := calendar.NewService()
	calendarUseCase := calendar.NewUseCase(
		calendarService,
		googleCalendarProvider,
		calendar.Config{
			Enabled:    cfg.GoogleCalendarEnabled,
			CalendarID: cfg.GoogleCalendarID,
			TimeZone:   cfg.GoogleCalendarTimeZone,
		},
	)

	// 4. Create task module
	taskService := task.NewService()
	taskUseCase := task.NewUseCase(taskService, internalClient, calendarUseCase, cfg)

	// 4. Create gatekeeper module
	gatekeeperUseCase := gatekeeper.NewUseCase(taskUseCase)
	gatekeeperHandler := gatekeeper.NewHandler(gatekeeperUseCase)

	// 5. Create HTTP router
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tasks", gatekeeperHandler.CreateTask)

	// Only register LINE webhook if channel secret, access token, and bot user ID are configured.
	if cfg.LineChannelSecret != "" && cfg.LineChannelAccessToken != "" && cfg.LineBotUserID != "" {
		lineHandler := gatekeeper.NewLineHandler(gatekeeperUseCase, lineReplyClient, cfg.LineChannelSecret, cfg.LineBotUserID)
		mux.HandleFunc("POST /api/line/webhook", lineHandler.ServeHTTP)
	}

	return &App{
		handler:        mux,
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

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}
