package task

import (
	"context"
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

// CreateFromText extracts a task from natural language text and dispatches by operation.
func (u *UseCase) CreateFromText(ctx context.Context, command CreateFromTextCommand) (TaskResult, error) {
	log.Printf("[INFO] task create start: source=%s text=%q referenceTime=%q timeZone=%q clientIP=%q", command.Source, command.Text, command.ReferenceTime, command.TimeZone, command.ClientIP)

	internalCommand := internalclient.LineTaskConsultCommand{
		AppID:              u.config.InternalAppID,
		BuilderID:          u.config.InternalBuilderID,
		MessageText:        command.Text,
		ReferenceTime:      command.ReferenceTime,
		TimeZone:           command.TimeZone,
		SupportedTaskTypes: SupportedTaskTypes(),
		ClientIP:           command.ClientIP,
	}

	log.Printf("[INFO] task calling internal grpc: appID=%s builderID=%d supportedTaskTypes=%v", internalCommand.AppID, internalCommand.BuilderID, internalCommand.SupportedTaskTypes)
	extractionResult, err := u.internalClient.LineTaskConsult(ctx, internalCommand)
	if err != nil {
		log.Printf("[INFO] task internal grpc failed: err=%v", err)
		return TaskResult{}, err
	}
	log.Printf("[INFO] task internal grpc success: taskType=%s operation=%s eventID=%q summary=%q startAt=%q endAt=%q queryStartAt=%q queryEndAt=%q missingFields=%v", extractionResult.TaskType, extractionResult.Operation, extractionResult.EventID, extractionResult.Summary, extractionResult.StartAt, extractionResult.EndAt, extractionResult.QueryStartAt, extractionResult.QueryEndAt, extractionResult.MissingFields)

	if err := u.service.ValidateTaskType(extractionResult.TaskType); err != nil {
		log.Printf("[INFO] task validate taskType failed: taskType=%s err=%v", extractionResult.TaskType, err)
		return TaskResult{}, err
	}
	if err := u.service.ValidateOperation(extractionResult.Operation); err != nil {
		log.Printf("[INFO] task validate operation failed: operation=%s err=%v", extractionResult.Operation, err)
		return TaskResult{}, err
	}

	log.Printf("[INFO] task dispatching: taskType=%s operation=%s", extractionResult.TaskType, extractionResult.Operation)
	switch TaskType(extractionResult.TaskType) {
	case TaskTypeCalendar:
		return u.executeCalendarOperation(ctx, extractionResult)
	default:
		return TaskResult{}, infra.NewTaskTypeUnsupportedError(extractionResult.TaskType)
	}
}

func (u *UseCase) executeCalendarOperation(ctx context.Context, extractionResult internalclient.LineTaskConsultResult) (TaskResult, error) {
	switch extractionResult.Operation {
	case "create":
		return u.createCalendarEvent(ctx, extractionResult)
	case "query":
		return u.queryCalendarEvents(ctx, extractionResult)
	case "delete":
		return u.deleteCalendarEvent(ctx, extractionResult)
	case "update":
		return u.updateCalendarEvent(ctx, extractionResult)
	default:
		return TaskResult{}, infra.NewOperationUnsupportedError(extractionResult.Operation)
	}
}

func (u *UseCase) createCalendarEvent(ctx context.Context, extractionResult internalclient.LineTaskConsultResult) (TaskResult, error) {
	event, err := u.calendarUseCase.Create(ctx, calendar.CreateCommand{
		Summary:  extractionResult.Summary,
		StartAt:  extractionResult.StartAt,
		EndAt:    extractionResult.EndAt,
		Location: extractionResult.Location,
	})
	if err != nil {
		log.Printf("[INFO] task calendar create failed: err=%v", err)
		return TaskResult{}, err
	}

	replyText, err := calendar.NewService().FormatEventsReply([]calendar.Event{event}, u.config.GoogleCalendarTimeZone)
	if err != nil {
		log.Printf("[INFO] task calendar create format failed: err=%v", err)
		return TaskResult{}, infra.NewError("CALENDAR_REPLY_FORMAT_FAILED", "格式化行事曆回覆失敗", 500)
	}

	result := TaskResult{
		Operation: "create",
		ReplyText: replyText,
		Events:    mapEvents([]calendar.Event{event}),
	}
	log.Printf("[INFO] task create completed: operation=%s eventID=%s", result.Operation, event.EventID)
	return result, nil
}

func (u *UseCase) queryCalendarEvents(ctx context.Context, extractionResult internalclient.LineTaskConsultResult) (TaskResult, error) {
	events, err := u.calendarUseCase.Query(ctx, calendar.QueryCommand{
		QueryStartAt: extractionResult.QueryStartAt,
		QueryEndAt:   extractionResult.QueryEndAt,
	})
	if err != nil {
		log.Printf("[INFO] task calendar query failed: err=%v", err)
		return TaskResult{}, err
	}

	replyText, err := calendar.NewService().FormatEventsReply(events, u.config.GoogleCalendarTimeZone)
	if err != nil {
		log.Printf("[INFO] task calendar query format failed: err=%v", err)
		return TaskResult{}, infra.NewError("CALENDAR_REPLY_FORMAT_FAILED", "格式化行事曆回覆失敗", 500)
	}

	result := TaskResult{
		Operation: "query",
		ReplyText: replyText,
		Events:    mapEvents(events),
	}
	log.Printf("[INFO] task query completed: operation=%s eventCount=%d", result.Operation, len(result.Events))
	return result, nil
}

func (u *UseCase) deleteCalendarEvent(ctx context.Context, extractionResult internalclient.LineTaskConsultResult) (TaskResult, error) {
	if err := u.calendarUseCase.Delete(ctx, calendar.DeleteCommand{
		EventID: extractionResult.EventID,
	}); err != nil {
		log.Printf("[INFO] task calendar delete failed: err=%v", err)
		return TaskResult{}, err
	}

	result := TaskResult{
		Operation: "delete",
		ReplyText: calendar.NewService().FormatDeleteSuccessReply(),
	}
	log.Printf("[INFO] task delete completed: operation=%s eventID=%s", result.Operation, extractionResult.EventID)
	return result, nil
}

func (u *UseCase) updateCalendarEvent(ctx context.Context, extractionResult internalclient.LineTaskConsultResult) (TaskResult, error) {
	event, err := u.calendarUseCase.Update(ctx, calendar.UpdateCommand{
		EventID:  extractionResult.EventID,
		Summary:  extractionResult.Summary,
		Location: extractionResult.Location,
	})
	if err != nil {
		log.Printf("[INFO] task calendar update failed: err=%v", err)
		return TaskResult{}, err
	}

	replyText, err := calendar.NewService().FormatEventsReply([]calendar.Event{event}, u.config.GoogleCalendarTimeZone)
	if err != nil {
		log.Printf("[INFO] task calendar update format failed: err=%v", err)
		return TaskResult{}, infra.NewError("CALENDAR_REPLY_FORMAT_FAILED", "格式化行事曆回覆失敗", 500)
	}

	result := TaskResult{
		Operation: "update",
		ReplyText: replyText,
		Events:    mapEvents([]calendar.Event{event}),
	}
	log.Printf("[INFO] task update completed: operation=%s eventID=%s", result.Operation, event.EventID)
	return result, nil
}

func mapEvents(events []calendar.Event) []TaskEvent {
	mapped := make([]TaskEvent, 0, len(events))
	for _, event := range events {
		mapped = append(mapped, TaskEvent{
			EventID:  event.EventID,
			Summary:  event.Summary,
			StartAt:  event.StartAt,
			EndAt:    event.EndAt,
			Location: event.Location,
		})
	}
	return mapped
}
