# LineBot Backend Code Review

---

# BLOCK 1: AI 對產品的想像

這個 Go backend 是一個「自然語句任務轉換器」。

這份文件是 current implementation walkthrough，不是驗收規格，也不是未來設計稿。

真正的產品核心不是做 AI，而是把日常對話轉成可執行任務。它不處理 AI 的 prompt 組裝、不管 Gemma 怎麼選，那些都交給 Internal AI Copilot。LineBot Backend 專注在：任務入口、任務分派、任務保存。

它的主要使用者有三種：
- 內部測試者：用 Postman 打 REST API，快速驗證整條鏈路。
- LINE 使用者：tag bot 後送出自然語句，系統自動建立任務。
- 開發者：維護 task type registry，新增功能時加新的 feature module。

它目前比較像「任務分派中心」，不是完整的任務管理系統。第一版只做 create，不做 update/delete/query。第一版只支援 calendar，未來會有 note、reminder 等。任務會先寫 Firestore；若啟用 Google Calendar 設定，會再同步建立 shared calendar event。

從 code 看得出的刻意選擇有幾個：
- REST 和 LINE webhook 共用同一條 task usecase，不分兩套業務流程。
- 任務類型用 TaskType registry 管理，LineBot Backend 告訴 Internal 支援哪些，Internal AI 從中選一個回傳。
- Feature module 自包含 usecase/service/repository，calendar 不依賴 task，未來加 note 也不會影響 calendar。
- Internal 的 request/response 序列化成 JSON 保存在 Firestore，方便追蹤 extraction 問題。
- 時間欄位（startAt/endAt）不做本地轉換，直接保存 Internal 回傳值，避免時區錯誤。
- location 視為 optional，因為很多事件沒有明確地點。

它目前不是：
- 不是 AI 引擎，AI extraction 完全交給 Internal AI Copilot。
- 不是完整的任務管理系統，沒有 update/delete/query/list。
- 不是完整 Google Calendar proxy，目前只支援 create -> events.insert，且 Firestore 仍是 source of truth。
- 不是多用戶系統，沒有 user authentication。

---

# BLOCK 2: 讀者模式

## A. 系統啟動後，骨架怎麼接起來

這個服務啟動後只開 HTTP server，至少有一條固定 REST route，並在 LINE config 齊全時再註冊 webhook route。

```text
main 啟動
   │
   ├─ 讀 env config
   │  ├─ LINEBOT_ADDR=:8083
   │  ├─ LINEBOT_FIRESTORE_PROJECT_ID
   │  ├─ LINEBOT_FIRESTORE_EMULATOR_HOST
   │  ├─ LINEBOT_INTERNAL_GRPC_ADDR
   │  ├─ LINEBOT_INTERNAL_APP_ID
   │  └─ LINEBOT_INTERNAL_BUILDER_ID
   │
   ├─ 建 Firestore store
   │  └─ 支援 emulator（透過 FIRESTORE_EMULATOR_HOST env）
   │
   ├─ 建 Internal gRPC client
   │  └─ 連到 Internal AI Copilot backend
   │
   ├─ 若 Google Calendar enabled
   │  └─ 建 Google Calendar client
   │
   ├─ 組 calendar module
   │  ├─ service (validation)
   │  ├─ repository (Firestore persistence)
   │  └─ usecase (orchestration)
   │
   ├─ 組 task module
   │  ├─ service (dispatch validation)
   │  └─ usecase (AI extraction + dispatch)
   │
   ├─ 組 gatekeeper module
   │  ├─ usecase (request mapping)
   │  └─ handler (HTTP boundary)
   │
   ├─ 註冊 HTTP route
   │  ├─ POST /api/tasks -> gatekeeper.Handler.CreateTask
   │  └─ POST /api/line/webhook -> gatekeeper.LineHandler.ServeHTTP
   │     └─ only when LINE channel secret + bot user id are configured
   │
   └─ 開 HTTP server
```

> 注意：第一版沒有 CORS 設定，預設不允許跨域請求。

> 注意：Firestore emulator 透過環境變數 `FIRESTORE_EMULATOR_HOST` 啟用，`infra.NewStoreWithOptions()` 會在建立 Firestore client 前設定。

## B. POST /api/tasks 完整流程

