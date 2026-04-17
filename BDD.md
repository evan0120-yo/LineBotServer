# LineBot Backend BDD

## Purpose

這份文件只定義對外可觀察行為與驗收條件。

```text
BDD scope
├─ input / request
├─ observable side effect
├─ LINE reply / API output
└─ what must not happen
```

這份文件不處理：
- package 怎麼切
- function call chain
- 目前 code 實作細節
- 未來 roadmap 敘事

## Scope

這份 BDD 反映目前已確認的目標行為：

```text
target scope
├─ REST local test entry
│  └─ POST /api/tasks
├─ LINE webhook entry
│  └─ POST /api/line/webhook
├─ Internal LineTaskConsult extraction
├─ Google Calendar as only persistence source
└─ operation factory
   ├─ create
   ├─ query
   ├─ delete
   └─ update
```

## Shared Output Rule

### Scenario: single event reply format

Given LineBot Backend 要回傳單一筆 calendar 結果  
When create / query(one result) / update 成功  
Then reply 文字格式必須固定為三行

```text
0001
小傑約明天吃晚餐
2026-04-18 12:00 (週五) ~ 2026-04-18 12:30 (週五)
```

And 第一行必須直接是 `eventId`  
And 不得出現 `eventId:` 這種前綴  
And 時間必須格式化成：

```text
yyyy-MM-dd HH:mm (週X) ~ yyyy-MM-dd HH:mm (週X)
```

### Scenario: multiple event reply format

Given query 回傳多筆事件  
When LineBot Backend 組 LINE reply  
Then 每筆事件必須各自用三行格式表示  
And 多筆事件之間可以用空行分隔  
And 每筆都必須直接顯示 `eventId`

### Scenario: no data reply

Given query 沒查到任何事件  
When LineBot Backend 組 reply  
Then reply 必須是 `沒資料`

### Scenario: error reply

Given create / query / delete / update 任一路徑失敗  
When LineBot Backend 組 reply  
Then reply 必須回精簡錯誤訊息  
And 不得把完整 log 或 stack trace 丟到 LINE

## Create

### Scenario: create event from natural language

Given local tester 呼叫 `POST /api/tasks`  
Or LINE webhook 收到有 mention bot 的文字訊息  
When Internal 回傳：
- `taskType="calendar"`
- `operation="create"`
- `summary`
- `startAt`
- `endAt`
Then LineBot Backend 應呼叫 Google Calendar create  
And 應使用 Google Calendar 回傳的 `eventId` 組 reply  
And reply 必須符合三行格式

### Scenario: create allows empty location

Given Internal 回傳 `operation="create"`  
And `summary`、`startAt`、`endAt` 皆存在  
And `location=""`  
When LineBot Backend 建立 Google Calendar event  
Then 應照常建立成功  
And 不應因 `location` 空值失敗

### Scenario: create rejects missing required fields

Given Internal 回傳 `operation="create"`  
And `summary` 或 `startAt` 或 `endAt` 缺失  
When LineBot Backend 處理 create  
Then 不應呼叫 Google Calendar create  
And reply 應回精簡錯誤  
And 不應產生任何 event

## Query

### Scenario: query uses time range only

Given Internal 回傳 `operation="query"`  
And 帶有 `queryStartAt` 與 `queryEndAt`  
When LineBot Backend 查詢 Google Calendar  
Then 查詢主條件只應使用時間區間  
And 不應使用 title / summary keyword 做主查詢條件

### Scenario: query returns overlapping events

Given Google Calendar 有一筆事件：

```text
12:00 ~ 15:00
```

When query 區間為以下任一種：

```text
12:00 ~ 12:30
14:30 ~ 15:30
11:30 ~ 12:00
```

Then 該事件都必須出現在查詢結果內

### Scenario: query overlap rule

Given query 不是查完全落入區間的事件  
When LineBot Backend 過濾 query 結果  
Then overlap 規則必須是：

```text
eventStart <= queryEnd
AND
eventEnd >= queryStart
```

### Scenario: query returns eventId

Given query 查到一筆或多筆事件  
When LineBot Backend 組 reply  
Then 每筆結果都必須帶出 Google Calendar 的 `eventId`  
And `eventId` 必須直接顯示在第一行  
And 使用者可以直接複製該 `eventId`

## Delete

### Scenario: delete by eventId

Given Internal 回傳 `operation="delete"`  
And 回傳 `eventId="0001"`  
When LineBot Backend 執行 delete  
Then 應直接呼叫 Google Calendar delete by `eventId`  
And reply 應只回成功或失敗

### Scenario: delete does not need title or time range

Given delete request 已有 `eventId`  
When LineBot Backend 執行 delete  
Then 不應再依 title 搜尋  
And 不應再依時間範圍搜尋

## Update

### Scenario: update title by eventId

Given Internal 回傳 `operation="update"`  
And 回傳 `eventId="0001"`  
And 回傳新的 `summary`  
When LineBot Backend 執行 update  
Then 應直接呼叫 Google Calendar update 該 `eventId`  
And 只更新 title  
And reply 應回更新後的三行結果

### Scenario: update does not change time when title-only update

Given update request 只帶 `eventId` 與新 `summary`  
When LineBot Backend 執行 update  
Then 不應修改該事件原本的開始與結束時間

## LINE Webhook

### Scenario: message without mention is ignored

Given LINE webhook 收到文字訊息  
And 訊息沒有 mention bot  
When LineBot Backend 處理 webhook  
Then 不應呼叫 Internal  
And 不應呼叫 Google Calendar  
And 不應回任務結果

### Scenario: message with mention uses same task usecase as REST

Given LINE webhook 收到有 mention bot 的文字訊息  
When LineBot Backend 處理 webhook  
Then 應先移除 mention 文字  
And 之後必須走和 REST 相同的 task usecase 邏輯

### Scenario: invalid signature is rejected

Given LINE webhook request 的 signature 不正確  
When LineBot Backend 驗證 webhook  
Then 應直接拒絕  
And 不應呼叫 Internal  
And 不應呼叫 Google Calendar

## Unsupported Behavior

### Scenario: unsupported taskType

Given Internal 回傳 `taskType="unknown"`  
When LineBot Backend 處理該結果  
Then 不應執行任何 Calendar 操作  
And reply 應回精簡錯誤

### Scenario: unsupported operation

Given Internal 回傳 `operation="unknown"`  
When LineBot Backend 處理該結果  
Then 不應執行任何 Calendar 操作  
And reply 應回精簡錯誤
