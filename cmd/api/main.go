package main

import (
	"log"
	"net/http"

	"linebot-backend/internal/app"
	"linebot-backend/internal/infra"
)

func main() {
	// Load configuration from environment
	cfg := infra.LoadConfigFromEnv()

	// Create application
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("Error closing application: %v", err)
		}
	}()

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      application.Handler(),
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
	}

	// Start server
	log.Printf("LineBot Backend starting on %s", cfg.Addr)
	log.Printf("Internal gRPC address: %s", cfg.InternalGRPCAddr)
	log.Printf("Internal App ID: %s", cfg.InternalAppID)
	log.Printf("Internal Builder ID: %d", cfg.InternalBuilderID)
	log.Printf("Google Calendar enabled: %t", cfg.GoogleCalendarEnabled)
	log.Printf("LINE webhook enabled: %t", cfg.LineChannelSecret != "" && cfg.LineBotUserID != "")
	if cfg.LineBotUserID != "" {
		log.Printf("LINE Bot User ID: %s", cfg.LineBotUserID)
	}
	if cfg.GoogleCalendarEnabled {
		log.Printf("Google Calendar ID: %s", cfg.GoogleCalendarID)
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