這是第一版最直接的測試入口，負責把自然語句轉成 calendar task。

```text
POST /api/tasks
{
  "text": "小傑約明天吃午餐",
  "referenceTime": "2026-04-15 14:00:00",   // optional
  "timeZone": "Asia/Taipei"                  // optional
}
   │
   ▼
gatekeeper.Handler.CreateTask
   ├─ DecodeJSONStrict (max 1MB)
   ├─ Validate text required
   │  └─ 空值或只有空白 -> TEXT_REQUIRED
   ├─ Resolve clientIP
   │  ├─ X-Forwarded-For (取第一個)
   │  ├─ X-Real-IP
   │  └─ RemoteAddr (fallback)
   └─ Build CreateTaskCommand
      │
      ▼
gatekeeper.UseCase.CreateTask
   └─ Map to task.CreateFromTextCommand
      ├─ source = "rest"
      ├─ text / referenceTime / timeZone / clientIP
      │
      ▼
task.UseCase.CreateFromText
   ├─ 組 Internal LineTaskConsult command
   │  ├─ appId from config (linebot-app)
   │  ├─ builderId from config (4)
   │  ├─ messageText from text
   │  ├─ referenceTime / timeZone from request
   │  ├─ supportedTaskTypes = ["calendar"]
   │  └─ clientIP
   │
   ├─ internalClient.LineTaskConsult
   │  │
   │  ▼
   │  Internal AI Copilot gRPC
   │  └─ Gemma extraction
   │     ├─ taskType: "calendar"
   │     ├─ operation: "create"
   │     ├─ summary: "小傑約吃午餐"
   │     ├─ startAt: "2026-04-16 12:00:00"
   │     ├─ endAt: "2026-04-16 12:30:00"
   │     ├─ location: ""
   │     └─ missingFields: ["location"]
   │
   ├─ task.Service.ValidateTaskType
   │  └─ "calendar" in ["calendar"] -> ✓
   │
   ├─ task.Service.ValidateOperation
   │  └─ "create" == "create" -> ✓
   │
   └─ Dispatch by taskType
      │
      ▼ (calendar)
calendar.UseCase.Create
   ├─ calendar.Service.ValidateCreate
   │  ├─ summary required -> ✓
   │  ├─ startAt required -> ✓
   │  ├─ endAt required -> ✓
   │  └─ location optional (不驗證)
   │
   └─ calendar.Repository.Create
      ├─ 生成 UUID
      ├─ 組 CalendarTaskDoc
      │  ├─ taskId = uuid
      │  ├─ source = "rest"
      │  ├─ rawText = "小傑約明天吃午餐"
      │  ├─ taskType = "calendar"
      │  ├─ operation = "create"
      │  ├─ summary / startAt / endAt / location
      │  ├─ missingFields = ["location"]
      │  ├─ status = "created"
      │  ├─ internalAppId / internalBuilderId
      │  ├─ internalRequest (序列化 JSON)
      │  ├─ internalResponse (序列化 JSON)
      │  ├─ createdAt / updatedAt
      │  │
      │  ▼
      │  Firestore
      │  calendar_tasks/{uuid}
      │  └─ Write document
      │
      ├─ sync disabled -> Return CalendarTask(calendarSyncStatus=not_enabled)
      │
      └─ sync enabled
         ├─ infra.GoogleCalendarClient.CreateEvent
         ├─ success -> UpdateSyncResult(calendar_synced + event metadata)
         └─ failure -> UpdateSyncResult(calendar_sync_failed + error)
         │
         ▼
task.UseCase (繼續)
   └─ Map to TaskResult
      │
      ▼
gatekeeper.Handler (繼續)
   └─ Map to CreateTaskResponseData
      └─ WriteJSON 200
         {
           "success": true,
           "data": {
             "taskId": "uuid",
             "operation": "create",
             "summary": "小傑約吃午餐",
             "startAt": "2026-04-16 12:00:00",
             "endAt": "2026-04-16 12:30:00",
             "location": "",
             "missingFields": ["location"],
             "calendarSyncStatus": "calendar_synced | calendar_sync_failed | not_enabled",
             "googleCalendarId": "",
             "googleCalendarEventId": "",
             "googleCalendarHtmlLink": "",
             "calendarSyncError": ""
            }
          }
```

