package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// GoogleCalendarClient performs CRUD-like operations through Google Calendar API.
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
	log.Printf("[INFO] google calendar create start: calendarID=%s summary=%q startAt=%q endAt=%q location=%q timeZone=%s", calendarID, command.Summary, command.StartAt, command.EndAt, command.Location, timeZoneName)

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
		log.Printf("[INFO] google calendar create failed: calendarID=%s err=%v", calendarID, err)
		return GoogleCalendarEventResult{}, fmt.Errorf("insert google calendar event: %w", err)
	}

	result, mapErr := c.mapEvent(calendarID, created, timeZoneName)
	if mapErr != nil {
		log.Printf("[INFO] google calendar create map failed: calendarID=%s err=%v", calendarID, mapErr)
		return GoogleCalendarEventResult{}, mapErr
	}
	log.Printf("[INFO] google calendar create success: calendarID=%s eventID=%s summary=%q startAt=%q endAt=%q", calendarID, result.EventID, result.Summary, result.StartAt, result.EndAt)
	return result, nil
}

// ListEvents queries Google Calendar events by time range.
func (c *GoogleCalendarClient) ListEvents(
	ctx context.Context,
	command GoogleCalendarListEventsCommand,
) ([]GoogleCalendarEventResult, error) {
	calendarID := firstNonEmpty(command.CalendarID, c.calendarID)
	timeZoneName := normalizeTimeZone(firstNonEmpty(command.TimeZone, c.timeZone))
	log.Printf("[INFO] google calendar query start: calendarID=%s queryStartAt=%q queryEndAt=%q timeZone=%s", calendarID, command.QueryStartAt, command.QueryEndAt, timeZoneName)

	queryStartAt, err := parseCalendarDateTime(command.QueryStartAt, timeZoneName)
	if err != nil {
		return nil, fmt.Errorf("parse queryStartAt: %w", err)
	}

	queryEndAt, err := parseCalendarDateTime(command.QueryEndAt, timeZoneName)
	if err != nil {
		return nil, fmt.Errorf("parse queryEndAt: %w", err)
	}

	candidateWindowStart := queryStartAt.AddDate(0, 0, -30)
	candidateWindowEnd := queryEndAt.AddDate(0, 0, 30)

	events, err := c.service.Events.List(calendarID).
		Context(ctx).
		SingleEvents(true).
		OrderBy("startTime").
		TimeMin(candidateWindowStart.Format(time.RFC3339)).
		TimeMax(candidateWindowEnd.Format(time.RFC3339)).
		Do()
	if err != nil {
		log.Printf("[INFO] google calendar query failed: calendarID=%s err=%v", calendarID, err)
		return nil, fmt.Errorf("list google calendar events: %w", err)
	}

	results := make([]GoogleCalendarEventResult, 0, len(events.Items))
	for _, item := range events.Items {
		result, mapErr := c.mapEvent(calendarID, item, timeZoneName)
		if mapErr != nil {
			log.Printf("[INFO] google calendar query map failed: calendarID=%s err=%v", calendarID, mapErr)
			return nil, mapErr
		}
		results = append(results, result)
	}
	log.Printf("[INFO] google calendar query success: calendarID=%s candidateCount=%d", calendarID, len(results))
	return results, nil
}

// DeleteEvent deletes a Google Calendar event.
func (c *GoogleCalendarClient) DeleteEvent(
	ctx context.Context,
	command GoogleCalendarDeleteEventCommand,
) error {
	calendarID := firstNonEmpty(command.CalendarID, c.calendarID)
	if strings.TrimSpace(command.EventID) == "" {
		return fmt.Errorf("google calendar event id is required")
	}
	log.Printf("[INFO] google calendar delete start: calendarID=%s eventID=%s", calendarID, command.EventID)
	if err := c.service.Events.Delete(calendarID, command.EventID).Context(ctx).Do(); err != nil {
		log.Printf("[INFO] google calendar delete failed: calendarID=%s eventID=%s err=%v", calendarID, command.EventID, err)
		return fmt.Errorf("delete google calendar event: %w", err)
	}
	log.Printf("[INFO] google calendar delete success: calendarID=%s eventID=%s", calendarID, command.EventID)
	return nil
}

