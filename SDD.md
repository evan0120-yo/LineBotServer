# LineBot Backend SDD

## Purpose

這份文件只定義系統內部怎麼切、怎麼接、誰負責什麼。

```text
SDD scope
├─ module boundary
├─ dependency direction
├─ runtime flow
├─ contracts
└─ ownership
```

這份文件不處理：
- 大量驗收場景
- bug review
- 目前 code 導覽口吻
- 測試案例清單

## System Overview

新的目標架構是 Calendar-only，LineBot Backend 不再依賴 Firestore。

```text
LineBot Backend
├─ transport boundary
│  ├─ REST API
│  └─ LINE webhook
├─ task orchestration
│  ├─ call Internal AI Copilot
│  ├─ receive structured JSON
│  └─ dispatch by operation
├─ operation factory
│  ├─ create
│  ├─ query
│  ├─ delete
│  └─ update
└─ external system
   └─ Google Calendar
```

## Package Architecture

```text
Backend/
├─ cmd/api
│  └─ process entrypoint
└─ internal
   ├─ app
   │  └─ config / module wiring / router
   ├─ gatekeeper
   │  └─ REST and LINE webhook boundary
   ├─ task
   │  └─ Internal extraction + operation dispatch
   ├─ calendar
   │  └─ Google Calendar feature module
   ├─ internalclient
   │  └─ Internal AI Copilot gRPC client
   └─ infra
      └─ config / errors / response / Google Calendar client
```

## Module Responsibilities

```text
app
├─ load config
├─ create Internal gRPC client
├─ create Google Calendar client
└─ wire HTTP handlers and operation modules

gatekeeper
├─ parse request
├─ validate boundary input
├─ verify LINE signature
├─ clean mention text
└─ map transport request to task usecase

task
├─ build Internal LineTaskConsult request
├─ validate taskType / operation
├─ normalize extraction result
└─ dispatch to operation factory

calendar
├─ create event
├─ query events by time range
├─ delete event by eventId
├─ update event title by eventId
└─ format reply payload

internalclient
├─ hide protobuf details
└─ map Internal gRPC request / response

infra
├─ shared config
├─ business errors
├─ HTTP envelope
└─ Google Calendar adapter
```

## Dependency Direction

```text
cmd/api
└─ app
   ├─ gatekeeper
   │  └─ task
   │     ├─ internalclient
   │     └─ calendar
   │        └─ infra
   └─ infra
```

Allowed:

```text
gatekeeper -> task
task -> internalclient
task -> calendar
calendar -> infra
internalclient -> infra
app -> all modules for wiring
```

Avoid:

```text
calendar -> task
internalclient -> calendar
handler -> google sdk
handler -> repository-like persistence
```

## Main Runtime Flow

### REST

```text
POST /api/tasks
│
▼
gatekeeper.Handler
├─ decode request
├─ validate text
└─ call gatekeeper.UseCase
   │
   ▼
gatekeeper.UseCase
└─ call task.UseCase.ExecuteFromText
```

### LINE webhook

```text
POST /api/line/webhook
│
▼
gatekeeper.LineHandler
├─ verify signature
├─ parse webhook JSON
├─ filter text message event
├─ check bot mention
├─ remove mention text
└─ call same gatekeeper.UseCase
```

### Shared task flow

```text
task.UseCase
├─ build Internal LineTaskConsult request
│  ├─ appId
│  ├─ builderId
│  ├─ messageText
│  ├─ referenceTime
│  ├─ timeZone
│  └─ supportedTaskTypes=["calendar"]
│
├─ call internalclient.Service.LineTaskConsult
├─ validate taskType
├─ validate operation
└─ dispatch by operation
   ├─ create -> calendar.UseCase.Create
   ├─ query  -> calendar.UseCase.Query
   ├─ delete -> calendar.UseCase.Delete
   └─ update -> calendar.UseCase.Update
```

## Operation Factory

目前工廠不是 taskType 工廠，而是 operation 工廠。

```text
calendar operation factory
├─ create
│  └─ create Google Calendar event
├─ query
│  └─ list events by time range + overlap filter
├─ delete
│  └─ delete by eventId
└─ update
   └─ update title by eventId
```