> 注意：referenceTime 和 timeZone 可由 request 覆蓋；若未提供，Internal backend 會補系統時間/時區。

> 注意：Internal request/response 序列化成 JSON 字串保存，方便日後 debug extraction 問題。

## C. TaskType Registry 與 Dispatch

第一版只有 calendar，但架構設計上支援未來擴充。

```text
task.SupportedTaskTypes()
└─ []string{"calendar"}

Internal AI 從中選出 taskType
└─ taskType = "calendar"

task.UseCase dispatch
├─ taskType == "calendar"
│  └─ calendarUseCase.Create
│
└─ 未來新增 note
   ├─ TaskTypeNote = "note"
   ├─ SupportedTaskTypes() 加入 "note"
   └─ dispatch 加 case
      └─ taskType == "note" -> noteUseCase.Create
```

規則：
- LineBot Backend 管理 supportedTaskTypes，告訴 Internal 可用的任務類型。
- Internal AI 依自然語句判斷 taskType，必須是 supported list 其中之一。
- task.UseCase 依 taskType 分派到對應的 feature module。
- 未來新增功能時，新增 feature module + 更新 registry + 更新 dispatch。

## D. Validation 與錯誤處理

系統有多層 validation，每層負責不同職責。

```text
Validation 層級
│
├─ gatekeeper.Handler
│  └─ text required
│     └─ TEXT_REQUIRED (400)
│
├─ task.Service
│  ├─ ValidateTaskType
│  │  └─ TASK_TYPE_UNSUPPORTED (400)
│  └─ ValidateOperation
│     └─ OPERATION_UNSUPPORTED (400)
│
├─ calendar.Service
│  └─ ValidateCreate
│     ├─ summary required
│     ├─ startAt required
│     ├─ endAt required
│     └─ INTERNAL_EXTRACTION_INCOMPLETE (400)
│
├─ internalclient.Service
│  └─ gRPC call failed
│     └─ INTERNAL_GRPC_ERROR (500)
│
└─ infra.Store
   └─ Firestore write failed
      └─ FIRESTORE_WRITE_ERROR (500)
```

錯誤碼一覽：

| 錯誤碼 | HTTP Status | 觸發條件 |
|--------|-------------|----------|
| TEXT_REQUIRED | 400 | request.text 空值或只有空白 |
| INTERNAL_EXTRACTION_INCOMPLETE | 400 | summary/startAt/endAt 缺失 |
| TASK_TYPE_UNSUPPORTED | 400 | taskType 不在 supported list |
| OPERATION_UNSUPPORTED | 400 | operation 不是 "create" |
| INTERNAL_GRPC_ERROR | 500 | Internal gRPC 呼叫失敗 |
| FIRESTORE_WRITE_ERROR | 500 | Firestore 寫入失敗 |

錯誤處理流程：

```text
業務層產生 BusinessError
├─ Code
├─ Message
├─ HTTPStatus
└─ MissingFields optional
   │
   ▼
傳播到 gatekeeper.Handler
   │
   ▼
infra.WriteError()
├─ AsBusinessError()
│  ├─ 是 BusinessError
│  │  └─ 使用 bizErr.Code / Message / HTTPStatus
│  └─ 不是 BusinessError
│     └─ 回 INTERNAL_ERROR / 500
└─ WriteJSON error response
   {
     "success": false,
     "error": {
       "code": "ERROR_CODE",
       "message": "error message",
       "missingFields": ["startAt", "endAt"]
     }
   }
```

> 注意：location 缺失不視為錯誤，這是刻意設計。

> 注意：第一版 operation 限制為 "create"，未來要放寬時在 task.Service.ValidateOperation 修改。

## E. Firestore Schema

calendar_tasks collection 儲存所有 calendar 任務。

