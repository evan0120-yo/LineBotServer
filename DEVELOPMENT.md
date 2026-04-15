# LineBot Backend Development Guide

## Purpose
這個專案是獨立的 LineBot Backend，不屬於 LinkChat，也不承擔 LinkChat 的任何業務邏輯。

核心責任：

```text
LineBot Backend
├─ 接收任務文字
├─ 呼叫 Internal AI Copilot gRPC LineTaskConsult
├─ 傳入 supported task types
├─ 驗證 Internal 回傳的任務抽取結果
├─ 依 taskType 分派到功能 module
├─ 將任務資料寫入 Firestore
└─ 未來再擴充 LINE webhook 與 Google Calendar
```

Internal AI Copilot 是 AI 溝通與自然語句解析的唯一來源。LineBot Backend 不自行建立另一套 AI pipeline。

## Project Boundary

```text
In scope
├─ REST API 測試入口
├─ 未來 LINE webhook handler
├─ Internal gRPC client
├─ Firestore task persistence
└─ Calendar task usecase orchestration

Out of scope
├─ LinkChat
├─ Internal AI prompt / Gemma 邏輯
├─ Google Calendar 第一版串接
└─ 任務文字的本地 AI 判斷
```

規則：
- 不碰 LinkChat 專案。
- 不在本專案重做 Internal 的 builder / aiclient / prompt 邏輯。
- 第一版只做 RESTful 測試入口，不接正式 LINE webhook。
- Google Calendar 後續再做，第一版只寫 Firestore。

## Architecture

本專案採三層加 UseCase 層：

```text
Handler
└─ UseCase
   └─ Service
      └─ Repository
```

### Layer Responsibility

```text
Layer 1: Handler
├─ REST handler
├─ future LINE webhook handler
├─ request parse
├─ response mapping
└─ 不做業務流程與 Firestore 存取

Layer 2: UseCase
├─ 單一業務案例的流程編排
├─ 呼叫 Internal gRPC client
├─ 呼叫 service 做 validation / normalize
├─ 呼叫 repository 寫入 Firestore
└─ 不處理 HTTP / LINE transport 細節

Layer 3: Service
├─ deterministic business rules
├─ 驗證 Internal extraction 是否完整
├─ 判斷哪些欄位可缺、哪些欄位不可缺
└─ 不直接呼叫 Firestore 或 transport

Layer 4: Repository
├─ Firestore read / write / query
├─ persistence mapping
└─ 不做業務判斷
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

禁止方向：
- Handler 不直接呼叫 Repository。
- Repository 不依賴 UseCase / Service。
- Service 不依賴 HTTP / LINE / gRPC transport model。
- future LineBot handler 與 REST handler 不各自複製任務流程，兩者必須共用 task UseCase。
- calendar 不反向依賴 task。
- internalclient 不理解 calendar 業務。

## Package Baseline

第一版 package 採 Internal Go 的 module-first 風格，不使用 `internal/usecase`、`internal/service`、`internal/repository` 這種技術層大 package。

```text
Backend/
├─ cmd/api/
│  └─ doc.go
├─ internal/
│  ├─ app/
│  │  └─ doc.go
│  ├─ gatekeeper/
│  │  └─ doc.go
│  ├─ task/
│  │  └─ doc.go
│  ├─ calendar/
│  │  └─ doc.go
│  ├─ internalclient/
│  │  └─ doc.go
│  └─ infra/
│     └─ doc.go
└─ go.mod
```

補充：
- `gatekeeper` 是 REST / future LINE webhook 的入口邊界。
- `task` 是 AI task orchestration、supported task type registry 與分派位置。
- `calendar` 是目前第一個功能 module，擁有自己的 usecase / service / repository。
- `internalclient` 只負責 Internal AI Copilot gRPC integration。
- 未來新增功能時，新增同層 module，再由 `task` 分派。

## First Version Flow

```text
Postman
└─ POST /api/tasks
   ├─ text
   ├─ referenceTime optional
   └─ timeZone optional
      │
      ▼
gatekeeper.Handler
└─ parse request
   │
   ▼
gatekeeper.UseCase
└─ task.UseCase
   ├─ 呼叫 internalclient.Service.LineTaskConsult
   ├─ 傳入 supportedTaskTypes=["calendar"]
   ├─ 檢查 taskType 是否支援
   ├─ 檢查 operation 是否支援
   └─ calendar.UseCase.Create
      ├─ calendar.Service.ValidateCreate
      │  ├─ summary required
      │  ├─ startAt required
      │  ├─ endAt required
      │  └─ location optional
      └─ calendar.Repository.Create
         └─ Firestore calendar_tasks
