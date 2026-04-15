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
│  └─ Firestore persistence
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