```text
calendar_tasks/{taskId}
├─ taskId              string      UUID
├─ source              string      "rest" / future "line"
├─ rawText             string      原始輸入文字
├─ taskType            string      "calendar"
├─ operation           string      "create"
├─ summary             string      事件標題
├─ startAt             string      開始時間（Internal 回傳格式）
├─ endAt               string      結束時間（Internal 回傳格式）
├─ location            string      地點（可空）
├─ missingFields       []string    Internal extraction 缺失欄位
├─ status              string      "created"
├─ calendarSyncStatus  string      "not_enabled" / "calendar_sync_pending" / "calendar_synced" / "calendar_sync_failed"
├─ googleCalendarId    string      shared calendar id
├─ googleCalendarEventId string    Google Calendar event id
├─ googleCalendarHtmlLink string   event link
├─ calendarSyncError   string      sync failure reason
├─ calendarSyncedAt    timestamp   sync timestamp
├─ internalAppId       string      "linebot-app"
├─ internalBuilderId   int         4
├─ internalRequest     string      LineTaskConsultRequest JSON
├─ internalResponse    string      LineTaskConsultResponse JSON
├─ createdAt           timestamp
└─ updatedAt           timestamp
```

規則：
- taskId 使用 github.com/google/uuid 生成。
- source 由 gatekeeper 設定："rest" 或 "line"（依據請求來源）。
- rawText 必須保存，方便追蹤 Internal extraction 問題。
- startAt/endAt 不做格式轉換，直接保存 Internal 回傳值。
- location 可空，missingFields 可包含 "location"。
- status 第一版固定 "created"。
- Google Calendar sync 使用 `calendarSyncStatus` 表達 `not_enabled` / `calendar_sync_pending` / `calendar_synced` / `calendar_sync_failed`。
- internalRequest/internalResponse 序列化成 JSON 保存，方便 debug。

> 注意：目前沒有索引設定，未來若需要 query API，需要針對 source / taskType / createdAt 建立索引。

## F. 未來擴充方向

系統設計上已預留擴充空間。

### F-1. LINE Webhook 串接（已實作）

```text
gatekeeper/line_handler.go
├─ 驗證 LINE signature (HMAC-SHA256)
│  └─ x-line-signature vs computed HMAC
├─ 解析 webhook JSON events
├─ 過濾 message events (type="message", message.type="text")
├─ 檢查 bot mention
│  └─ compare mentionee.UserID with botUserID
├─ 移除 mention 文字 (using index + length)
└─ 共用 gatekeeper.UseCase.CreateTask
   └─ source = "line"

HTTP router
└─ POST /api/line/webhook -> lineHandler.ServeHTTP
   └─ 只在 LineChannelSecret 和 LineBotUserID 都設定時註冊（fail-closed）
```

實作要點：
- mention 檢測使用 webhook metadata，確保只有 mention 到這個 bot 才觸發。
- mention 移除使用 `index` 和 `length` 欄位，準確移除（支援多位元組字元）。
- 處理所有符合條件的 events，request-level response 固定回 200 ack + counts。
- 共用 task usecase，不複製 calendar persistence 流程。

### F-2. 新增 Task Type (note)

```text
新增 internal/note/
├─ model.go
├─ service.go
├─ repository.go
└─ usecase.go

修改 task/model.go
├─ TaskTypeNote = "note"
└─ SupportedTaskTypes() 加入 "note"

修改 task/usecase.go
└─ switch taskType
   ├─ calendar
   └─ note -> noteUseCase.Create

修改 app/app.go
└─ 建立 note module + 注入 taskUseCase

Firestore 新增
└─ notes/{noteId}
```

規則：
- 每個 feature module 自包含完整邏輯。
- note 不依賴 calendar，calendar 也不知道 note 存在。
- app.go 負責組裝與注入。

### F-3. 支援 Update/Delete/Query

```text
修改 task/service.go
└─ ValidateOperation() 放寬
   ├─ "create"
   ├─ "update"
   ├─ "delete"
   └─ "query"

修改 calendar/usecase.go
├─ Update(ctx, UpdateCommand)
├─ Delete(ctx, DeleteCommand)
└─ Query(ctx, QueryCommand)

新增 REST routes
├─ PUT /api/tasks/{taskId}
├─ DELETE /api/tasks/{taskId}
└─ GET /api/tasks?...
```

規則：
- update/delete 需要 taskId 驗證（檢查是否存在）。
- query 需要 Firestore 索引設計。
- Internal extraction 也需要支援 update/delete/query 的語意解析。

### F-4. Google Calendar 串接

