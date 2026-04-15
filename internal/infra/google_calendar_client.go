package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendarapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const googleCalendarDateTimeLayout = "2006-01-02 15:04:05"

// GoogleCalendarClientOptions holds options for creating a GoogleCalendarClient.
type GoogleCalendarClientOptions struct {
	CredentialsFile string
	TokenFile       string
	CalendarID      string
	TimeZone        string
}

// GoogleCalendarClient creates events through Google Calendar API.
type GoogleCalendarClient struct {
	service    *calendarapi.Service
	calendarID string
	timeZone   string
}

// NewGoogleCalendarClient creates a Google Calendar API client using a stored OAuth token.
func NewGoogleCalendarClient(ctx context.Context, opts GoogleCalendarClientOptions) (*GoogleCalendarClient, error) {
	if strings.TrimSpace(opts.CalendarID) == "" {
		return nil, fmt.Errorf("google calendar id is required")
	}
	if strings.TrimSpace(opts.CredentialsFile) == "" {
		return nil, fmt.Errorf("google oauth credentials file is required")
	}
	if strings.TrimSpace(opts.TokenFile) == "" {
		return nil, fmt.Errorf("google oauth token file is required")
	}

	credentials, err := os.ReadFile(opts.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read google oauth credentials file: %w", err)
	}

	oauthConfig, err := google.ConfigFromJSON(credentials, calendarapi.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("parse google oauth credentials file: %w", err)
	}

	token, err := readOAuthToken(opts.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("read google oauth token file: %w", err)
	}

	service, err := calendarapi.NewService(ctx, option.WithHTTPClient(oauthConfig.Client(ctx, token)))
	if err != nil {
		return nil, fmt.Errorf("create google calendar service: %w", err)
	}

	return &GoogleCalendarClient{
		service:    service,
		calendarID: opts.CalendarID,
		timeZone:   normalizeTimeZone(opts.TimeZone),
	}, nil
}

// CreateEvent creates a Google Calendar event.
func (c *GoogleCalendarClient) CreateEvent(
	ctx context.Context,
	command GoogleCalendarCreateEventCommand,
) (GoogleCalendarEventResult, error) {
	calendarID := firstNonEmpty(command.CalendarID, c.calendarID)
	timeZoneName := normalizeTimeZone(firstNonEmpty(command.TimeZone, c.timeZone))

	startAt, err := parseCalendarDateTime(command.StartAt, timeZoneName)
	if err != nil {
		return GoogleCalendarEventResult{}, fmt.Errorf("parse startAt: %w", err)
	}

	endAt, err := parseCalendarDateTime(command.EndAt, timeZoneName)
	if err != nil {
		return GoogleCalendarEventResult{}, fmt.Errorf("parse endAt: %w", err)
	}

	event := &calendarapi.Event{
		Summary:  command.Summary,
		Location: command.Location,
		Start: &calendarapi.EventDateTime{
			DateTime: startAt.Format(time.RFC3339),
			TimeZone: timeZoneName,
		},
		End: &calendarapi.EventDateTime{
			DateTime: endAt.Format(time.RFC3339),
			TimeZone: timeZoneName,
		},
	}

	created, err := c.service.Events.Insert(calendarID, event).Context(ctx).Do()
	if err != nil {
		return GoogleCalendarEventResult{}, fmt.Errorf("insert google calendar event: %w", err)
	}

	return GoogleCalendarEventResult{
		CalendarID: calendarID,
		EventID:    created.Id,
		HTMLLink:   created.HtmlLink,
	}, nil
}

func readOAuthToken(path string) (*oauth2.Token, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var token oauth2.Token
	if err := json.NewDecoder(file).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func parseCalendarDateTime(value, timeZoneName string) (time.Time, error) {
	location, err := time.LoadLocation(timeZoneName)
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(googleCalendarDateTimeLayout, value, location)
}

func normalizeTimeZone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "Asia/Taipei"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
