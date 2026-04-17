# LineBot Backend Code Review

---

# BLOCK 1: AI 對產品的想像

這個專案現在已經是一個很明確的「LINE / REST 任務入口 + Internal AI 結構化轉譯 + Google Calendar 執行器」。

它不是 LinkChat，也不是通用型任務平台。從目前 code 來看，它就是專門接自然語句，交給 Internal 轉成穩定 JSON，然後直接對 Google Calendar 做事。

我現在對它的理解是：

```text
current product shape
├─ LINE 群組中可 tag bot
├─ REST 也可直接打本機測試
├─ Internal 是唯一 AI 理解來源
├─ Calendar 是唯一資料來源
└─ 回覆文字已整理成可直接貼回 LINE 的格式
```

它現在不是：

```text
it is not
├─ Firestore-backed task system
├─ multi-module database platform
├─ 本地 prompt / builder / Gemma 管理器
└─ LinkChat runtime
```

如果只看目前落地的 code，這版已經不是「create-only + Firestore sync」了，而是 Calendar-only 的四條 operation 主幹：

```text
current code
├─ create
├─ query
├─ delete
└─ update
```

---

# BLOCK 2: 讀者模式

## A. 啟動後這支服務會接好哪些東西

啟動時它會把兩個外部系統接起來：
- Internal gRPC
- Google Calendar

如果 LINE access token / channel secret / bot user id 都有設定，還會順便把 LINE webhook 開起來。

```text
startup
├─ load env config
├─ create Internal gRPC client
├─ optional create Google Calendar client
├─ wire calendar module
├─ wire task module
├─ wire gatekeeper module
└─ register routes
   ├─ POST /api/tasks
   └─ POST /api/line/webhook
```

> 注意：現在 route 是否註冊是 fail-closed。LINE 三個必要設定沒齊時，`/api/line/webhook` 不會開。

## B. REST 路線現在怎麼跑

`POST /api/tasks` 還是最直接的本機測試入口。

它做的事很單純：

```text
POST /api/tasks
└─ gatekeeper.Handler.CreateTask
   ├─ parse JSON
   ├─ validate text
   └─ gatekeeper.UseCase.CreateTask
      └─ task.UseCase.CreateFromText
         ├─ call Internal LineTaskConsult
         ├─ validate taskType
         ├─ validate operation
         └─ dispatch calendar operation
```

現在 task usecase 不再先寫 Firestore，再同步 Calendar。它是直接讓 Calendar module 操作 Google Calendar。

## C. LINE webhook 路線現在怎麼跑

LINE webhook 現在是另一個 transport boundary，但後面共用同一條 task usecase。

```text
POST /api/line/webhook
└─ gatekeeper.LineHandler
   ├─ verify x-line-signature
   ├─ parse webhook JSON
   ├─ filter text message event
   ├─ verify bot mention
   ├─ remove mention text
   ├─ call same gatekeeper usecase
   ├─ reply LINE text
   └─ return request-level 200 ack
```

這條線現在已經補了足夠的 INFO log，所以在 CMD 直接能看到：
- webhook 有沒有進來
- mention 有沒有命中
- 清理後文字是什麼
- Internal gRPC 有沒有成功
- operation 最後走到哪裡

> 注意：每個 event 目前用自己的 timeout context 跑，不再把 Internal gRPC 直接綁死在 `r.Context()` 上。

## D. Calendar 模組現在真的做了什麼

現在 calendar module 已經不是單一 create service，而是四條 operation 都有落地：

```text
calendar module
├─ Create
├─ Query
├─ Delete
└─ Update
```

### create

```text
create
├─ validate summary/startAt/endAt
├─ call Google Calendar events.insert
└─ format 3-line reply
```

### query

```text
query
├─ validate queryStartAt/queryEndAt
├─ list candidate events from Google Calendar
├─ apply overlap rule in service layer
└─ format 0..N rows
```

query 的命中規則是：

```text
eventStart <= queryEnd
AND
eventEnd >= queryStart
```

也就是只要事件和查詢區間有交集，就算命中。

### delete

```text
delete
├─ require eventId
└─ call Google Calendar events.delete
```

### update

```text
update
├─ require eventId
├─ require summary
└─ call Google Calendar events.patch
```

## E. Internal contract 現在怎麼被用