```text
方案 C: shared Google Calendar
├─ 建立一個你與伴侶共用的 Google Calendar
├─ 透過 OAuth user consent 取得可寫入該 calendar 的 refresh token
├─ LineBot Backend 寫入 configured shared calendar id
└─ Pixel / Google Calendar app 透過 Google 帳號同步事件

calendar.UseCase 透過 infra.GoogleCalendarProvider interface 依賴外部 calendar 寫入能力。

已新增 infra/google_calendar_client.go
├─ OAuth token load
├─ Calendar service client
└─ events.insert

calendar.UseCase.Create
├─ 先寫 Firestore
├─ sync enabled -> 呼叫 CalendarProvider.CreateEvent
├─ 成功 -> calendarSyncStatus = calendar_synced
└─ 失敗 -> calendarSyncStatus = calendar_sync_failed，保留 Firestore task

CalendarTaskDoc
├─ calendarSyncStatus
├─ googleCalendarId
├─ googleCalendarEventId
├─ googleCalendarHtmlLink
├─ calendarSyncError
└─ calendarSyncedAt
```

規則：
- 目前只做 create -> Google Calendar events.insert。
- Firestore 仍是 task source of truth。
- Google Calendar sync 失敗不應刪除已建立 task。
- credentials / token 不可 commit。
- service account 不是目前主方案；個人 / 家庭場景以 OAuth user + shared calendar 較合理。

---

# BLOCK 3: 技術補充

## A. Package 組織與依賴

系統採用 module-first package 組織，避免技術層大 package。

```text
Backend/
├─ cmd/api                    進入點
│  └─ main.go
│
└─ internal/
   ├─ app                     應用程式組裝
   │  └─ app.go               wiring all modules
   │
   ├─ gatekeeper              REST / LINE webhook boundary
   │  ├─ handler.go
   │  ├─ usecase.go
   │  └─ model.go
   │
   ├─ task                    AI task orchestration + dispatch
   │  ├─ usecase.go
   │  ├─ service.go
   │  └─ model.go
   │
   ├─ calendar                Calendar feature module
   │  ├─ usecase.go
   │  ├─ service.go
   │  ├─ repository.go
   │  └─ model.go
   │
   ├─ internalclient          Internal AI Copilot gRPC client
   │  ├─ service.go
   │  ├─ model.go
   │  └─ pb/                  generated protobuf
   │
   └─ infra                   Shared infrastructure
      ├─ config.go
      ├─ errors.go
      ├─ google_calendar_client.go
      ├─ http.go
      ├─ store.go
      └─ model.go
```

依賴方向：

```text
允許
├─ gatekeeper -> task
├─ task -> internalclient
├─ task -> calendar
├─ calendar -> infra
├─ internalclient -> infra
└─ app -> all (wiring only)

禁止
├─ calendar -> task
├─ internalclient -> calendar
├─ repository -> usecase
└─ handler -> repository
```

設計原則：
- 三層 + UseCase 架構：Handler → UseCase → Service → Repository
- Feature module 自包含：calendar 擁有自己的 usecase/service/repository
- 共享基礎設施放 infra：config / errors / http / store
- 依賴方向由外到內：gatekeeper → task → feature → infra

## B. 環境變數與配置

所有配置透過環境變數載入，提供合理預設值。

```bash
# HTTP server
LINEBOT_ADDR=:8083

# Firestore
LINEBOT_FIRESTORE_PROJECT_ID=dailo-467502
LINEBOT_FIRESTORE_EMULATOR_HOST=localhost:8090

# Internal gRPC
LINEBOT_INTERNAL_GRPC_ADDR=localhost:9091
LINEBOT_INTERNAL_APP_ID=linebot-app
LINEBOT_INTERNAL_BUILDER_ID=4

# Google Calendar
LINEBOT_GOOGLE_CALENDAR_ENABLED=true
LINEBOT_GOOGLE_CALENDAR_ID=<shared-calendar-id>
LINEBOT_GOOGLE_CALENDAR_TIME_ZONE=Asia/Taipei
LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE=<client-secret-json-path>
LINEBOT_GOOGLE_OAUTH_TOKEN_FILE=<stored-token-json-path>

# Timeout
LINEBOT_SERVER_READ_TIMEOUT=10s
LINEBOT_SERVER_WRITE_TIMEOUT=5m
```

infra/config.go 負責：
- 環境變數讀取
- 預設值提供
- 型別轉換（int / duration）

> 注意：Firestore emulator 透過 FIRESTORE_EMULATOR_HOST 啟用，`infra.NewStoreWithOptions()` 會在建立 Firestore client 前設定此環境變數。

