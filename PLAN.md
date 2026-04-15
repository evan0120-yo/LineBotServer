# LineBot Backend Plan

## Block 1 - Project Overview

### Project Purpose

LineBot Backend 是獨立專案，目標是把日常對話中的自然語句轉成可執行任務。

第一版先不接正式 LINE webhook。第一版用 REST API 方便 Postman 驗證完整鏈路，calendar task 先寫 Firestore，並可透過方案 C 同步到 Google shared calendar：

```text
Postman
└─ LineBot Backend REST API
   └─ Internal AI Copilot gRPC LineTaskConsult
      └─ Firestore create
         └─ optional Google Calendar events.insert
```

長期目標不是只做行事曆，而是成為「LINE / REST 入口 + Internal AI 任務解析 + 多功能任務執行」的 backend。

```text
LineBot Backend long-term
├─ transport entry
│  ├─ REST for local/dev testing
│  └─ LINE webhook for real chat usage
│
├─ AI task interpretation
│  └─ 交給 Internal AI Copilot
│
├─ task dispatch
│  ├─ calendar
│  ├─ future notes
│  ├─ future reminders
│  ├─ future shopping / household tasks
│  └─ future custom AI tools
│
└─ persistence / integrations
   ├─ Firestore
   └─ Google Calendar shared calendar sync
```

### High-Level Rules

- LineBot Backend 不處理 LinkChat。
- LineBot Backend 不重做 Internal 的 AI pipeline。
- Internal AI Copilot 負責自然語句解析、Gemma 溝通與 structured extraction。
- LineBot Backend 負責入口、任務分派、Firestore persistence、外部服務同步。
- 第一版只支援 `calendar create`。
- Google Calendar 串接採方案 C：OAuth user token 寫入共用 calendar。
- 未來功能增加時，新增同層 module，再由 `task` 分派。
- LineBot Backend 會用本地 task registry 管理可用 task types；第一版只有 `calendar`。
- Internal gRPC request 會帶 `supportedTaskTypes=["calendar"]`，Internal response 會回 `taskType` 供 LineBot Backend 分派。

### First Version Scope

```text
In scope
├─ 建立 RESTful API
├─ 呼叫 Internal gRPC LineTaskConsult
├─ 驗證 summary / startAt / endAt
├─ location optional
├─ operation=create 寫入 Firestore
├─ Google Calendar API create sync（可由 env 關閉）
└─ 回傳 taskId + extraction result + sync result

Out of scope
├─ LINE webhook 正式串接
├─ update / delete / query
├─ 固定前綴指令
├─ LinkChat
└─ 本地 AI 判斷
```

### Google Calendar Shared Calendar

Google Calendar 不直接寫個人裝置本地行事曆，而是寫入一個你與伴侶共同訂閱 / 共用的 Google Calendar。

```text
方案 C
├─ 建立 shared Google Calendar
├─ 你與伴侶都加入 / 訂閱該 calendar
├─ LineBot Backend 使用 OAuth user consent 取得 refresh token
├─ server 寫入 configured shared calendar id
└─ Pixel 上的 Google Calendar app 透過 Google 帳號同步顯示事件
```

決策：
- Firestore 仍是任務 source of truth。
- Google Calendar 是外部同步目標。
- Google Calendar sync 失敗不能讓已解析出的任務資料消失。
- 第一版 sync 只做 `create -> events.insert`。
- `update/delete/query` 等 operation 後續再接。

### Future Behavior Direction

未來接 LINE 時不使用固定前綴。LINE webhook 只用 tag bot 判斷是否要處理。

```text
LINE chat
├─ 沒 tag bot
│  └─ ignore
└─ 有 tag bot
   ├─ 移除 mention
   ├─ 取得自然語句
   └─ 走同一個 task usecase
```

tag bot 是聊天室噪音閘門，不是任務指令。任務語意仍交給 Internal + Gemma 判斷。

## Block 2 - Module Map And Flow

