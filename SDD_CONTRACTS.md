# LineBot Backend SDD Contracts

這份文件是 SDD.md 的合約附錄。
面向需要對接欄位、查 config、實作 adapter 的時候才帶入。

> 系統拓樸與邊界牆請看 SDD.md。

---

## Internal Contract

### Request

```text
LineTaskConsultRequest
├─ appId
├─ builderId
├─ messageText
├─ referenceTime
├─ timeZone
├─ supportedTaskTypes[]   first version: ["calendar"]
└─ clientIp
```

### Response

```text
LineTaskConsultResponse
├─ taskType
├─ operation
├─ eventId
├─ summary
├─ startAt
├─ endAt
├─ queryStartAt
├─ queryEndAt
├─ location
└─ missingFields[]
```

### Rules

```text
create
└─ eventId may be empty string (actual id comes from Google Calendar)

query
└─ use queryStartAt / queryEndAt as primary condition

delete / update
└─ use eventId directly
```

## Google Calendar Contract

### Create

```text
input
├─ calendarId
├─ summary
├─ startAt
├─ endAt
├─ timeZone
└─ location (optional)

output
├─ eventId
├─ summary
├─ startAt
└─ endAt
```

### Query

```text
input
├─ calendarId
├─ queryStartAt
├─ queryEndAt
└─ timeZone

output events[]
├─ eventId
├─ summary
├─ startAt
├─ endAt
└─ location
```

### Delete

```text
input
├─ calendarId
└─ eventId
```

### Update

```text
input
├─ calendarId
├─ eventId
└─ summary (title only)
```

## Reply Formatting Contract

### Single event

```text
{eventId}
{summary}
{yyyy-MM-dd HH:mm (週X)} ~ {yyyy-MM-dd HH:mm (週X)}
```

### Multiple events

```text
{eventId}
{summary}
{time range}

{eventId}
{summary}
{time range}
```

### No data

```text
沒資料
```

### Delete success

```text
刪除成功
```

### Error

```text
{concise error message via infra.ErrorReplyMessage}
```

## LINE Webhook Design

```text
LINE webhook integration
├─ POST /api/line/webhook
├─ verify X-Line-Signature
├─ parse webhook events array
├─ filter: text message event only
├─ require: bot mention in message
├─ strip: mention text before passing downstream
└─ reuse: same task usecase as REST

rules
├─ group + private: both require mention (first version)
├─ mention: triggers gate only, not task classification
├─ operation dispatch: done by task layer after Internal response
└─ each event runs with own timeout context
```

## Config

```text
required env vars
├─ LINEBOT_INTERNAL_GRPC_ADDR
├─ LINEBOT_INTERNAL_APP_ID
├─ LINEBOT_INTERNAL_BUILDER_ID
├─ LINEBOT_GOOGLE_CALENDAR_ENABLED
├─ LINEBOT_GOOGLE_CALENDAR_ID
├─ LINEBOT_GOOGLE_CALENDAR_TIME_ZONE
├─ LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE
├─ LINEBOT_GOOGLE_OAUTH_TOKEN_FILE
├─ LINEBOT_LINE_CHANNEL_SECRET
├─ LINEBOT_LINE_CHANNEL_ACCESS_TOKEN
└─ LINEBOT_LINE_BOT_USER_ID

removed
└─ all Firestore-related config
```