## C. Proto 與 gRPC Client

internalclient 封裝 Internal AI Copilot 的 gRPC client。

```text
Proto 處理
├─ 拷貝 Internal proto
│  └─ api/grpc/internal_ai.proto
│
├─ 修改 go_package
│  └─ option go_package = "linebot-backend/internal/internalclient/pb;grpcpb"
│
├─ 生成 Go code
│  ├─ protoc --go_out --go-grpc_out
│  ├─ internal_ai.pb.go
│  └─ internal_ai_grpc.pb.go
│
└─ internalclient.Service wrapper
   ├─ 隱藏 protobuf 細節
   ├─ 提供 domain model
   └─ 錯誤轉換
```

internalclient.Service 職責：
- 建立 gRPC connection（insecure credentials for local dev）
- 組裝 LineTaskConsultRequest
- 呼叫 IntegrationServiceClient.LineTaskConsult
- 解析 LineTaskConsultResponse
- Map 成 domain-friendly LineTaskConsultResult
- gRPC 錯誤轉成 infra.NewInternalGRPCError

> 注意：不做 calendar 業務判斷，只做 transport mapping。

## D. Firestore Client 封裝

infra.Store 封裝 Firestore client，提供 type-safe 操作。

```text
Store 職責
├─ NewStoreWithOptions()
│  ├─ 建立 firestore.Client
│  ├─ 支援 emulator（建立 client 前設定 FIRESTORE_EMULATOR_HOST）
│  └─ 回傳 Store instance
│
├─ CreateCalendarTask()
│  ├─ collection("calendar_tasks")
│  ├─ doc(taskId)
│  └─ Set(CalendarTaskDoc)
│
├─ GetCalendarTask()
│  ├─ doc(taskId).Get()
│  └─ DataTo(&CalendarTaskDoc)
│
└─ Close()
   └─ client.Close()
```

規則：
- Store 只提供 type-safe 操作，不做業務判斷。
- 錯誤轉成 infra.NewFirestoreWriteError。
- 未來新增功能時，新增對應的 Create/Get/Update/Delete 方法。

## E. HTTP JSON Envelope

所有 HTTP API 使用統一的 JSON envelope。

```go
type APIResponse struct {
    Success bool      `json:"success"`
    Data    any       `json:"data,omitempty"`
    Error   *APIError `json:"error,omitempty"`
}
```

成功回應：

```json
{
  "success": true,
  "data": { ... }
}
```

錯誤回應：

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "error message"
  }
}
```

infra/http.go 提供：
- WriteJSON(w, status, data) - 寫成功回應
- WriteError(w, err) - 寫錯誤回應
- DecodeJSONStrict(w, r, target, maxBytes) - 嚴格解析 JSON

> 注意：DecodeJSONStrict 使用 DisallowUnknownFields，防止多餘欄位。

## F. 設計決策

### F-1. Module-First 而非技術層 Package

決策：採用 module-first（app/gatekeeper/task/calendar）而非技術層（usecase/service/repository）。

原因：
- 避免大 package，每個 feature module 自包含完整邏輯。
- 擴充新功能時只需新增 module，不影響其他 module。
- 參考 Internal AI Copilot Go Backend 的成功經驗。

### F-2. TaskType Registry 由 LineBot 管理

決策：LineBot Backend 管理 supportedTaskTypes，傳給 Internal 讓 AI 選擇。

原因：
- LineBot Backend 決定目前可執行哪些功能。
- Internal AI 從 supported list 中選出 taskType。
- 職責清晰：LineBot 管可用性，Internal 管判斷。
- 未來擴充時，更新 registry 即可。

### F-3. Location Optional

決策：location 缺失不視為錯誤。

原因：
- 真實場景中很多事件沒有明確地點。
- Internal extraction 可能無法從自然語句提取 location。
- 符合 BDD 場景：「location is missing but task is still created」。
- summary/startAt/endAt 才是核心必填。

### F-4. 保存完整 Internal Request/Response

決策：Firestore 保存 Internal 的 request/response JSON。

原因：
- 方便 debug extraction 問題。
- 可追蹤 Internal 回傳了什麼。
- 未來可用於 ML training data。
- JSON 序列化成本低。

### F-5. 時間欄位不做本地轉換

決策：startAt/endAt 直接保存 Internal 回傳值，不做格式轉換。

原因：
- 避免時區轉換錯誤。
- Internal 已處理相對時間語意（「明天」、「下週」）。
- Google Calendar client 會把 Internal 回傳的 `yyyy-MM-dd HH:mm:ss` 依 configured timezone 轉成 RFC3339 後送出。
- 未來串接時再做統一格式化。

### F-6. 第一版不支援 Update/Delete/Query

決策：第一版只實作 create operation。

原因：
- 快速驗證整條鏈路（extraction → validation → persistence）。
- update/delete/query 需要額外的 query API 與狀態管理。
- create 已涵蓋核心流程，足以驗證架構設計。
- 未來擴充時在 task.Service 放寬 operation validation。

## G. 測試策略

第一版專注架構驗證，測試策略分三層。

### G-1. Unit Tests

```text
calendar.Service.ValidateCreate
├─ summary required
├─ startAt required
├─ endAt required
└─ location optional

