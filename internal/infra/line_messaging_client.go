package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const lineReplyEndpoint = "https://api.line.me/v2/bot/message/reply"

// LineMessagingClient sends reply messages through LINE Messaging API.
type LineMessagingClient struct {
	channelAccessToken string
	httpClient         *http.Client
}

// NewLineMessagingClient creates a LINE Messaging API reply client.
func NewLineMessagingClient(channelAccessToken string) *LineMessagingClient {
	return &LineMessagingClient{
		channelAccessToken: strings.TrimSpace(channelAccessToken),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// ReplyText sends a text reply message to LINE.
func (c *LineMessagingClient) ReplyText(ctx context.Context, command LineReplyCommand) error {
	if strings.TrimSpace(command.ReplyToken) == "" {
		return fmt.Errorf("line reply token is required")
	}
	if strings.TrimSpace(command.Text) == "" {
		return fmt.Errorf("line reply text is required")
	}
	if c.channelAccessToken == "" {
		return fmt.Errorf("line channel access token is required")
	}

	payload := map[string]any{
		"replyToken": command.ReplyToken,
		"messages": []map[string]string{
			{
				"type": "text",
				"text": command.Text,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal line reply payload: %w", err)
	}
	log.Printf("[INFO] line reply start: replyTokenPrefix=%s textPreview=%q", previewReplyToken(command.ReplyToken), previewText(command.Text, 160))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, lineReplyEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build line reply request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.channelAccessToken)

	response, err := c.httpClient.Do(request)
	if err != nil {
		log.Printf("[INFO] line reply failed before response: err=%v", err)
		return fmt.Errorf("reply line message: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		log.Printf("[INFO] line reply success: status=%s", response.Status)
		return nil
	}

	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
	message := strings.TrimSpace(string(responseBody))
	if message == "" {
		message = response.Status
	}
	log.Printf("[INFO] line reply failed: status=%s body=%q", response.Status, message)
	return fmt.Errorf("reply line message failed: %s", message)
}

func previewReplyToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 8 {
		return trimmed
	}
	return trimmed[:8]
}

func previewText(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if max <= 0 || len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max] + "..."
}
