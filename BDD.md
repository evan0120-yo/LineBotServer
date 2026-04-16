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

目前驗收範圍只包含已落地能力：

```text
current scope
├─ POST /api/tasks
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
