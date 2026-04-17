package gatekeeper

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"linebot-backend/internal/infra"
	"linebot-backend/internal/task"
)

type fakeTaskCreator struct {
	results  []task.TaskResult
	errors   []error
	commands []task.CreateFromTextCommand
}

func (f *fakeTaskCreator) CreateFromText(_ context.Context, command task.CreateFromTextCommand) (task.TaskResult, error) {
	f.commands = append(f.commands, command)

	index := len(f.commands) - 1
	var result task.TaskResult
	var err error

	if index < len(f.results) {
		result = f.results[index]
	}
	if index < len(f.errors) {
		err = f.errors[index]
	}

	return result, err
}

type fakeReplySender struct {
	commands []infra.LineReplyCommand
	err      error
}

func (f *fakeReplySender) ReplyText(_ context.Context, command infra.LineReplyCommand) error {
	f.commands = append(f.commands, command)
	return f.err
}

func TestLineHandlerServeHTTPRejectsInvalidSignature(t *testing.T) {
	taskCreator := &fakeTaskCreator{}
	replySender := &fakeReplySender{}
	handler := NewLineHandler(NewUseCase(taskCreator), replySender, "secret", "bot-user")

	req := httptest.NewRequest(http.MethodPost, "/api/line/webhook", strings.NewReader(validLineWebhookBody()))
	req.Header.Set("x-line-signature", "invalid")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if len(taskCreator.commands) != 0 {
		t.Fatalf("expected no task calls, got %d", len(taskCreator.commands))
	}
	if len(replySender.commands) != 0 {
		t.Fatalf("expected no reply calls, got %d", len(replySender.commands))
	}
}

func TestLineHandlerServeHTTPCleansMentionAndReplies(t *testing.T) {
	taskCreator := &fakeTaskCreator{
		results: []task.TaskResult{
			{Operation: "create", ReplyText: "event-1\n小傑約明天吃午餐\n2026-04-16 12:00 (週四) ~ 2026-04-16 12:30 (週四)"},
		},
	}
	replySender := &fakeReplySender{}
	handler := NewLineHandler(NewUseCase(taskCreator), replySender, "secret", "bot-user")

	body := `{"events":[{"type":"message","replyToken":"reply-1","message":{"type":"text","text":"@bot 小傑約明天吃午餐","mention":{"mentionees":[{"index":0,"length":5,"userId":"bot-user"}]}},"source":{"type":"group","groupId":"g1","userId":"u1"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/line/webhook", strings.NewReader(body))
	req.Header.Set("x-line-signature", signLineBody("secret", body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(taskCreator.commands) != 1 {
		t.Fatalf("expected 1 task call, got %d", len(taskCreator.commands))
	}
	if len(replySender.commands) != 1 {
		t.Fatalf("expected 1 reply call, got %d", len(replySender.commands))
	}

	command := taskCreator.commands[0]
	if command.Source != "line" {
		t.Fatalf("expected source line, got %q", command.Source)
	}
	if command.Text != "小傑約明天吃午餐" {
		t.Fatalf("expected cleaned text, got %q", command.Text)
	}
	if replySender.commands[0].ReplyToken != "reply-1" {
		t.Fatalf("expected reply token reply-1, got %q", replySender.commands[0].ReplyToken)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Status         string `json:"status"`
			ProcessedCount int    `json:"processedCount"`
			SuccessCount   int    `json:"successCount"`
			ErrorCount     int    `json:"errorCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !response.Success || response.Data.ProcessedCount != 1 || response.Data.SuccessCount != 1 || response.Data.ErrorCount != 0 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestLineHandlerServeHTTPIgnoresMessageWithoutBotMention(t *testing.T) {
	taskCreator := &fakeTaskCreator{}
	replySender := &fakeReplySender{}
	handler := NewLineHandler(NewUseCase(taskCreator), replySender, "secret", "bot-user")

	body := `{"events":[{"type":"message","replyToken":"reply-1","message":{"type":"text","text":"明天吃午餐"},"source":{"type":"user","userId":"u1"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/line/webhook", strings.NewReader(body))
	req.Header.Set("x-line-signature", signLineBody("secret", body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(taskCreator.commands) != 0 {
		t.Fatalf("expected no task calls, got %d", len(taskCreator.commands))
	}
	if len(replySender.commands) != 0 {
		t.Fatalf("expected no reply calls, got %d", len(replySender.commands))
	}
}

func TestLineHandlerServeHTTPAcknowledgesWebhookWhenSomeEventsFail(t *testing.T) {
	taskCreator := &fakeTaskCreator{
		errors: []error{
			errors.New("first failed"),
			nil,
		},
		results: []task.TaskResult{
			{},
			{Operation: "create", ReplyText: "event-2\n第二筆\n2026-04-16 12:00 (週四) ~ 2026-04-16 12:30 (週四)"},
		},
	}
	replySender := &fakeReplySender{}
	handler := NewLineHandler(NewUseCase(taskCreator), replySender, "secret", "bot-user")

	body := `{"events":[{"type":"message","replyToken":"reply-1","message":{"type":"text","text":"@bot 第一筆","mention":{"mentionees":[{"index":0,"length":5,"userId":"bot-user"}]}},"source":{"type":"group","groupId":"g1","userId":"u1"}},{"type":"message","replyToken":"reply-2","message":{"type":"text","text":"@bot 第二筆","mention":{"mentionees":[{"index":0,"length":5,"userId":"bot-user"}]}},"source":{"type":"group","groupId":"g1","userId":"u2"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/line/webhook", strings.NewReader(body))
	req.Header.Set("x-line-signature", signLineBody("secret", body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(taskCreator.commands) != 2 {
		t.Fatalf("expected 2 task calls, got %d", len(taskCreator.commands))
	}
	if len(replySender.commands) != 2 {
		t.Fatalf("expected 2 reply calls, got %d", len(replySender.commands))
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Status         string `json:"status"`
			ProcessedCount int    `json:"processedCount"`
			SuccessCount   int    `json:"successCount"`
			ErrorCount     int    `json:"errorCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success response, got %+v", response)
	}
	if response.Data.ProcessedCount != 2 || response.Data.SuccessCount != 1 || response.Data.ErrorCount != 1 {
		t.Fatalf("unexpected counts: %+v", response.Data)
	}
}

func validLineWebhookBody() string {
	return `{"events":[{"type":"message","replyToken":"reply-1","message":{"type":"text","text":"@bot test","mention":{"mentionees":[{"index":0,"length":5,"userId":"bot-user"}]}},"source":{"type":"user","userId":"u1"}}]}`
}

func signLineBody(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