task.Service.ValidateTaskType
├─ calendar supported
└─ unknown unsupported

task.Service.ValidateOperation
├─ create supported
└─ update/delete/query unsupported
```

### G-2. Integration Tests（需 emulator）

```text
internalclient.Service.LineTaskConsult
└─ require Internal gRPC running

calendar.Repository.Create
└─ require Firestore emulator

gatekeeper.Handler.CreateTask
├─ text missing
├─ text empty
├─ successful create
└─ location optional
```

### G-3. End-to-End Tests

```text
app integration test
├─ POST /api/tasks
├─ Internal extraction success
├─ Firestore persistence
└─ response mapping
```

> 注意：第一版重點是架構驗證，測試覆蓋率不是目標。

## H. 已知限制與未來改進

### H-1. 第一版限制

```text
限制
├─ 只支援 calendar create
├─ 不支援 update/delete/query
├─ Google Calendar sync 只支援 create（方案 C）
├─ 沒有 user authentication
├─ 沒有 CORS 設定
└─ 沒有索引設計
```

### H-2. 未來改進方向

```text
改進
├─ 新增 task types（note / reminder）
├─ 支援 update/delete/query
├─ 擴充 Google Calendar update/delete/query
├─ 新增 Firestore 索引
└─ 單元測試覆蓋（包含 LINE webhook 測試）
```

---

## 總結

### 第一版完成項目

```text
完成
├─ REST API POST /api/tasks
├─ Internal gRPC LineTaskConsult integration
├─ TaskType registry (calendar)
├─ Operation validation (create only)
├─ Calendar validation (summary/startAt/endAt required)
├─ Firestore calendar_tasks persistence
├─ Optional Google Calendar shared calendar events.insert sync
├─ Google OAuth token generator cmd/googleauth
├─ Calendar usecase sync unit tests
├─ Error handling (6 error codes)
└─ Complete documentation
   ├─ PLAN.md
   ├─ SDD.md
   ├─ BDD.md
   ├─ DEVELOPMENT.md
   └─ CODE_REVIEW.md
```

### 架構優勢

```text
優勢
├─ 職責清晰
│  ├─ gatekeeper: boundary
│  ├─ task: orchestration
│  ├─ calendar: feature
│  ├─ internalclient: external integration
│  └─ infra: shared infrastructure
│
├─ 易於擴充
│  ├─ 新增 task type: 新增 feature module
│  ├─ 新增 operation: 放寬 service validation
│  └─ 新增入口: 新增 handler，共用 usecase
│
└─ 易於測試
   ├─ Unit test: service validation
   ├─ Integration test: repository + client
   └─ E2E test: full flow
```

### 啟動步驟

```bash
# 1. 啟動 Firestore emulator
firebase emulators:start --only firestore --project dailo-467502

# 2. 啟動 Internal AI Copilot backend
cd InternalAICopliot/Backend/Go
go run cmd/api/main.go

# 3. 啟動 LineBot Backend
cd LineBot/Backend
go run cmd/api/main.go

# 4. 測試 API
curl -X POST http://localhost:8083/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"text": "小傑約明天吃午餐"}'
```

---

**文件版本**：v1.0
**最後更新**：2026-04-15
**作者**：Claude Opus 4.6
