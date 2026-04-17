# LineBot Backend SDD

## Target Topology

```text
┌──────────────────────────────────────┐
│  transport boundary                  │
│  REST  POST /api/tasks               │
│  LINE  POST /api/line/webhook        │
└─────────────────┬────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────┐
│  gatekeeper                          │
│  parse / validate / verify sig       │
│  strip mention text                  │
└─────────────────┬────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────┐
│  task usecase                        │
│  build Internal request              │
│  validate taskType + operation       │
│  dispatch to calendar operation      │
└────────┬─────────────────┬───────────┘
         │                 │
         ▼                 ▼
┌────────────────┐  ┌──────────────────┐
│ internalclient │  │ calendar module  │
│ gRPC wrapper   │  │ create / query   │
│ hides protobuf │  │ delete / update  │
└────────────────┘  └────────┬─────────┘
                             │
                             ▼
                   ┌──────────────────┐
                   │  infra           │
                   │  Google Calendar │
                   │  API adapter     │
                   └──────────────────┘
```

Target: Calendar-only. No Firestore in any path.

## Boundary Walls + Runtime Skeleton

### Boundary Walls

```text
must not cross
├─ handler → Google SDK directly
├─ calendar → task layer
├─ internalclient → calendar
└─ gatekeeper → operation dispatch (belongs to task)

allowed direction
├─ gatekeeper → task
├─ task → internalclient
├─ task → calendar
├─ calendar → infra
└─ app → all (wiring only)
```

### Runtime Skeleton

```text
REST or LINE
└─ gatekeeper
   ├─ [LINE only] verify signature → filter text event → check mention → strip text
   └─ UseCase.CreateTask
      └─ task.UseCase.ExecuteFromText
         ├─ build LineTaskConsult request
         ├─ call internalclient.LineTaskConsult
         ├─ validate taskType = "calendar"
         ├─ validate operation
         └─ dispatch
            ├─ create → calendar.UseCase.Create
            ├─ query  → calendar.UseCase.Query
            ├─ delete → calendar.UseCase.Delete
            └─ update → calendar.UseCase.Update
```

### Package Map

```text
Backend/
├─ cmd/api          process entrypoint
└─ internal
   ├─ app           config / wiring / router
   ├─ gatekeeper    REST + LINE transport boundary
   ├─ task          Internal call + operation dispatch
   ├─ calendar      Google Calendar CRUD + formatter
   ├─ internalclient   Internal gRPC client wrapper
   └─ infra         config / errors / HTTP / Google Calendar adapter
```

> Field-level contracts, config list, LINE webhook detail → see SDD_CONTRACTS.md
