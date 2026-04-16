package gatekeeper

import (
	"net/http"
	"strings"

	"linebot-backend/internal/infra"
)

// Handler handles HTTP requests for LineBot REST API.
type Handler struct {
	useCase *UseCase
}

// NewHandler creates a new gatekeeper Handler.
func NewHandler(useCase *UseCase) *Handler {
	return &Handler{
		useCase: useCase,
	}
}

// CreateTask handles POST /api/tasks.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Decode request
	var req CreateTaskRequest
	if err := infra.DecodeJSONStrict(w, r, &req, 1024*1024); err != nil {
		infra.WriteError(w, err)
		return
	}

	// 2. Validate text required
	if strings.TrimSpace(req.Text) == "" {
		infra.WriteError(w, infra.NewTextRequiredError())
		return
	}

	// 3. Resolve client IP
	clientIP := resolveClientIP(r)

	// 4. Build CreateTaskCommand
	command := CreateTaskCommand{
		Source:        "rest",
		Text:          req.Text,
		ReferenceTime: req.ReferenceTime,
		TimeZone:      req.TimeZone,
		ClientIP:      clientIP,
	}

	// 5. Call usecase
	result, err := h.useCase.CreateTask(ctx, command)
	if err != nil {
		infra.WriteError(w, err)
		return
	}

	// 6. Map to response
	responseData := CreateTaskResponseData{
		TaskID:                 result.TaskID,
		Operation:              result.Operation,
		Summary:                result.Summary,
		StartAt:                result.StartAt,
		EndAt:                  result.EndAt,
		Location:               result.Location,
		MissingFields:          result.MissingFields,
		CalendarSyncStatus:     result.CalendarSyncStatus,
		GoogleCalendarID:       result.GoogleCalendarID,
		GoogleCalendarEventID:  result.GoogleCalendarEventID,
		GoogleCalendarHTMLLink: result.GoogleCalendarHTMLLink,
		CalendarSyncError:      result.CalendarSyncError,
	}

	// 7. Write JSON response
	infra.WriteJSON(w, http.StatusOK, responseData)
}

// resolveClientIP resolves the client IP from request headers or remote address.
func resolveClientIP(r *http.Request) string {
	// Try X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Try X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr
	return r.RemoteAddr
}
