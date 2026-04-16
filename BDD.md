# LineBot Backend BDD

## Purpose

這份文件只定義「對外可觀察行為」與「驗收條件」。

```text
BDD scope
├─ request input
├─ observable side effect
├─ response / error
└─ what must not happen
```

不在這份文件裡處理：
- package 怎麼切
- function / call chain
- 未來 module 設計
- 目前 code 細節導覽

## Scope

目前驗收範圍包含已落地能力：

```text
current scope
├─ POST /api/tasks (REST API)
├─ POST /api/line/webhook (LINE Bot)
├─ Internal LineTaskConsult extraction
├─ Firestore calendar_tasks create
└─ optional Google Calendar create sync
```

## Scenario Group: Request Boundary

### Scenario: text is missing

Given local tester 呼叫 `POST /api/tasks`  
And request body 未帶 `text` 或 `text` 只有空白  
When LineBot Backend 處理 request  
Then 不應呼叫 Internal gRPC  
And response 應回 `TEXT_REQUIRED`

### Scenario: request can override referenceTime and timeZone

Given local tester 呼叫 `POST /api/tasks`  
And request body 帶入 `referenceTime` 與 `timeZone`  
When LineBot Backend 呼叫 Internal `LineTaskConsult`  
Then request 應將這兩個欄位原樣傳給 Internal

## Scenario Group: Calendar Create

### Scenario: create calendar task from natural language

Given local tester 呼叫 `POST /api/tasks`  
And request body 帶入 `text="小傑約明天吃午餐"`  
And LineBot Backend config 已設定 Internal `appId`、`builderId`、`supportedTaskTypes=["calendar"]`  
When Internal 回傳 `taskType="calendar"`、`operation="create"`、`summary`、`startAt`、`endAt`  
Then LineBot Backend 應建立 `calendar_tasks/{taskId}`  
And response 應回傳 `taskId` 與 extraction result

```text
observable result
POST /api/tasks
├─ response 200
└─ Firestore calendar_tasks/{taskId}
```

### Scenario: location is missing but task is still created

Given Internal 回傳 `taskType="calendar"`  
And `operation="create"`  
And `summary`、`startAt`、`endAt` 皆存在  
And `location=""`  
And `missingFields=["location"]`  
When LineBot Backend 處理此 extraction result  
Then LineBot Backend 應照常寫入 Firestore  
And response 應成功回傳  
And 不應因 `location` 缺失回錯

### Scenario: required extraction field is missing

Given Internal 回傳 `taskType="calendar"`  
And `operation="create"`  
And `summary` 或 `startAt` 或 `endAt` 缺失  
When LineBot Backend 處理 extraction result  
Then LineBot Backend 不應寫入 Firestore  
And 不應呼叫 Google Calendar API  
And response 應回 `INTERNAL_EXTRACTION_INCOMPLETE`  
And error 應包含對應的 `missingFields`

## Scenario Group: Google Calendar Sync

這組場景只驗收 create sync 的對外結果。

```text
Google Calendar sync boundary
├─ Firestore 永遠先保存 task
├─ Google Calendar 是外部同步目標
└─ Firestore task 不會因 sync failure 消失
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

## Scenario Group: Unsupported Behavior

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

## Scenario Group: LINE Webhook

這組場景驗收 LINE Bot webhook 整合。

```text
LINE webhook boundary
├─ POST /api/line/webhook
├─ 驗證 LINE signature
├─ 解析 LINE webhook events
├─ 過濾 message event
├─ 檢查 mention (tag bot)
└─ 共用 task usecase (與 REST API 相同流程)
```

### Scenario: LINE message without mention is ignored

Given LINE webhook 收到 message event
And message text = "明天吃午餐"
And message 沒有 mention bot
When LineBot Backend 處理 webhook
Then 不應呼叫 task usecase
And 不應呼叫 Internal gRPC
And 不應建立 Firestore task

### Scenario: LINE message with mention creates task

Given LINE webhook 收到 message event
And message text = "@bot 小傑約明天吃午餐"
And message 有 mention bot
When LineBot Backend 處理 webhook
Then 應移除 mention 文字
And 應呼叫 task usecase with text = "小傑約明天吃午餐"
And 應建立 `calendar_tasks/{taskId}` with `source="line"`
And 不應因為 mention 影響 extraction 結果

### Scenario: LINE webhook signature verification fails

Given LINE webhook 收到 request
And `x-line-signature` header 不正確
When LineBot Backend 驗證 signature
Then 不應處理 request body
And 不應呼叫 task usecase
And response 應回 401 或 403

### Scenario: LINE webhook with non-message event

Given LINE webhook 收到 event
And event type = "follow" 或 "unfollow" 或其他非 message event
When LineBot Backend 處理 webhook
Then 不應呼叫 task usecase
And 不應建立 Firestore task

### Scenario: LINE message with mention in group chat

Given LINE webhook 收到 message event
And source type = "group"
And message text = "@bot 提醒我明天開會"
And message 有 mention bot
When LineBot Backend 處理 webhook
Then 應移除 mention 文字
And 應呼叫 task usecase
And 應建立 task with `source="line"`

### Scenario: LINE webhook with multiple valid message events

Given LINE webhook 一次收到多個 message event
And 至少一筆 event mention bot 且可成功建立 task
When LineBot Backend 處理 webhook
Then 應逐筆處理可用的 message event
And 不應因為其中一筆 event 失敗而中止整個 webhook
And webhook response 應回 200 ack，避免整包 request 被 LINE 重送

### Scenario: LINE message without mention in private chat

Given LINE webhook 收到 message event
And source type = "user" (私聊)
And message text = "小傑約明天吃午餐"
And message 沒有 mention bot
When LineBot Backend 處理 webhook
Then 第一版應忽略此訊息 (需要 mention 才處理)
And 不應呼叫 task usecase

> 注意：第一版統一要求 mention，避免在私聊時誤觸發。未來可放寬為「群組需要 mention，私聊不需要」。
