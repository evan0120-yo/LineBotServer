package calendar

import "testing"

func TestServiceFilterOverlappingEvents(t *testing.T) {
	service := NewService()
	events := []Event{
		{
			EventID: "event-1",
			Summary: "午餐",
			StartAt: "2026-04-18 12:00:00",
			EndAt:   "2026-04-18 15:00:00",
		},
	}

	testCases := []struct {
		name         string
		queryStartAt string
		queryEndAt   string
		wantCount    int
	}{
		{name: "event fully contains query", queryStartAt: "2026-04-18 12:00:00", queryEndAt: "2026-04-18 12:30:00", wantCount: 1},
		{name: "overlap at right boundary", queryStartAt: "2026-04-18 14:30:00", queryEndAt: "2026-04-18 15:30:00", wantCount: 1},
		{name: "overlap at left boundary", queryStartAt: "2026-04-18 11:30:00", queryEndAt: "2026-04-18 12:00:00", wantCount: 1},
		{name: "no overlap", queryStartAt: "2026-04-18 15:01:00", queryEndAt: "2026-04-18 16:00:00", wantCount: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := service.FilterOverlappingEvents(events, QueryCommand{
				QueryStartAt: testCase.queryStartAt,
				QueryEndAt:   testCase.queryEndAt,
			}, "Asia/Taipei")
			if err != nil {
				t.Fatalf("FilterOverlappingEvents returned error: %v", err)
			}
			if len(result) != testCase.wantCount {
				t.Fatalf("event count = %d, want %d", len(result), testCase.wantCount)
			}
		})
	}
}

func TestServiceFormatEventsReply(t *testing.T) {
	service := NewService()
	replyText, err := service.FormatEventsReply([]Event{
		{
			EventID: "event-1",
			Summary: "小傑約明天吃晚餐",
			StartAt: "2026-04-18 12:00:00",
			EndAt:   "2026-04-18 12:30:00",
		},
	}, "Asia/Taipei")
	if err != nil {
		t.Fatalf("FormatEventsReply returned error: %v", err)
	}

	expected := "event-1\n小傑約明天吃晚餐\n2026-04-18 12:00 (週六) ~ 2026-04-18 12:30 (週六)"
	if replyText != expected {
		t.Fatalf("replyText = %q, want %q", replyText, expected)
	}
}

func TestServiceFormatEventsReplyNoData(t *testing.T) {
	service := NewService()
	replyText, err := service.FormatEventsReply(nil, "Asia/Taipei")
	if err != nil {
		t.Fatalf("FormatEventsReply returned error: %v", err)
	}
	if replyText != "沒資料" {
		t.Fatalf("replyText = %q, want 沒資料", replyText)
	}
}
