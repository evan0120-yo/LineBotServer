# LineBot Backend Development Guide

## Purpose

這個專案是獨立的 LineBot Backend，不屬於 LinkChat，也不承擔 LinkChat 的任何產品邏輯。

核心責任：

```text
LineBot Backend
├─ 接收 REST / LINE 訊息入口
├─ 呼叫 Internal AI Copilot gRPC LineTaskConsult
├─ 依 operation 分派 calendar 行為
├─ 呼叫 Google Calendar
└─ 組成 LINE / REST 可直接使用的結果
```

Internal 是唯一的 AI 理解來源。LineBot Backend 不在本地重做 builder / prompt / Gemma 判斷。

## Project Boundary

```text
In scope
├─ REST local test entry
├─ LINE webhook entry
├─ Internal gRPC client
├─ calendar create/query/delete/update
├─ LINE reply formatting
└─ Google Calendar integration

Out of scope
├─ LinkChat
├─ Internal prompt / builder / aiclient implementation
├─ Firestore persistence
└─ 本地 AI 判斷
```

## Architecture

本專案維持三層加 UseCase 層，但現在外部系統只剩 Google Calendar。

```text
Handler
└─ UseCase
   └─ Service
      └─ External adapter
```

### Layer Responsibility

```text
Handler
├─ REST / LINE transport
├─ request parse
├─ signature verify
├─ mention cleanup
└─ response mapping

UseCase
├─ 單一業務案例編排
├─ 呼叫 Internal gRPC
├─ operation dispatch
└─ 呼叫 calendar module

Service
├─ deterministic business rules
├─ required field validation
├─ overlap filter
└─ reply formatting rule

External adapter
└─ Google Calendar API interaction
```

## Module Baseline

```text
Backend/
├─ cmd/api
└─ internal
   ├─ app
   ├─ gatekeeper
   ├─ task
   ├─ calendar
   ├─ internalclient
   └─ infra
```

### Responsibility Split

```text
gatekeeper
├─ REST handler
├─ LINE webhook handler
└─ shared request boundary

task
├─ supportedTaskTypes registry
├─ Internal request build
├─ operation validation
└─ operation factory dispatch

calendar
├─ create
├─ query
├─ delete
├─ update
└─ formatter

internalclient
└─ Internal AI Copilot gRPC transport wrapper

infra
├─ config
├─ errors
├─ HTTP envelope
└─ Google Calendar client
```

## First Design Flow

```text
REST or LINE
└─ gatekeeper
   └─ task.UseCase.ExecuteFromText
      ├─ call Internal LineTaskConsult
      ├─ validate taskType=calendar
      ├─ validate operation
      └─ calendar operation factory
         ├─ create
         ├─ query
         ├─ delete
         └─ update
```

## Internal Contract Rule

LineBot Backend 呼叫 Internal 的核心 request 不變：

```text
LineTaskConsult
├─ appId
├─ builderId
├─ messageText
├─ referenceTime
├─ timeZone
├─ supportedTaskTypes
└─ clientIp
```

回傳 contract 需要補：

```text
Internal structured result
├─ taskType
├─ operation
├─ eventId
├─ summary
├─ startAt
├─ endAt
├─ queryStartAt
├─ queryEndAt
├─ location
└─ missingFields
```

Rules:
- `eventId` 必須成為 prompt return 欄位之一。
- `create` 時可為空字串，後續以 Google Calendar create 結果覆蓋。
- `delete` / `update` 必須靠 `eventId` 執行。
- `query` 靠 `queryStartAt` / `queryEndAt`。

## Operation Rule

### Create

```text
create
├─ validate summary/startAt/endAt
├─ call Google Calendar create
└─ format reply using returned eventId
```

### Query

```text
query
├─ use time range only
├─ fetch candidate events from Google Calendar
├─ apply overlap rule
└─ return 0..N formatted event rows
```

### Delete

```text
delete
├─ require eventId
└─ call Google Calendar delete directly
```

### Update

```text
update
├─ require eventId
├─ require new summary
└─ call Google Calendar update title only
```

## Query Rule

query 主條件只用時間範圍，不用 title 搜尋。

```text
query input
├─ queryStartAt
└─ queryEndAt
```

overlap 規則固定：

```text
eventStart <= queryEnd
AND
eventEnd >= queryStart
```

## LINE Reply Rule

固定格式：

```text
0001
小傑約明天吃晚餐
2026-04-18 12:00 (週五) ~ 2026-04-18 12:30 (週五)
```

Rules:
- 第一行直接是 `eventId`
- 不加 `eventId:` 前綴
- 沒資料回 `沒資料`
- 錯誤回精簡錯誤
- 不把整個 log 回給使用者

## Removed Rule

這次設計的明確方向：

```text
removed
├─ Firestore collection
├─ Firestore repository
├─ Firestore sync status
└─ Firestore as source of truth
```

也就是：

```text
Calendar
└─ becomes the only data source for
   ├─ create result
   ├─ query result
   ├─ delete target
   └─ update target
```

## Google Calendar Rule

LineBot Backend 直接以 Google Calendar 當唯一外部資料來源。

```text
Google Calendar
├─ create -> events.insert
├─ query  -> events.list + overlap filter
├─ delete -> events.delete
└─ update -> events.patch / update title
```

需要的最小資訊：

```text
create
├─ summary
├─ startAt
└─ endAt

query
├─ queryStartAt
└─ queryEndAt

delete
└─ eventId

update
├─ eventId
└─ summary
```

## Logging Direction

大量 log 能讓 AI 在下次 debug / 討論時直接讀 log，而不需要重新追 code 或反覆問 context，大幅省 token。

原則：**寫越多 INFO log 越好**，上線前再統一拔掉或降級。

```text
log 的時機
├─ 任何 branch decision 點
│  ├─ operation 走哪條
│  ├─ mention 有沒有命中
│  ├─ 驗證通過 / 失敗原因
│  └─ fallback 觸發
├─ 每次呼叫外部系統前後
│  ├─ Internal gRPC call (request + response summary)
│  ├─ Google Calendar API call
│  └─ LINE reply 送出
├─ 每個 usecase 入口
│  ├─ 收到的 text
│  ├─ 解出的 operation
│  └─ 最終回覆內容
└─ 任何 error / unexpected state
   ├─ 欄位缺失詳細列出是哪個
   ├─ external API error 完整 message
   └─ unexpected taskType / operation 值是什麼
```

格式建議：

```text
log key-value 結構
├─ 用 slog 或 log.Printf 帶 key=value 方便 grep
├─ 每條 log 起碼含：[module] + [action] + [key values]
└─ 範例：
   [task] operation=create summary="小傑約明天吃晚餐" startAt=...
   [calendar] create ok eventId=xxxx
   [line] mention found cleanedText="..."
   [gatekeeper] validation failed reason="missing summary"
```

> 注意：這些 log 是開發期工具，上線前確認移除或改為 DEBUG level。

## Testing Direction

先補：

```text
1. task operation dispatch
2. query overlap rule
3. line reply formatter
4. LINE webhook boundary
```

再補：

```text
5. Google Calendar create/query/delete/update integration
```

## Documentation Rule

本專案文件角色分工：

```text
BDD
└─ 對外可觀察行為與驗收

SDD
└─ 模組邊界 / flow / contract / ownership

TDD
└─ 測試策略與測項規劃

CODE_REVIEW
└─ 目前 code 真相
```

當設計先行、code 尚未落地時：
- BDD / SDD / TDD / PLAN / DEVELOPMENT 可以先反映目標設計
- `CODE_REVIEW.md` 必須保持 code-first，不得把未落地設計寫成現況