### Package Baseline

本專案參考 Internal AI Copilot Go Backend 的 module-first 風格，不採技術層大 package，也不做完整 DDD 架構。

```text
Backend/
├─ cmd/
│  └─ api/
│     └─ doc.go
│
└─ internal/
   ├─ app/
   │  └─ doc.go
   │
   ├─ gatekeeper/
   │  └─ doc.go
   │
   ├─ task/
   │  └─ doc.go
   │
   ├─ calendar/
   │  └─ doc.go
   │
   ├─ internalclient/
   │  └─ doc.go
   │
   └─ infra/
      └─ doc.go
```

後續實作時，各 module 內再依 Internal 風格新增檔案：

```text
module/
├─ handler.go       optional
├─ usecase.go
├─ service.go
├─ repository.go   optional
├─ model.go
└─ *_test.go
```

### Module Responsibilities

```text
app
├─ process wiring
├─ config load
├─ Firestore store 建立
├─ usecase / service / repository 組裝
└─ HTTP router setup

gatekeeper
├─ REST handler
├─ future LINE webhook handler
├─ request parse
├─ request boundary validation
├─ client source / client IP resolve
└─ 轉交 task usecase

task
├─ AI task 總入口
├─ 呼叫 internalclient 取得 extraction
├─ 傳入 supportedTaskTypes
├─ 判斷 operation / task kind 是否支援
├─ 第一版直接轉交 calendar module
└─ 未來功能增加時，這裡演進出 router / factory

calendar
├─ calendar task usecase
├─ create validation
├─ future update / delete / query
├─ Firestore calendar_tasks persistence
└─ Google Calendar sync

internalclient
├─ Internal AI Copilot gRPC client
├─ LineTaskConsult request mapping
├─ LineTaskConsult response mapping
└─ 不做 calendar 業務判斷

infra
├─ config
├─ shared errors
├─ HTTP response envelope
├─ Firestore store
├─ Google Calendar client
└─ shared persistence / runtime helpers
```

### First Version Data Flow

```text
POST /api/tasks
│
▼
gatekeeper.Handler
├─ parse JSON body
├─ validate text exists
└─ call gatekeeper.UseCase
   │
   ▼
gatekeeper.UseCase
└─ call task.UseCase.CreateFromText
   │
   ▼
task.UseCase
├─ build Internal LineTaskConsult command
├─ call internalclient.Service.LineTaskConsult
│  │
│  ▼
│  Internal AI Copilot
│  └─ Gemma extraction
│     └─ taskType / operation / summary / startAt / endAt / location / missingFields
│
├─ task.Service validates supported task behavior
│  ├─ taskType=calendar supported
│  ├─ operation=create supported
│  ├─ update/delete/query unsupported in first version
│  └─ unknown task type unsupported
│
└─ call calendar.UseCase.Create
   │
   ▼
calendar.UseCase
├─ calendar.Service.ValidateCreate
├─ calendar.Repository.Create
│  └─ Firestore calendar_tasks/{taskId}
└─ optional infra.GoogleCalendarProvider.CreateEvent
   └─ Google Calendar shared calendar
```

### Google Calendar Sync Flow

```text
calendar.UseCase.Create
├─ ValidateCreate
├─ Repository.Create
│  ├─ sync enabled -> calendarSyncStatus = calendar_sync_pending
│  └─ sync disabled -> calendarSyncStatus = not_enabled
│
├─ infra.GoogleCalendarProvider.CreateEvent
│  ├─ calendarId = LINEBOT_GOOGLE_CALENDAR_ID
│  ├─ summary
│  ├─ startAt
│  ├─ endAt
│  ├─ timeZone
│  └─ location optional
│
├─ success
│  └─ Repository.UpdateSyncResult
│     ├─ calendarSyncStatus = calendar_synced
│     ├─ googleCalendarEventId
│     ├─ googleCalendarHtmlLink
│     └─ calendarSyncedAt
│
└─ failure
   └─ Repository.UpdateSyncResult
      ├─ calendarSyncStatus = calendar_sync_failed
      └─ calendarSyncError
```