Rules:
- 第一版只支援 `taskType="calendar"`。
- operation factory 必須至少支援 `create / query / delete / update`。
- delete / update 不再依賴 Firestore 查 mapping。

## Internal Contract

LineBot Backend 仍把 AI 理解交給 Internal。

### Request

```text
LineTaskConsultRequest
├─ appId
├─ builderId
├─ messageText
├─ referenceTime
├─ timeZone
├─ supportedTaskTypes[]
└─ clientIp
```

### Response

```text
LineTaskConsultResponse
├─ taskType
├─ operation
├─ eventId
├─ summary
├─ startAt
├─ endAt
├─ queryStartAt
├─ queryEndAt
├─ location
└─ missingFields[]
```

Rules:
- Internal prompt return 需要補 `eventId` 欄位。
- `create` 時，Internal 回傳的 `eventId` 可為空字串；真正 eventId 由 Google Calendar create 結果決定。
- `delete` / `update` 時，LineBot Backend 直接使用 JSON 內的 `eventId`。
- `query` 時，主查詢條件是 `queryStartAt` 與 `queryEndAt`。

## Google Calendar Contract

### Create

```text
input
├─ calendarId
├─ summary
├─ startAt
├─ endAt
├─ timeZone
└─ location optional

output
├─ eventId
├─ summary
├─ startAt
├─ endAt
└─ htmlLink optional
```

### Query

```text
input
├─ calendarId
├─ queryStartAt
├─ queryEndAt
└─ timeZone

output events[]
├─ eventId
├─ summary
├─ startAt
├─ endAt
└─ location
```

### Delete

```text
input
├─ calendarId
└─ eventId
```

### Update

```text
input
├─ calendarId
├─ eventId
└─ summary
```

## Query Design

### Query Strategy

query 主條件只用時間區間，不用 title 搜尋。

```text
query strategy
├─ fetch candidate events by time window
└─ apply overlap rule in LineBot Backend
```

### Overlap Rule

```text
match when
eventStart <= queryEnd
AND
eventEnd >= queryStart
```

這代表：
- 事件完全包住查詢區間，要查得到
- 查詢區間包住事件，要查得到
- 只要時間邊界相接，也算查得到

## Reply Formatting Contract

LineBot 最終回覆不是 raw JSON，而是格式化文字。

### Single event

```text
0001
小傑約明天吃晚餐
2026-04-18 12:00 (週五) ~ 2026-04-18 12:30 (週五)
```

### Multiple events

```text
0001
小傑約明天吃晚餐
2026-04-18 12:00 (週五) ~ 2026-04-18 12:30 (週五)

0002
回診
2026-04-18 15:00 (週五) ~ 2026-04-18 15:30 (週五)
```

Rules:
- `eventId` 不加前綴。
- 沒資料固定回 `沒資料`。
- 錯誤回精簡文字，不回完整 log。

## LINE Webhook Design

```text
LINE webhook integration
├─ POST /api/line/webhook
├─ verify LINE signature
├─ parse webhook events
├─ filter text message
├─ require bot mention
├─ remove mention text
└─ reuse same task usecase
```

Rules:
- group/private 第一版都要求 mention。
- mention 只負責觸發閘門，不負責任務分類。
- LINE handler 不做 operation 判斷；operation 由 Internal 回傳後交給 task 層處理。

## Config

```text
required config
├─ LINEBOT_INTERNAL_GRPC_ADDR
├─ LINEBOT_INTERNAL_APP_ID
├─ LINEBOT_INTERNAL_BUILDER_ID
├─ LINEBOT_GOOGLE_CALENDAR_ENABLED
├─ LINEBOT_GOOGLE_CALENDAR_ID
├─ LINEBOT_GOOGLE_CALENDAR_TIME_ZONE
├─ LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE
├─ LINEBOT_GOOGLE_OAUTH_TOKEN_FILE
├─ LINEBOT_LINE_CHANNEL_SECRET
├─ LINEBOT_LINE_CHANNEL_ACCESS_TOKEN
└─ LINEBOT_LINE_BOT_USER_ID
```

Removed from target design:

```text
removed
└─ all Firestore-related config and persistence flow
```
