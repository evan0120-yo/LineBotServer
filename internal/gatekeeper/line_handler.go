package gatekeeper

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"linebot-backend/internal/infra"
)

const lineEventProcessingTimeout = 30 * time.Second

// LineHandler handles LINE webhook requests.
type LineHandler struct {
	useCase       *UseCase
	channelSecret string
	botUserID     string
}

// NewLineHandler creates a new LINE webhook handler.
func NewLineHandler(useCase *UseCase, channelSecret string, botUserID string) *LineHandler {
	return &LineHandler{
		useCase:       useCase,
		channelSecret: channelSecret,
		botUserID:     botUserID,
	}
}

// ServeHTTP handles LINE webhook POST requests.
func (h *LineHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[INFO] line webhook read body failed: err=%v", err)
		infra.WriteError(w, infra.NewError("INVALID_REQUEST", "Failed to read request body", http.StatusBadRequest))
		return
	}

	// Verify LINE signature
	signature := r.Header.Get("x-line-signature")
	if !h.verifySignature(body, signature) {
		log.Printf("[INFO] line webhook signature verification failed: remote=%s", r.RemoteAddr)
		infra.WriteError(w, infra.NewError("INVALID_SIGNATURE", "LINE signature verification failed", http.StatusUnauthorized))
		return
	}

	// Parse LINE webhook JSON
	var webhookReq LineWebhookRequest
	if err := json.Unmarshal(body, &webhookReq); err != nil {
		log.Printf("[INFO] line webhook invalid json: err=%v", err)
		infra.WriteError(w, infra.NewError("INVALID_JSON", "Failed to parse LINE webhook JSON", http.StatusBadRequest))
		return
	}
	log.Printf("[INFO] line webhook received: events=%d remote=%s", len(webhookReq.Events), r.RemoteAddr)

	// Process all events and always acknowledge the webhook once signature and JSON are valid.
	// Event-level failures should not cause LINE to retry the whole webhook payload.
	processedCount := 0
	successCount := 0
	errorCount := 0

	for i, event := range webhookReq.Events {
		log.Printf("[INFO] line webhook event[%d]: type=%s messageType=%s sourceType=%s sourceUserID=%s", i, event.Type, event.Message.Type, event.Source.Type, event.Source.UserID)

		// Filter message event
		if event.Type != "message" {
			log.Printf("[INFO] line webhook event[%d] skipped: reason=non_message type=%s", i, event.Type)
			continue
		}

		if event.Message.Type != "text" {
			log.Printf("[INFO] line webhook event[%d] skipped: reason=non_text messageType=%s", i, event.Message.Type)
			continue
		}

		// Check if bot is mentioned
		botMention := h.findBotMention(event.Message)
		if botMention == nil {
			log.Printf("[INFO] line webhook event[%d] skipped: reason=bot_not_mentioned configuredBotUserID=%s mentioneeCount=%d", i, h.botUserID, mentionCount(event.Message))
			continue
		}
		log.Printf("[INFO] line webhook event[%d] mention matched: mentionUserID=%s index=%d length=%d", i, botMention.UserID, botMention.Index, botMention.Length)

		// Remove mention text using mention metadata
		cleanedText := h.removeMentionText(event.Message.Text, botMention)
		if cleanedText == "" {
			log.Printf("[INFO] line webhook event[%d] skipped: reason=empty_after_cleanup rawText=%q", i, event.Message.Text)
			continue
		}

		// Validate text not empty after cleaning
		cleanedText = strings.TrimSpace(cleanedText)
		if cleanedText == "" {
			log.Printf("[INFO] line webhook event[%d] skipped: reason=blank_after_trim rawText=%q", i, event.Message.Text)
			continue
		}
		log.Printf("[INFO] line webhook event[%d] accepted: rawText=%q cleanedText=%q", i, event.Message.Text, cleanedText)

		// Build command
		command := CreateTaskCommand{
			Source:        "line",
			Text:          cleanedText,
			ReferenceTime: "", // Use default from Internal
			TimeZone:      "", // Use default from Internal
			ClientIP:      resolveClientIP(r),
		}

		// Call usecase
		processedCount++
		log.Printf("[INFO] line webhook event[%d] calling gatekeeper usecase", i)
		ctx, cancel := context.WithTimeout(context.Background(), lineEventProcessingTimeout)
		_, err := h.useCase.CreateTask(ctx, command)
		cancel()
		if err != nil {
			errorCount++
			log.Printf("[INFO] line webhook event[%d] usecase failed: err=%v", i, err)
			continue
		}

		successCount++
		log.Printf("[INFO] line webhook event[%d] completed successfully", i)
	}
	log.Printf("[INFO] line webhook response: processed=%d success=%d error=%d", processedCount, successCount, errorCount)

	infra.WriteJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"processedCount": processedCount,
		"successCount":   successCount,
		"errorCount":     errorCount,
	})
}

// verifySignature verifies LINE webhook signature.
func (h *LineHandler) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.channelSecret))
	mac.Write(body)
	computed := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(computed))
}

// findBotMention finds the mention of this bot in the message.
// Returns nil if bot is not mentioned.
func (h *LineHandler) findBotMention(message LineMessage) *LineMentionee {
	if message.Mention == nil {
		return nil
	}

	// Check if bot is mentioned by comparing userId
	for _, mentionee := range message.Mention.Mentionees {
		if mentionee.UserID == h.botUserID {
			return &mentionee
		}
	}

	return nil
}

func mentionCount(message LineMessage) int {
	if message.Mention == nil {
		return 0
	}

	return len(message.Mention.Mentionees)
}

// removeMentionText removes mention text using LINE webhook mention metadata.
func (h *LineHandler) removeMentionText(text string, mention *LineMentionee) string {
	if mention == nil {
		return text
	}

	// Use mention.Index and mention.Length to extract mention text
	// Then remove it from the original text
	runes := []rune(text)

	// Validate indices
	if mention.Index < 0 || mention.Index >= len(runes) {
		return text
	}

	endIndex := mention.Index + mention.Length
	if endIndex > len(runes) {
		endIndex = len(runes)
	}

	// Remove mention part
	before := runes[:mention.Index]
	after := runes[endIndex:]

	result := string(before) + string(after)
	return strings.TrimSpace(result)
}