LineBot Backend 還是只把自然語句丟給 Internal 做理解。

Internal 回來的結果，現在除了原本的：
- `taskType`
- `operation`
- `summary`
- `startAt`
- `endAt`
- `location`
- `missingFields`

還已經多了：
- `eventId`
- `queryStartAt`
- `queryEndAt`

現在四條 operation 的依賴關係是：

```text
Internal response
├─ create
│  └─ 用 summary/startAt/endAt 建立 Google event
├─ query
│  └─ 用 queryStartAt/queryEndAt 查 Google Calendar
├─ delete
│  └─ 用 eventId 刪 Google event
└─ update
   └─ 用 eventId + summary 改標題
```

> 注意：create 的最終 `eventId` 不是 Internal 生的，而是 Google Calendar create 成功後真正回來的 event id。

## F. 回覆格式現在的真相

create / query / update 的使用者回覆，都會走同一個 formatter。

格式是：

```text
eventId
summary
yyyy-MM-dd HH:mm (週X) ~ yyyy-MM-dd HH:mm (週X)
```

多筆結果之間會用空行分隔。  
沒有資料就固定回：

```text
沒資料
```

delete 成功就回：

```text
刪除成功
```

錯誤則不回整包 log，而是透過 `infra.ErrorReplyMessage` 回精簡訊息。

---

# BLOCK 3: 技術補充

## A. 主要 entrypoints 與呼叫鏈

### REST

```text
cmd/api/main.go
└─ app.New
   └─ gatekeeper.NewHandler
      └─ Handler.CreateTask
         └─ gatekeeper.UseCase.CreateTask
            └─ task.UseCase.CreateFromText
               └─ executeCalendarOperation
```

### LINE webhook

```text
cmd/api/main.go
└─ app.New
   └─ gatekeeper.NewLineHandler
      └─ LineHandler.ServeHTTP
         └─ gatekeeper.UseCase.CreateTask
            └─ task.UseCase.CreateFromText
               └─ executeCalendarOperation
```

## B. 現在 task operation 驗證的真相

`internal/task/service.go` 現在接受：

```text
supported operations now
├─ create
├─ query
├─ delete
└─ update
```

不是舊版的 create-only。

## C. Firestore 已不在主流程中

現在 LineBot 專案內部已經沒有：

```text
removed from runtime
├─ Firestore store bootstrap
├─ calendar repository
├─ calendar_tasks persistence
├─ sync status writeback
└─ Firestore as source of truth
```

如果用現在的 code 看，它唯一的外部資料操作就是 Google Calendar。

## D. 現在有的測試

目前這次改動後，LineBot repo 真正有自動化覆蓋的主區塊是：

```text
tests that exist now
├─ internal/calendar/service_test.go
├─ internal/calendar/usecase_test.go
├─ internal/gatekeeper/line_handler_test.go
├─ internal/infra/http_test.go
└─ internal/task/service_test.go
```

這些測試已經碰到的重點有：
- overlap rule
- create/query/delete/update validation
- LINE mention cleanup
- request-level webhook ack
- strict JSON helper

## E. 目前驗證狀態

這輪我能直接確認通過的有：

```text
verified now
├─ LineBot Backend: go test ./...
├─ LineBot Backend: go vet ./...
├─ Internal frontend: npm exec tsc -- --noEmit
├─ Internal backend touched packages
│  ├─ internal/aiclient
│  ├─ internal/builder (targeted line-task test)
│  └─ internal/grpcapi
└─ protobuf contract 已更新到實際 import 路徑
```

另外有一件要誠實記下來：

```text
repo baseline issue
└─ Internal Go 的部分 gatekeeper 測試會在 store reset/seed 階段卡住 Firestore query
```

這個 timeout 從 stack trace 看，是測試啟動本地 store 時打到 Firestore `RunQuery` 卡住，不是這輪 LineBot / Calendar-only contract 改動本身造成的 compile error。

## F. 這版 code 最值得保留的切口

如果後面還要繼續擴，現在最值得保留的是：

```text
good seams now
├─ transport boundary 分開
│  ├─ REST
│  └─ LINE webhook
├─ shared task usecase
├─ Internal contract wrapper
├─ calendar-only operation module
└─ LINE reply formatter
```

也就是說，現在要繼續擴 query / delete / update 或調整回覆格式，基本上都不用重切整個架構。