Package addition:

```text
internal/calendar
└─ usecase.go
   └─ orchestrates repository + infra.GoogleCalendarProvider

internal/infra
└─ google_calendar_client.go
   └─ Google Calendar API implementation
```

### Future Factory / Router Flow

第一版只有 calendar，不急著硬抽 factory struct。先讓分派點停在 `task.UseCase`。

當第二個功能出現時，再把 `task.UseCase` 裡的 switch 提出成 `task/router.go` 或 `task/task_executor_factory.go`。

```text
task.UseCase future
├─ call Internal LineTaskConsult / future task consult
├─ task router resolves target module
│  ├─ calendar -> calendar.UseCase
│  ├─ note     -> note.UseCase
│  ├─ reminder -> reminder.UseCase
│  └─ others   -> future module
└─ selected module executes its own usecase/service/repository flow
```

設計原則：

```text
task owns
├─ AI extraction orchestration
├─ supported task type registry
├─ task dispatch decision
├─ unsupported task handling
└─ shared task command / result shape

feature module owns
├─ own usecase
├─ own service
├─ own repository if needed
├─ own Firestore schema if needed
└─ own future external integration if needed
```

### Module Dependency Direction

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
repository -> usecase
handler -> repository
gatekeeper -> calendar repository
```

## Block 3 - Details

### REST API First Version

```text
POST /api/tasks
├─ text required
├─ referenceTime optional
└─ timeZone optional
```

Request:

```json
{
  "text": "小傑約明天吃午餐",
  "referenceTime": "2026-04-15 14:00:00",
  "timeZone": "Asia/Taipei"
}
```

Response:

```json
{
  "success": true,
  "data": {
    "taskId": "generated-task-id",
    "operation": "create",
    "summary": "小傑約吃午餐",
    "startAt": "2026-04-16 12:00:00",
    "endAt": "2026-04-16 12:30:00",
    "location": "",
    "missingFields": ["location"],
    "calendarSyncStatus": "not_enabled",
    "googleCalendarEventId": "",
    "googleCalendarHtmlLink": ""
  }
}
```

### Internal gRPC Contract Usage

LineBot Backend 呼叫 Internal：

```text
IntegrationService.LineTaskConsult
├─ appId from config
├─ builderId from config
├─ messageText from request.text
├─ referenceTime optional override
├─ timeZone optional override
├─ supportedTaskTypes from task registry
└─ clientIp from request context
```

規則：
- REST request 不直接傳 `appId` / `builderId`。
- `appId` / `builderId` 由 LineBot Backend config 管理。
- 第一版應確認 Internal seed 允許此 app 使用 `line-memo-crud` builder。
- 第一版 `supportedTaskTypes=["calendar"]`。
- Internal response 的 `taskType` 必須是 supported task types 其中之一。

### Task Type Registry

Go 版以 string alias + const 表達 Java enum 類似語意。

```go
type TaskType string

const (
    TaskTypeCalendar TaskType = "calendar"
)
```

第一版 registry：

```text
supportedTaskTypes
└─ calendar
```

用途：
- LineBot Backend 告訴 Internal 目前可用功能。
- Internal 依自然語句判斷 `taskType`。
- `task` module 依 `taskType` 分派到 feature module。

### Calendar Create Validation

```text
required
├─ taskType
├─ operation
├─ summary
├─ startAt
└─ endAt