// UpdateEventSummary updates the title of a Google Calendar event.
func (c *GoogleCalendarClient) UpdateEventSummary(
	ctx context.Context,
	command GoogleCalendarUpdateEventCommand,
) (GoogleCalendarEventResult, error) {
	calendarID := firstNonEmpty(command.CalendarID, c.calendarID)
	timeZoneName := normalizeTimeZone(firstNonEmpty(command.TimeZone, c.timeZone))
	if strings.TrimSpace(command.EventID) == "" {
		return GoogleCalendarEventResult{}, fmt.Errorf("google calendar event id is required")
	}
	log.Printf("[INFO] google calendar update start: calendarID=%s eventID=%s summary=%q timeZone=%s", calendarID, command.EventID, command.Summary, timeZoneName)

	event := &calendarapi.Event{
		Summary: command.Summary,
	}

	updated, err := c.service.Events.Patch(calendarID, command.EventID, event).Context(ctx).Do()
	if err != nil {
		log.Printf("[INFO] google calendar update failed: calendarID=%s eventID=%s err=%v", calendarID, command.EventID, err)
		return GoogleCalendarEventResult{}, fmt.Errorf("update google calendar event: %w", err)
	}

	result, mapErr := c.mapEvent(calendarID, updated, timeZoneName)
	if mapErr != nil {
		log.Printf("[INFO] google calendar update map failed: calendarID=%s eventID=%s err=%v", calendarID, command.EventID, mapErr)
		return GoogleCalendarEventResult{}, mapErr
	}
	log.Printf("[INFO] google calendar update success: calendarID=%s eventID=%s summary=%q startAt=%q endAt=%q", calendarID, result.EventID, result.Summary, result.StartAt, result.EndAt)
	return result, nil
}

func (c *GoogleCalendarClient) mapEvent(calendarID string, event *calendarapi.Event, timeZoneName string) (GoogleCalendarEventResult, error) {
	startAt, err := eventDateTimeToLocalString(event.Start, timeZoneName, false)
	if err != nil {
		return GoogleCalendarEventResult{}, fmt.Errorf("map google calendar start time: %w", err)
	}

	endAt, err := eventDateTimeToLocalString(event.End, timeZoneName, true)
	if err != nil {
		return GoogleCalendarEventResult{}, fmt.Errorf("map google calendar end time: %w", err)
	}

	return GoogleCalendarEventResult{
		CalendarID: calendarID,
		EventID:    event.Id,
		Summary:    strings.TrimSpace(event.Summary),
		StartAt:    startAt,
		EndAt:      endAt,
		Location:   strings.TrimSpace(event.Location),
		HTMLLink:   event.HtmlLink,
	}, nil
}

func eventDateTimeToLocalString(eventDateTime *calendarapi.EventDateTime, timeZoneName string, isEnd bool) (string, error) {
	location, err := time.LoadLocation(normalizeTimeZone(timeZoneName))
	if err != nil {
		return "", err
	}

	if eventDateTime == nil {
		return "", nil
	}

	if strings.TrimSpace(eventDateTime.DateTime) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, eventDateTime.DateTime)
		if parseErr != nil {
			return "", parseErr
		}
		return parsed.In(location).Format(googleCalendarDateTimeLayout), nil
	}

	if strings.TrimSpace(eventDateTime.Date) != "" {
		parsed, parseErr := time.ParseInLocation("2006-01-02", eventDateTime.Date, location)
		if parseErr != nil {
			return "", parseErr
		}
		if isEnd {
			parsed = parsed.Add(-time.Second)
		}
		return parsed.Format(googleCalendarDateTimeLayout), nil
	}

	return "", nil
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
