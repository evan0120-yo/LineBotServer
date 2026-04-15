# LineBot Backend SDD

## System Overview

LineBot Backend 是獨立 Go backend，負責接收自然語句、交給 Internal AI Copilot 解析，並依解析結果執行本地任務 module。

```text
LineBot Backend
├─ transport boundary
│  ├─ REST first version
│  └─ LINE webhook future
│
├─ task orchestration
│  ├─ calls Internal AI Copilot
│  ├─ receives taskType + operation
│  └─ dispatches to feature module
│
├─ feature modules
│  └─ calendar first version
│
└─ persistence
   └─ Firestore
```

## Package Architecture

```text
Backend/
├─ cmd/api
│  └─ process entrypoint
│
└─ internal
   ├─ app
   │  └─ config / store / module wiring
   │
   ├─ gatekeeper
   │  └─ REST and future LINE request boundary
   │
   ├─ task
   │  └─ Internal extraction + task dispatch
   │
   ├─ calendar
   │  └─ calendar task usecase / service / repository
   │
   ├─ internalclient
   │  └─ Internal AI Copilot gRPC client
   │
   └─ infra
      └─ config / errors / response / Firestore store
```

## Module Responsibilities

```text
app
├─ load infra.Config
├─ create Firestore store
├─ create internalclient service
├─ create calendar usecase
├─ create task usecase
├─ create gatekeeper handler
└─ expose HTTP handler

gatekeeper
├─ parse REST JSON request
├─ validate request boundary
├─ resolve client IP / source
├─ map API request to task command
└─ map result / error to HTTP response

task
├─ call Internal gRPC LineTaskConsult
├─ pass supportedTaskTypes
├─ validate taskType is supported
├─ validate operation is supported by first version
└─ dispatch calendar task to calendar module

calendar
├─ validate calendar create extraction
├─ require summary / startAt / endAt
├─ treat location as optional
├─ create calendar task record
└─ own calendar_tasks Firestore schema

internalclient
├─ build LineTaskConsult gRPC request
├─ map LineTaskConsult gRPC response
├─ hide protobuf details from task module
└─ does not own calendar rules

infra
├─ shared config
├─ business errors
├─ HTTP JSON envelope
├─ Firestore client / store
└─ runtime helpers
```

## Data Flow

```text
POST /api/tasks
│
▼
gatekeeper.Handler
├─ decode request
├─ require text
└─ call gatekeeper.UseCase
   │
   ▼
gatekeeper.UseCase
└─ call task.UseCase.CreateFromText
   │
   ▼
task.UseCase
├─ build Internal request
│  ├─ appId from config
│  ├─ builderId from config
│  ├─ messageText from text
│  ├─ referenceTime optional
│  ├─ timeZone optional
│  └─ supportedTaskTypes from task registry
│
├─ internalclient.Service.LineTaskConsult
│  │
│  ▼
│  Internal AI Copilot
│  └─ returns taskType / operation / event fields
│
├─ task.Service validate dispatch
│  ├─ taskType == calendar
│  └─ operation == create
│
└─ calendar.UseCase.Create
   ├─ calendar.Service.ValidateCreate
   └─ calendar.Repository.Create
      └─ Firestore calendar_tasks
```

## Task Type Contract

LineBot Backend uses a local task registry similar to Java enum semantics.

```go
type TaskType string

const (
	TaskTypeCalendar TaskType = "calendar"
)
```

First version registry:

```text
supportedTaskTypes
└─ calendar
```

Internal gRPC should receive the supported task list and return the selected task type:

```text
LineTaskConsultRequest
├─ supportedTaskTypes[]
└─ ...

LineTaskConsultResponse
├─ taskType
├─ operation
├─ summary
├─ startAt
├─ endAt
├─ location
└─ missingFields[]
```

Rule:
- LineBot Backend decides what task types are available.
- Internal decides which one the user message belongs to.
- LineBot Backend dispatches by `taskType`.

## Firestore Model

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
├─ internalAppId
├─ internalBuilderId
├─ internalRequest
├─ internalResponse
├─ createdAt
└─ updatedAt
```

Rules:
- `source=rest` in first version.
- future LINE webhook uses `source=line`.
- `taskType=calendar` for first version.
- `location` may be empty.
- `startAt` and `endAt` are stored as separate fields.

## Future Extension Model

When adding a second feature:

```text
internal/
├─ task
│  └─ add router / factory
├─ calendar
└─ new_feature
   ├─ usecase.go
   ├─ service.go
   ├─ repository.go optional
   └─ model.go
```

`task` remains the dispatch layer. Feature modules own their own usecase / service / repository.