optional
├─ location
└─ missingFields
```

規則：
- `taskType` 不支援：不寫 Firestore，回錯。
- `summary` 空值：不寫 Firestore，回錯。
- `startAt` 空值：不寫 Firestore，回錯。
- `endAt` 空值：不寫 Firestore，回錯。
- `location` 空值：照常寫入。
- `missingFields` 包含 `location`：照常寫入。
- start/end time 缺失代表 Internal extraction incomplete，不由 LineBot Backend 自行補值。

### Firestore First Version Model

```text
calendar_tasks/{taskId}
├─ taskId
├─ source
├─ rawText
├─ taskType
├─ operation
├─ summary
├─ startAt
├─ endAt
├─ location
├─ missingFields
├─ status
├─ calendarSyncStatus
├─ googleCalendarId
├─ googleCalendarEventId
├─ googleCalendarHtmlLink
├─ calendarSyncError
├─ calendarSyncedAt
├─ internalAppId
├─ internalBuilderId
├─ internalRequest
├─ internalResponse
├─ createdAt
└─ updatedAt
```

欄位規則：
- `source=rest` for first version.
- future LINE webhook uses `source=line`.
- `taskType=calendar` for first version.
- `rawText` 必須保存，方便回查原始輸入。
- `startAt` / `endAt` 拆欄存。
- `internalResponse` 可保存完整 extraction 結果方便 debug。

### Google Calendar Config

```text
LINEBOT_GOOGLE_CALENDAR_ENABLED=true
LINEBOT_GOOGLE_CALENDAR_ID=<shared-calendar-id>
LINEBOT_GOOGLE_CALENDAR_TIME_ZONE=Asia/Taipei
LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE=<client-secret-json-path>
LINEBOT_GOOGLE_OAUTH_TOKEN_FILE=<stored-token-json-path>
```

規則：
- credentials / token 不可 commit。
- token 必須代表可寫入 shared calendar 的 Google user。
- service account 不是第一選擇；個人 / 家庭場景以 OAuth user consent 較合理。
- shared calendar id 必須可由該 OAuth user 寫入。

### Error Shape

Missing text:

```json
{
  "success": false,
  "error": {
    "code": "TEXT_REQUIRED",
    "message": "text is required"
  }
}
```

Internal extraction incomplete:

```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_EXTRACTION_INCOMPLETE",
    "message": "Internal extraction did not return required fields",
    "missingFields": ["startAt", "endAt"]
  }
}
```

Unsupported operation:

```json
{
  "success": false,
  "error": {
    "code": "OPERATION_UNSUPPORTED",
    "message": "Operation update is not supported in the first version"
  }
}
```

### Milestones

```text
M1: Empty Go module
└─ go.mod

M2: Documentation and package skeleton
├─ PLAN.md
├─ DEVELOPMENT.md
├─ internal/app
├─ internal/gatekeeper
├─ internal/task
├─ internal/calendar
├─ internal/internalclient
└─ internal/infra

M3: REST entry
├─ cmd/api main
├─ app wiring
└─ gatekeeper REST handler

M4: Internal gRPC client
├─ proto / generated client setup
└─ LineTaskConsult call

M5: Calendar create persistence
├─ task usecase dispatch
├─ calendar usecase/service/repository
└─ Firestore calendar_tasks write

M6: Tests
├─ missing text
├─ missing summary/startAt/endAt
├─ location optional
├─ unsupported operation
└─ successful Firestore create

M7: Google Calendar shared calendar sync
├─ infra.GoogleCalendarProvider interface
├─ sync status Firestore fields
├─ OAuth token file loading
├─ Google Calendar events.insert
├─ sync success / failure result update
└─ manual Postman -> Firestore -> Google Calendar verification

M8: Future LINE webhook
├─ verify LINE signature
├─ tag bot trigger
└─ same task usecase
```

### Open Questions

- Go module path 是否維持 `linebot-backend`。
- REST route 是否確定使用 `POST /api/tasks`。
- Firestore project id / emulator port 是否沿用現有 ProjectAI emulator 設定。
- Internal gRPC server address env name。
- Google shared calendar id 要使用哪一個 calendar。
- OAuth token 檔案要放在哪個非 git 路徑。
- Calendar sync 失敗時 API response 要維持 200 還是回 partial failure status。
