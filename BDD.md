# LineBot Backend BDD

## Behavior World

```text
behavior world
├─ entry
│  ├─ REST: POST /api/tasks
│  └─ LINE: POST /api/line/webhook
├─ interpretation source
│  └─ Internal AI Copilot (gRPC LineTaskConsult only)
├─ execution target
│  └─ Google Calendar (only, no Firestore)
├─ operation space
│  ├─ create
│  ├─ query
│  ├─ delete
│  └─ update
└─ output space
   ├─ single event → 3-line block
   ├─ multiple events → repeated 3-line blocks
   ├─ no data → 沒資料
   └─ error → concise text only
```

## Hard Behavior Rules

```text
reply contract
├─ shape: 3 lines per event
│  ├─ line 1: raw eventId (no prefix, no "eventId:")
│  ├─ line 2: summary
│  └─ line 3: yyyy-MM-dd HH:mm (週X) ~ yyyy-MM-dd HH:mm (週X)
├─ multi: repeat 3-line block, blank line between events
├─ empty: 沒資料
└─ error: concise text only, no log, no stack trace

create contract
├─ required: summary + startAt + endAt
├─ optional: location (empty string ok, not a failure)
├─ missing required → reject, must not call Google Calendar
└─ eventId: from Google Calendar response only, not from Internal

query contract
├─ primary condition: time range only (queryStartAt + queryEndAt)
├─ no keyword / title search as main condition
└─ overlap rule: eventStart <= queryEnd AND eventEnd >= queryStart

delete contract
├─ direct by eventId only
└─ must not search by title or time range

update contract
├─ direct by eventId + new summary only
└─ must not change original startAt / endAt

webhook contract
├─ no mention → ignore (no Internal call, no Calendar call, no reply)
├─ invalid signature → reject immediately, no downstream call
└─ mention → strip mention text → same task usecase as REST
```

## Edge Scenarios

### Scenario: query overlap boundary touch

Given an event exists: 12:00 ~ 15:00  
When query range is 11:30 ~ 12:00  
Then the event must appear in results

> eventEnd (15:00) >= queryStart (12:00) → true. Boundary-equal counts as overlap.

### Scenario: unsupported taskType or operation

Given Internal returns taskType or operation not in the supported set  
When LineBot Backend processes the result  
Then no Google Calendar operation must be executed  
And reply must be concise error text  
And no exception or unhandled panic
