package infra

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONStrictRejectsTrailingJSONObject(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/tasks", strings.NewReader(`{"text":"a"}{"text":"b"}`))
	w := httptest.NewRecorder()

	var body struct {
		Text string `json:"text"`
	}

	err := DecodeJSONStrict(w, req, &body, 1024)
	if err == nil {
		t.Fatal("DecodeJSONStrict should reject trailing JSON object")
	}

	bizErr := AsBusinessError(err)
	if bizErr == nil {
		t.Fatalf("error = %T, want BusinessError", err)
	}
	if bizErr.Code != "INVALID_JSON" {
		t.Fatalf("code = %q, want INVALID_JSON", bizErr.Code)
	}
}

func TestNewInternalExtractionIncompleteErrorKeepsMissingFields(t *testing.T) {
	err := NewInternalExtractionIncompleteError([]string{"startAt", "endAt"})

	bizErr := AsBusinessError(err)
	if bizErr == nil {
		t.Fatalf("error = %T, want BusinessError", err)
	}

	if len(bizErr.MissingFields) != 2 {
		t.Fatalf("missingFields length = %d, want 2", len(bizErr.MissingFields))
	}
	if bizErr.MissingFields[0] != "startAt" || bizErr.MissingFields[1] != "endAt" {
		t.Fatalf("missingFields = %#v", bizErr.MissingFields)
	}
}

func TestWriteErrorIncludesMissingFields(t *testing.T) {
	w := httptest.NewRecorder()
	err := NewInternalExtractionIncompleteError([]string{"startAt", "endAt"})

	WriteError(w, err)

	var response APIResponse
	if decodeErr := json.NewDecoder(w.Body).Decode(&response); decodeErr != nil {
		t.Fatalf("Decode response failed: %v", decodeErr)
	}

	if response.Success {
		t.Fatal("success should be false")
	}
	if response.Error == nil {
		t.Fatal("error should be present")
	}
	if len(response.Error.MissingFields) != 2 {
		t.Fatalf("missingFields length = %d, want 2", len(response.Error.MissingFields))
	}
	if response.Error.MissingFields[0] != "startAt" || response.Error.MissingFields[1] != "endAt" {
		t.Fatalf("missingFields = %#v", response.Error.MissingFields)
	}
}
