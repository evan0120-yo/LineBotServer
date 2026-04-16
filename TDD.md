# LineBot Backend TDD

## Purpose

這份文件列出目前已實作的測試，反映實際測試狀態。

```text
test scope
├─ 已補測試：核心邏輯鎖住（Google Calendar sync + error handling + LINE webhook boundary）
└─ 未補測試：其餘簡單邏輯透過手動驗證
```

## Test Strategy

第一版測試重點放在最容易出錯的邏輯：

```text
test priority
├─ calendar usecase
│  └─ Google Calendar sync 三種狀態（enabled/disabled/failed）
│     └─ 確保 sync failure 不會導致 Firestore task 消失
│
├─ infra http
│  └─ error handling 細節（missingFields / JSON strict decode）
│
└─ gatekeeper LINE webhook
   └─ request-level ack / signature / mention cleanup / source mapping
```

## Implemented Tests

### calendar usecase

```text
calendar.UseCase tests
├─ create with sync disabled
│  └─ provider 不應被呼叫
│  └─ calendarSyncStatus = not_enabled
│
├─ create with sync enabled
│  └─ provider 應被呼叫
│  └─ sync result 應寫回 Firestore
│  └─ calendarSyncStatus = calendar_synced
│
├─ sync failure without dropping task
│  └─ Firestore task 應保留
│  └─ calendarSyncStatus = calendar_sync_failed
│  └─ calendarSyncError 應保存
│
└─ required time missing
   └─ startAt/endAt 缺失應拒絕
   └─ repository 不應被呼叫
```

**Test files**: `internal/calendar/usecase_test.go`

**Test functions**:
- `TestUseCaseCreateWhenGoogleCalendarDisabled`
- `TestUseCaseCreateSyncsGoogleCalendar`
- `TestUseCaseCreateMarksSyncFailedWithoutDroppingTask`
- `TestUseCaseCreateReturnsErrorWhenRequiredTimeMissing`

### infra http

```text
infra.HTTP tests
├─ DecodeJSONStrict rejects trailing JSON
│  └─ 確保只接受單一 JSON object
│
├─ NewInternalExtractionIncompleteError keeps missingFields
│  └─ BusinessError 應保留 missingFields
│
└─ WriteError includes missingFields
   └─ HTTP response error 應包含 missingFields
```

**Test files**: `internal/infra/http_test.go`

**Test functions**:
- `TestDecodeJSONStrictRejectsTrailingJSONObject`
- `TestNewInternalExtractionIncompleteErrorKeepsMissingFields`
- `TestWriteErrorIncludesMissingFields`

### gatekeeper LINE webhook

```text
gatekeeper.LineHandler tests
├─ invalid signature
│  └─ should return 401 and skip task usecase
├─ mention cleanup
│  └─ should remove mention text and pass source="line"
├─ no bot mention
│  └─ should ignore event without calling task usecase
└─ mixed event outcomes
   └─ should still ack 200 and continue later events
```

**Test files**: `internal/gatekeeper/line_handler_test.go`

**Test functions**:
- `TestLineHandlerServeHTTPRejectsInvalidSignature`
- `TestLineHandlerServeHTTPCleansMentionAndUsesLineSource`
- `TestLineHandlerServeHTTPIgnoresMessageWithoutBotMention`
- `TestLineHandlerServeHTTPAcknowledgesWebhookWhenSomeEventsFail`

## Not Tested (Manual Verification)

以下邏輯透過手動驗證，未寫自動化測試：

```text
not tested
├─ gatekeeper (REST API)
│  └─ text validation / request mapping
│     └─ 簡單 if 判斷，手動測試即可
│
├─ task
│  └─ taskType / operation validation / dispatch
│     └─ switch case 邏輯，手動測試即可
│
└─ repository
   └─ Firestore mapping
      └─ 需要 emulator，成本效益不高
```

## Test Coverage Summary

```text
coverage
├─ calendar usecase: 4 tests (Google Calendar sync 邏輯)
├─ infra http: 3 tests (error handling 細節)
├─ gatekeeper line webhook: 4 tests (signature / mention / ack 邏輯)
└─ others: manual verification
```

第一版測試策略：**鎖住最容易出錯的邏輯（Google Calendar sync + webhook boundary），其他透過手動驗證**。

---

**文件版本**：v2.0
**最後更新**：2026-04-16
**測試覆蓋**：核心邏輯鎖住，簡單邏輯手動驗證
