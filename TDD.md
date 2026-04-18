# LineBot Backend TDD

## Purpose

這份文件定義新的 Calendar-only 架構要怎麼測、測到哪裡、為什麼這樣測。

```text
test scope
├─ operation factory
├─ Calendar integration behavior
├─ LINE webhook boundary
└─ reply formatting
```

## Test Strategy

這一輪重點不再是 Firestore，而是：

```text
priority
├─ task operation dispatch
├─ calendar create/query/delete/update
├─ query overlap rule
├─ LINE webhook mention flow
└─ LINE reply formatting
```

## Unit Tests

### task operation validation

應覆蓋：

```text
task.Service / task.UseCase
├─ taskType=calendar accepted
├─ unsupported taskType rejected
├─ operation=create accepted
├─ operation=query accepted
├─ operation=delete accepted
├─ operation=update accepted
└─ unknown operation rejected
```

### query overlap rule

應覆蓋：

```text
overlap
├─ event fully contains query range
├─ query fully contains event
├─ overlap at left boundary
├─ overlap at right boundary
├─ exact same range
└─ no overlap
```

代表案例：

```text
event: 12:00 ~ 15:00
query: 12:00 ~ 12:30 -> match
query: 14:30 ~ 15:30 -> match
query: 11:30 ~ 12:00 -> match
query: 15:01 ~ 16:00 -> no match
```

### reply formatting

應覆蓋：

```text
formatter
├─ single event -> 3 lines
├─ multiple events -> blank line separator
├─ no data -> "沒資料"
├─ error -> concise message
└─ eventId has no prefix
```

## Integration Tests

### calendar create

應覆蓋：

```text
create integration
├─ valid create command -> events.insert called
├─ returned eventId propagated to result
├─ location empty still succeeds
├─ location provided → written to Google Calendar event
└─ missing summary/startAt/endAt rejected
```

### calendar query

應覆蓋：

```text
query integration
├─ query uses time range only
├─ Google Calendar returns candidate events
├─ overlap filter keeps matching events
└─ result maps eventId / summary / startAt / endAt
```

### calendar delete

應覆蓋：

```text
delete integration
├─ eventId present -> events.delete called
├─ eventId missing -> reject
└─ provider error -> concise delete failure
```

### calendar update

應覆蓋：

```text
update integration
├─ eventId + summary -> events.patch called
├─ title updated only
├─ eventId missing -> reject
└─ provider error -> concise update failure
```

## LINE Webhook Tests

應覆蓋：

```text
line webhook
├─ invalid signature rejected
├─ non-text event ignored
├─ no mention ignored
├─ mention cleaned correctly
├─ accepted event enters shared task usecase
└─ webhook still acks request-level 200
```

## Manual Verification

這輪仍保留必要手動驗證，因為 LINE 與 Google Calendar 屬於真實外部整合。

### Manual flow

```text
manual verification
├─ Postman -> POST /api/tasks -> create
├─ LINE group mention -> create
├─ query by time range
├─ copy returned eventId
├─ delete by eventId
└─ update by eventId + new title
```

### Manual checkpoints

```text
checkpoints
├─ create reply shows real eventId
├─ query can find overlapping event
├─ query no result returns "沒資料"
├─ delete removes target event
├─ update changes title only
└─ error reply stays concise
```

## Risk-Based Coverage

最容易出錯的地方：

```text
risk hotspots
├─ Internal operation dispatch
├─ query overlap filter
├─ eventId mapping
├─ LINE mention cleanup
└─ reply formatting contract
```

所以第一批自動化測試應先鎖：

```text
phase 1
├─ task dispatch
├─ overlap rule
├─ formatter
└─ webhook boundary
```

第二批再補：

```text
phase 2
├─ Google Calendar create
├─ Google Calendar query
├─ Google Calendar delete
└─ Google Calendar update
```

## Deprecated Coverage

這次設計完成後，下列測試不再是目標主軸：

```text
deprecated
├─ Firestore mapping
├─ Firestore sync status
└─ Firestore persistence recovery
```
