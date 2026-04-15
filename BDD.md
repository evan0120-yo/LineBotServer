# LineBot Backend BDD

## Behavior Scope

LineBot Backend 第一版只處理 REST 測試入口。它把自然語句交給 Internal AI Copilot 做 structured extraction，再依 extraction result 寫入 Firestore。

```text
actor
├─ local tester
│  └─ 使用 Postman 呼叫 REST API
└─ future LINE user
   └─ tag bot 後由 LINE webhook 轉入同一條 task usecase

system boundary
├─ LineBot Backend
│  ├─ request boundary
│  ├─ task dispatch
│  ├─ Firestore persistence
│  └─ optional Google Calendar sync
└─ Internal AI Copilot
   └─ natural language -> structured task extraction
```

## Scenario Group: REST Calendar Create

### Scenario: create calendar task from natural language

Given local tester 呼叫 `POST /api/tasks`  
And request body 帶入 `text="小傑約明天吃午餐"`  
And LineBot Backend config 已設定 Internal `appId`、`builderId`、`supportedTaskTypes=["calendar"]`  
When LineBot Backend 呼叫 Internal `LineTaskConsult`  
And Internal 回傳 `taskType="calendar"`、`operation="create"`、`summary`、`startAt`、`endAt`  
Then LineBot Backend 應建立 `calendar_tasks/{taskId}`  
And response 應回傳 `taskId` 與 extraction result

```text
expected flow
POST /api/tasks
└─ gatekeeper
   └─ task
      ├─ internalclient.LineTaskConsult
      └─ calendar.Create
         └─ Firestore calendar_tasks
```

### Scenario: location is missing but task is still created

Given Internal 回傳 `taskType="calendar"`  
And `operation="create"`  
And `summary`、`startAt`、`endAt` 皆存在  
And `location=""`  
And `missingFields=["location"]`  
When LineBot Backend 處理此 extraction result  
Then LineBot Backend 應照常寫入 Firestore  
And 不應因 `location` 缺失回錯

## Scenario Group: Google Calendar Sync

第一個 Google Calendar 串接版本採方案 C：使用一個共用 Google Calendar，LineBot Backend 透過已完成 OAuth 授權的 Google 帳號寫入該共用行事曆。Firestore 仍是 LineBot Backend 的 task source of truth。

```text
Google Calendar sync boundary
├─ Firestore calendar_tasks 永遠先保存 task
├─ Google Calendar 是外部同步目標
├─ OAuth token 代表可寫入 shared calendar 的 Google user
└─ shared calendar 由使用者與伴侶共同訂閱 / 共用
```

### Scenario: create task and sync to shared Google Calendar

Given Internal 回傳 `taskType="calendar"`  
And `operation="create"`  
And `summary`、`startAt`、`endAt` 皆存在  
And Google Calendar sync is enabled  
And OAuth token 可寫入 configured shared calendar  
When LineBot Backend 建立 calendar task  
Then LineBot Backend 應先寫入 `calendar_tasks/{taskId}`  
And 應呼叫 Google Calendar API 建立 event  
And Firestore 應保存 `googleCalendarEventId`、`googleCalendarHtmlLink`、`calendarSyncStatus="calendar_synced"`  
And response 應回傳 calendar sync result

```text
expected flow
calendar.UseCase.Create
├─ repository.Create(calendarSyncStatus=calendar_sync_pending)
├─ calendarProvider.CreateEvent(sharedCalendarId)
└─ repository.UpdateSyncResult(calendarSyncStatus=calendar_synced)
```

### Scenario: Google Calendar sync fails after Firestore create

Given Internal extraction result is valid  
And Firestore create succeeds  
And Google Calendar API returns an error  
When LineBot Backend handles the task  
Then Firestore 應保留已建立的 task  
And task `calendarSyncStatus` 應更新為 `calendar_sync_failed`  
And task 應保存 `calendarSyncError`  
And response 應清楚告知 sync failed  
And 不應遺失 Internal extraction result

### Scenario: Google Calendar sync disabled

Given Google Calendar sync is disabled by config  
When LineBot Backend 建立 calendar task  
Then LineBot Backend 只應寫入 Firestore  
And 不應呼叫 Google Calendar API  
And task `calendarSyncStatus` 應保持 `not_enabled`

### Scenario: missing required event time prevents sync

Given Internal 回傳 `startAt=""` or `endAt=""`  
When LineBot Backend 處理 extraction result  
Then LineBot Backend 不應寫入 Firestore  
And 不應呼叫 Google Calendar API  
And response 應回 `INTERNAL_EXTRACTION_INCOMPLETE`

### Scenario: startAt is missing

Given Internal 回傳 `taskType="calendar"`  
And `operation="create"`  
And `summary` 存在  
And `startAt=""`  
When LineBot Backend 處理此 extraction result  
Then LineBot Backend 不應寫入 Firestore  
And response 應回 `INTERNAL_EXTRACTION_INCOMPLETE`

### Scenario: endAt is missing

Given Internal 回傳 `taskType="calendar"`  
And `operation="create"`  
And `summary` 存在  
And `endAt=""`  
When LineBot Backend 處理此 extraction result  
Then LineBot Backend 不應寫入 Firestore  
And response 應回 `INTERNAL_EXTRACTION_INCOMPLETE`

### Scenario: summary is missing

Given Internal 回傳 `taskType="calendar"`  
And `operation="create"`  
And `summary=""`  
When LineBot Backend 處理此 extraction result  
Then LineBot Backend 不應寫入 Firestore  
And response 應回 `INTERNAL_EXTRACTION_INCOMPLETE`

## Scenario Group: Unsupported Task Or Operation

### Scenario: unsupported taskType

Given Internal 回傳 `taskType="unknown"`  
When LineBot Backend 處理此 extraction result  
Then LineBot Backend 不應寫入 Firestore  
And response 應回 `TASK_TYPE_UNSUPPORTED`

### Scenario: unsupported operation in first version

Given Internal 回傳 `taskType="calendar"`  
And `operation="update"`  
When LineBot Backend 處理此 extraction result  
Then LineBot Backend 不應寫入 Firestore  
And response 應回 `OPERATION_UNSUPPORTED`

## Scenario Group: Request Boundary

### Scenario: text is missing

Given local tester 呼叫 `POST /api/tasks`  
And request body 未帶 `text` 或 `text` 只有空白  
When gatekeeper 處理 request  
Then 不應呼叫 Internal gRPC  
And response 應回 `TEXT_REQUIRED`

### Scenario: request can override referenceTime and timeZone

Given local tester 呼叫 `POST /api/tasks`  
And request body 帶入 `referenceTime` 與 `timeZone`  
When task usecase 呼叫 Internal `LineTaskConsult`  
Then request 應將這兩個欄位原樣傳給 Internal

## Scenario Group: Future LINE Webhook

### Scenario: LINE message without bot mention is ignored

Given LINE webhook 收到 message event  
And message 沒有 tag bot  
When future linebot handler 處理 event  
Then 不應呼叫 task usecase

### Scenario: LINE message with bot mention uses the same task usecase

Given LINE webhook 收到 message event  
And message 有 tag bot  
When future linebot handler 移除 mention 文字  
Then 應呼叫與 REST 相同的 task usecase  
And 不應複製 calendar persistence 流程