```

## Internal gRPC Rule

LineBot Backend 呼叫 Internal 的正式路徑是 gRPC：

```text
Internal IntegrationService.LineTaskConsult
├─ appId
├─ builderId
├─ messageText
├─ referenceTime
├─ timeZone
├─ supportedTaskTypes
└─ clientIp
```

規則：
- `appId` 與 `builderId` 由 LineBot Backend config 固定管理。
- 第一版 REST request 不要求使用者傳 `appId` / `builderId`。
- `messageText` 使用 REST body 的 `text`。
- `referenceTime` / `timeZone` 可由 request 覆蓋；未提供時由 LineBot Backend 或 Internal 補值。
- `supportedTaskTypes` 由 LineBot Backend task registry 產生；第一版固定為 `["calendar"]`。
- Internal response 必須回 `taskType`，供 task module 分派。
- Internal 回傳是任務抽取結果，本專案只做保存與後續外部整合。

## Task Type Registry Rule

Go 中以 string alias + const 表達 Java enum 類似語意。

```go
type TaskType string

const (
	TaskTypeCalendar TaskType = "calendar"
)
```

規則：
- registry 第一版只有 `calendar`。
- LineBot Backend 傳 `supportedTaskTypes` 給 Internal。
- Internal 從 `supportedTaskTypes` 中選出 `taskType` 回傳。
- `task` module 依 `taskType` 決定要呼叫哪個 feature module。

## Task Extraction Validation Rule

Internal 已負責讓 Gemma 將自然語句解析成結構化時間資料。本專案不自行補時間。

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
- `startAt` 空值：不寫 Firestore，回錯，視為 Internal extraction incomplete。
- `endAt` 空值：不寫 Firestore，回錯，視為 Internal extraction incomplete。
- `location` 空值：照常寫入，不回錯。
- `missingFields` 包含 `location` 不影響 create。

## Firestore Rule

第一版只做新增。

建議 collection：

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
├─ internalAppId
├─ internalBuilderId
├─ internalRequest
├─ internalResponse
├─ createdAt
└─ updatedAt
```

規則：
- `startAt` / `endAt` 應拆欄存，不存單一區間字串。
- `rawText` 必須保存，方便追蹤 Internal extraction 問題。
- `internalResponse` 可保存第一版完整回應，方便 debug。
- 後續 update / delete / query 需要時，再補查詢欄位與索引設計。

## Future LINE Webhook Rule

未來接 LINE 時使用 tag bot 作為觸發門檻，不使用固定前綴。

```text
LINE message event
├─ 沒有 tag bot
│  └─ ignore
└─ 有 tag bot
   ├─ 移除 mention
   ├─ 取得 message text
   └─ 呼叫同一個 Task UseCase
```

規則：
- tag bot 只負責聊天室噪音閘門，不是業務指令。
- 是否為任務內容交給 Internal + Gemma 判斷。
- LINE webhook handler 不複製 REST handler 的業務流程。

## Future Google Calendar Rule

Google Calendar 不列入第一版。

後續串接時：
- Calendar integration 應放在 UseCase 編排下。
- Firestore 必須保存 Google Calendar event id。
- create / update / delete 應以 Firestore 作為任務對照表。
- 授權方式需另行確認，不能假設 server 可直接寫個人 Pixel 上的日曆。

## Testing Strategy

測試順序：

```text
1. UseCase tests
   └─ REST 與 LINE 都共用同一條任務流程

2. Service tests
   └─ extraction required fields validation

3. Repository tests
   └─ Firestore document mapping

4. Handler tests
   └─ request parse / response envelope / error mapping
```

第一版至少要覆蓋：
- REST request 成功呼叫 usecase。
- Internal 回 summary / startAt / endAt 時會寫 Firestore。
- Internal 缺 startAt / endAt 時不寫 Firestore 並回錯。
- location 缺失仍可成功新增。

## Development Checklist

每次新增功能前確認：

1. 是否仍和 LinkChat 無關。
2. 是否仍由 Internal 負責 AI parsing。
3. Handler 是否只做 transport mapping。
4. REST 與 future LINE 是否共用同一個 UseCase。
5. startAt / endAt 缺失是否被視為錯誤。
6. location 是否維持 optional。
7. 是否需要同步 PLAN.md。
