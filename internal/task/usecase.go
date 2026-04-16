package task

import (
	"context"
	"encoding/json"
	"log"

	"linebot-backend/internal/calendar"
	"linebot-backend/internal/infra"
	"linebot-backend/internal/internalclient"
)

// UseCase orchestrates AI task extraction and dispatch.
type UseCase struct {
	service         *Service
	internalClient  *internalclient.Service
	calendarUseCase *calendar.UseCase
	config          infra.Config
}

// NewUseCase creates a new task UseCase.
func NewUseCase(
	service *Service,
	internalClient *internalclient.Service,
	calendarUseCase *calendar.UseCase,
	config infra.Config,
) *UseCase {
	return &UseCase{
		service:         service,
		internalClient:  internalClient,
		calendarUseCase: calendarUseCase,
		config:          config,
	}
}

// CreateFromText creates a task from natural language text.
// Calls Internal AI Copilot for extraction, validates the result, and dispatches to the appropriate feature module.
func (u *UseCase) CreateFromText(ctx context.Context, command CreateFromTextCommand) (TaskResult, error) {
	log.Printf("[INFO] task create start: source=%s text=%q referenceTime=%q timeZone=%q clientIP=%q", command.Source, command.Text, command.ReferenceTime, command.TimeZone, command.ClientIP)

	// 1. Build Internal LineTaskConsult command
	internalCommand := internalclient.LineTaskConsultCommand{
		AppID:              u.config.InternalAppID,
		BuilderID:          u.config.InternalBuilderID,
		MessageText:        command.Text,
		ReferenceTime:      command.ReferenceTime,
		TimeZone:           command.TimeZone,
		SupportedTaskTypes: SupportedTaskTypes(),
		ClientIP:           command.ClientIP,
	}

	// 2. Call Internal AI Copilot
	log.Printf("[INFO] task calling internal grpc: appID=%s builderID=%d supportedTaskTypes=%v", internalCommand.AppID, internalCommand.BuilderID, internalCommand.SupportedTaskTypes)
	extractionResult, err := u.internalClient.LineTaskConsult(ctx, internalCommand)
	if err != nil {
		log.Printf("[INFO] task internal grpc failed: err=%v", err)
		return TaskResult{}, err
	}
	log.Printf("[INFO] task internal grpc success: taskType=%s operation=%s summary=%q startAt=%q endAt=%q missingFields=%v", extractionResult.TaskType, extractionResult.Operation, extractionResult.Summary, extractionResult.StartAt, extractionResult.EndAt, extractionResult.MissingFields)

	// 3. Validate taskType
	if err := u.service.ValidateTaskType(extractionResult.TaskType); err != nil {
		log.Printf("[INFO] task validate taskType failed: taskType=%s err=%v", extractionResult.TaskType, err)
		return TaskResult{}, err
	}

	// 4. Validate operation
	if err := u.service.ValidateOperation(extractionResult.Operation); err != nil {
		log.Printf("[INFO] task validate operation failed: operation=%s err=%v", extractionResult.Operation, err)
		return TaskResult{}, err
	}
	log.Printf("[INFO] task dispatching: taskType=%s operation=%s", extractionResult.TaskType, extractionResult.Operation)

	// 5. Dispatch by taskType
	switch TaskType(extractionResult.TaskType) {
	case TaskTypeCalendar:
		return u.createCalendarTask(ctx, command, internalCommand, extractionResult)
	default:
		// Should not happen due to ValidateTaskType, but handle defensively
		return TaskResult{}, infra.NewTaskTypeUnsupportedError(extractionResult.TaskType)
	}
}

func (u *UseCase) createCalendarTask(
	ctx context.Context,
	command CreateFromTextCommand,
	internalCommand internalclient.LineTaskConsultCommand,
	extractionResult internalclient.LineTaskConsultResult,
) (TaskResult, error) {
	log.Printf("[INFO] task calendar create: summary=%q startAt=%q endAt=%q location=%q", extractionResult.Summary, extractionResult.StartAt, extractionResult.EndAt, extractionResult.Location)

	// Build calendar.CreateCommand
	calendarCommand := calendar.CreateCommand{
		Source:            command.Source,
		RawText:           command.Text,
		TaskType:          extractionResult.TaskType,
		Operation:         extractionResult.Operation,
		Summary:           extractionResult.Summary,
		StartAt:           extractionResult.StartAt,
		EndAt:             extractionResult.EndAt,
		Location:          extractionResult.Location,
		MissingFields:     extractionResult.MissingFields,
		InternalAppID:     u.config.InternalAppID,
		InternalBuilderID: u.config.InternalBuilderID,
		InternalRequest:   serializeToJSON(internalCommand),
		InternalResponse:  serializeToJSON(extractionResult),
	}

	// Call calendar usecase
	calendarTask, err := u.calendarUseCase.Create(ctx, calendarCommand)
	if err != nil {
		log.Printf("[INFO] task calendar create failed: err=%v", err)
		return TaskResult{}, err
	}
	log.Printf("[INFO] task calendar create success: taskID=%s syncStatus=%s googleEventID=%s", calendarTask.TaskID, calendarTask.CalendarSyncStatus, calendarTask.GoogleCalendarEventID)

	// Map to TaskResult
	result := TaskResult{
		TaskID:                 calendarTask.TaskID,
		Operation:              extractionResult.Operation,
		Summary:                calendarTask.Summary,
		StartAt:                calendarTask.StartAt,
		EndAt:                  calendarTask.EndAt,
		Location:               calendarTask.Location,
		MissingFields:          calendarTask.MissingFields,
		CalendarSyncStatus:     calendarTask.CalendarSyncStatus,
		GoogleCalendarID:       calendarTask.GoogleCalendarID,
		GoogleCalendarEventID:  calendarTask.GoogleCalendarEventID,
		GoogleCalendarHTMLLink: calendarTask.GoogleCalendarHTMLLink,
		CalendarSyncError:      calendarTask.CalendarSyncError,
	}
	log.Printf("[INFO] task create completed: taskID=%s operation=%s", result.TaskID, result.Operation)

	return result, nil
}

func serializeToJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
