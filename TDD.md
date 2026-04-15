# LineBot Backend TDD

## Test Strategy

本專案以 BDD scenario 對應測試。第一版測試重點放在 task flow、calendar validation、handler request boundary、Firestore mapping。

```text
test priority
├─ usecase tests
│  └─ lock end-to-end orchestration without real network
├─ service tests
│  └─ lock deterministic validation
├─ repository tests
│  └─ lock Firestore mapping
└─ handler tests
   └─ lock HTTP parse / response mapping
```

## Test Modules

```text
gatekeeper tests
├─ POST /api/tasks rejects missing text
├─ POST /api/tasks maps request to task usecase command
└─ POST /api/tasks returns task result envelope

task tests
├─ passes supportedTaskTypes=["calendar"] to internalclient
├─ dispatches taskType=calendar operation=create to calendar usecase
├─ rejects unsupported taskType
└─ rejects unsupported operation

calendar tests
├─ create succeeds with summary/startAt/endAt
├─ create allows empty location
├─ create rejects missing summary
├─ create rejects missing startAt
└─ create rejects missing endAt

repository tests
├─ creates calendar_tasks document
├─ stores rawText
├─ stores taskType
├─ stores startAt/endAt separately
└─ stores internal request/response snapshots
```

## First Test Cases

### gatekeeper

```text
TestCreateTaskRejectsMissingText
Given request body text is empty
When POST /api/tasks is called
Then response code is 400
And error code is TEXT_REQUIRED
And task usecase is not called
```

```text
TestCreateTaskMapsRequestToTaskUseCase
Given request body has text/referenceTime/timeZone
When POST /api/tasks is called
Then task usecase receives the same text/referenceTime/timeZone
```

### task

```text
TestTaskUseCaseSendsSupportedCalendarTaskType
Given task registry contains calendar
When CreateFromText calls internalclient
Then LineTaskConsult command includes supportedTaskTypes=["calendar"]
```

```text
TestTaskUseCaseDispatchesCalendarCreate
Given internalclient returns taskType=calendar and operation=create
When CreateFromText runs
Then calendar usecase Create is called
And result includes taskId
```

```text
TestTaskUseCaseRejectsUnsupportedTaskType
Given internalclient returns taskType=unknown
When CreateFromText runs
Then calendar usecase is not called
And error code is TASK_TYPE_UNSUPPORTED
```

```text
TestTaskUseCaseRejectsUnsupportedOperation
Given internalclient returns taskType=calendar and operation=update
When CreateFromText runs
Then calendar usecase is not called
And error code is OPERATION_UNSUPPORTED
```

### calendar

```text
TestCalendarCreateAllowsMissingLocation
Given extraction has summary/startAt/endAt and empty location
When calendar Create runs
Then repository Create is called
```

```text
TestCalendarCreateRejectsMissingRequiredFields
Given extraction is missing summary or startAt or endAt
When calendar Create runs
Then repository Create is not called
And error code is INTERNAL_EXTRACTION_INCOMPLETE
```

## Fake Dependencies

UseCase tests should use fakes instead of real network:

```text
task usecase test
├─ fake internalclient
└─ fake calendar usecase

calendar usecase test
├─ real calendar service
└─ fake calendar repository

gatekeeper handler test
└─ fake task usecase
```

Repository tests may use Firestore emulator once repository code exists.

## What Not To Test First

第一版不要先寫：
- real LINE webhook tests
- real Google Calendar tests
- live Internal AI Copilot tests
- Gemma parsing tests

這些屬於 integration / future scope。第一版先把 LineBot Backend 自己的 orchestration 和 persistence 邊界鎖住。

