# LineBot Backend SDD

## Purpose

這份文件只定義系統內部怎麼切、怎麼接、誰負責什麼。

```text
SDD scope
├─ module boundary
├─ dependency direction
├─ runtime data flow
├─ ownership
└─ contracts / schema
```

不在這份文件裡處理：
- 大量驗收場景
- 目前 code 導覽口吻
- bug / risk review
- 測試清單

## System Overview

```text
LineBot Backend
├─ transport boundary
│  ├─ REST first version
│  └─ LINE webhook future
├─ task orchestration
│  ├─ call Internal AI Copilot
│  ├─ receive taskType + operation
│  └─ dispatch to feature module
├─ feature modules
│  └─ calendar + optional Google Calendar sync
└─ persistence / integration
   ├─ Firestore
   └─ Google Calendar shared calendar
```

## Package Architecture

```text
Backend/
├─ cmd/api
│  └─ process entrypoint
└─ internal
   ├─ app
   │  └─ config / store / module wiring
   ├─ gatekeeper
   │  └─ REST and future LINE request boundary
   ├─ task
   │  └─ Internal extraction + task dispatch
   ├─ calendar
   │  └─ calendar task usecase / service / repository
   ├─ internalclient
   │  └─ Internal AI Copilot gRPC client
   └─ infra
      └─ config / errors / response / Firestore / Google Calendar client
```

## Module Responsibilities

```text
app
├─ load config
├─ create store / gRPC client / Google client
└─ wire modules and HTTP router

gatekeeper
├─ parse request
├─ validate boundary input
├─ resolve client IP
└─ map HTTP <-> task usecase

task
├─ build LineTaskConsult request
├─ validate supported taskType / operation
└─ dispatch to feature module

calendar
├─ validate required extraction fields
├─ persist calendar task
└─ optionally sync external Google Calendar event

internalclient
├─ hide protobuf details
└─ map Internal gRPC request / response

infra
├─ shared config
├─ business errors
├─ HTTP envelope
├─ Firestore store
└─ Google Calendar adapter
```

## Dependency Direction

```text
cmd/api
└─ app
   ├─ gatekeeper
   │  └─ task
   │     ├─ internalclient
   │     └─ calendar
   │        └─ infra
   └─ infra
```

Allowed:

```text
gatekeeper -> task
task -> internalclient
task -> calendar
calendar -> infra
internalclient -> infra
app -> all modules for wiring
```

Avoid:

```text
calendar -> task
internalclient -> calendar
repository -> usecase
handler -> repository
```

## Main Runtime Flow

```text
POST /api/tasks
│
▼
gatekeeper.Handler
├─ decode request
├─ validate text
└─ call gatekeeper.UseCase
   │
   ▼
gatekeeper.UseCase
└─ call task.UseCase.CreateFromText
   │
   ▼
task.UseCase
├─ build Internal request
├─ call internalclient.Service.LineTaskConsult
├─ validate taskType
├─ validate operation
└─ dispatch by taskType
   │
   ▼
calendar.UseCase.Create
├─ ValidateCreate(summary/startAt/endAt)
├─ Repository.Create
│  └─ Firestore calendar_tasks
└─ optional infra.GoogleCalendarProvider.CreateEvent
   └─ Repository.UpdateSyncResult
```

## Google Calendar Sync Design

```text
ownership
├─ Firestore
│  └─ task source of truth
├─ Google Calendar shared calendar
│  └─ user-visible external calendar
└─ OAuth token
   └─ authorizes write access to configured shared calendar
```

```text
calendar.UseCase.Create
├─ Repository.Create
│  ├─ sync enabled  -> calendarSyncStatus=calendar_sync_pending
│  └─ sync disabled -> calendarSyncStatus=not_enabled
│
├─ sync disabled?
│  └─ return Firestore-only result
│
├─ infra.GoogleCalendarProvider.CreateEvent
│  ├─ calendarId
│  ├─ summary
│  ├─ startAt + timeZone
│  ├─ endAt + timeZone
│  └─ location optional
│
├─ success
│  └─ Repository.UpdateSyncResult
│     ├─ calendarSyncStatus=calendar_synced
│     ├─ googleCalendarId
│     ├─ googleCalendarEventId
│     ├─ googleCalendarHtmlLink
│     └─ calendarSyncedAt
│
└─ failure
   └─ Repository.UpdateSyncResult
      ├─ calendarSyncStatus=calendar_sync_failed
      └─ calendarSyncError
```

Rules:
- Firestore create success is preserved even when Google Calendar sync fails.
- Google Calendar sync result must be visible in Firestore and API response.
- Google adapter is replaceable; calendar usecase depends on interface, not SDK types.

## Task Type Contract

```text
supportedTaskTypes
└─ calendar
```

```text
LineTaskConsultRequest
├─ appId
├─ builderId
├─ messageText
├─ referenceTime
├─ timeZone
├─ supportedTaskTypes[]
└─ clientIp

LineTaskConsultResponse
├─ taskType
├─ operation
├─ summary
├─ startAt
├─ endAt
├─ location
└─ missingFields[]
```

Rules:
- LineBot Backend decides which task types are available.
- Internal decides which one the message belongs to.
- `task` dispatches by returned `taskType`.

## Firestore Contract

```text
calendar_tasks/{taskId}
├─ taskId
├─ source
├─ rawText
├─ taskType
├─ operation
├─ summary
├─ startAt
├─ endAt
├─ location
├─ missingFields
├─ status
├─ calendarSyncStatus
├─ googleCalendarId
├─ googleCalendarEventId
├─ googleCalendarHtmlLink
├─ calendarSyncError
├─ calendarSyncedAt
├─ internalAppId
├─ internalBuilderId
├─ internalRequest
├─ internalResponse
├─ createdAt
└─ updatedAt
```

Rules:
- `source=rest` in first version.
- `taskType=calendar` in first version.
- `location` may be empty.
- `startAt` and `endAt` are stored separately.
- `calendarSyncStatus` values:
  - `not_enabled`
  - `calendar_sync_pending`
  - `calendar_synced`
  - `calendar_sync_failed`

## Future Extension Shape

```text
internal/
├─ task
│  └─ router / factory can be extracted when second feature lands
├─ calendar
└─ new_feature
   ├─ usecase.go
   ├─ service.go
   ├─ repository.go optional
   └─ model.go
```

`task` remains the dispatch layer. Feature modules own their own usecase / service / repository.
