# LineBot Backend Plan

## Block 1 - Project Overview

### Project Purpose

LineBot Backend 是「LINE / REST 任務入口 + Internal AI task 轉譯 + Google Calendar 執行器」。

目前目標已經很單純：

```text
LINE / REST
└─ LineBot Backend
   └─ Internal AI Copilot
      └─ structured JSON
         └─ Google Calendar
```

它不是資料平台，也不是多模組資料庫系統。這一輪的核心定位是：

```text
small focused backend
├─ 接自然語句
├─ 丟給 Internal 轉穩定 JSON
├─ 執行 calendar operation
└─ 回可直接貼回 LINE 的結果
```

### What It Is

```text
it is
├─ LineBot server
├─ REST local test server
├─ Internal AI consumer
└─ Google Calendar operator
```

### What It Is Not

```text
it is not
├─ LinkChat
├─ 本地 AI pipeline
├─ Firestore-based task system
└─ 通用型任務平台
```

### First Intentional Shape

這一版設計刻意選擇：

```text
intentional choices
├─ Firestore 拔掉
├─ Google Calendar 當唯一資料來源
├─ create/query/delete/update 直接對 Calendar 做
├─ query 只靠時間區間
└─ delete/update 直接靠 eventId
```

### User-Facing Rule

回覆格式固定：

```text
0001
小傑約明天吃晚餐
2026-04-18 12:00 (週五) ~ 2026-04-18 12:30 (週五)
```

補充規則：
- `eventId` 不加前綴
- 查無資料回 `沒資料`
- 錯誤回精簡訊息
- 不把 log 原文回給使用者

## Block 2 - Module Map And Flow

### High-Level Architecture

```text
LineBot Backend
├─ transport
│  ├─ POST /api/tasks
│  └─ POST /api/line/webhook
├─ orchestration
│  └─ task.UseCase
├─ operation factory
│  └─ calendar module
│     ├─ create
│     ├─ query
│     ├─ delete
│     └─ update
├─ ai integration
│  └─ internalclient
└─ external integration
   └─ Google Calendar
```

### Package Layout

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

### Flow: Create

```text
REST / LINE
└─ gatekeeper
   └─ task.UseCase
      ├─ call Internal LineTaskConsult
      ├─ validate taskType=calendar
      ├─ validate operation=create
      └─ calendar.Create
         ├─ validate summary/startAt/endAt
         ├─ events.insert
         └─ format reply
```

### Flow: Query

```text
REST / LINE
└─ gatekeeper
   └─ task.UseCase
      ├─ call Internal LineTaskConsult
      ├─ validate operation=query
      └─ calendar.Query
         ├─ use queryStartAt/queryEndAt
         ├─ list candidate events
         ├─ apply overlap filter
         └─ format 0..N rows
```

### Flow: Delete

```text
REST / LINE
└─ gatekeeper
   └─ task.UseCase
      ├─ call Internal LineTaskConsult
      ├─ validate operation=delete
      └─ calendar.Delete
         ├─ require eventId
         └─ events.delete
```

### Flow: Update

```text
REST / LINE
└─ gatekeeper
   └─ task.UseCase
      ├─ call Internal LineTaskConsult
      ├─ validate operation=update
      └─ calendar.Update
         ├─ require eventId
         ├─ require new summary
         └─ events.patch title
```

### Query Overlap Rule

query 的核心不是完全落入區間，而是有交集就算命中。

```text
match when
eventStart <= queryEnd
AND
eventEnd >= queryStart
```

例子：

```text
event = 12:00 ~ 15:00

query 12:00 ~ 12:30 -> hit
query 14:30 ~ 15:30 -> hit
query 11:30 ~ 12:00 -> hit
```

### Internal Contract Direction

Internal 仍然是 AI 結構化的唯一來源，但回傳 contract 要補 `eventId`。

```text
response shape
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

規則：
- `create` 時 eventId 可空
- `query` 回傳查詢條件欄位
- `delete` / `update` 直接使用 eventId

## Block 3 - Detailed Notes

### Design Rules

```text
rules
├─ supportedTaskTypes 先維持 ["calendar"]
├─ operation 工廠拆四條
├─ Firestore 互動全部砍掉
├─ query 不用主旨搜尋
├─ query 只靠時間區間
├─ delete 直接吃 eventId
└─ update 直接吃 eventId + new title
```

### Minimal Calendar Inputs

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

### Reply Contract

single result:

```text
0001
小傑約明天吃晚餐
2026-04-18 12:00 (週五) ~ 2026-04-18 12:30 (週五)
```

multiple results:

```text
0001
小傑約明天吃晚餐
2026-04-18 12:00 (週五) ~ 2026-04-18 12:30 (週五)

0002
回診
2026-04-18 15:00 (週五) ~ 2026-04-18 15:30 (週五)
```

no result:

```text
沒資料
```

### Build Order

建議實作順序：

```text
1. Internal contract 補 eventId/queryStartAt/queryEndAt
2. 拔掉 Firestore create flow
3. calendar create 直接回 eventId
4. 補 query + overlap rule
5. 補 delete by eventId
6. 補 update title by eventId
7. 補 formatter 與 LINE reply
```

### Documentation Rule

這份 PLAN 記的是目標設計，不是當前 code 真相。  
當前 code 真相應看 `CODE_REVIEW.md`。
