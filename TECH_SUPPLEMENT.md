# LineBot Backend Tech Supplement

這份文件是 CODE_REVIEW.md 的技術補充，對應原本的 BLOCK 3。
主要面向：需要深入了解 call chain、測試覆蓋、設計決策的讀者。

> 快速導覽請看 CODE_REVIEW.md 的 BLOCK 1 / BLOCK 2。

---

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

---

**文件版本**：v1.0
**最後更新**：2026-04-17
**作者**：Claude Sonnet 4.6
