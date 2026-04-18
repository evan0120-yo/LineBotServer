# LineBot Backend — Cloud Run Deployment Detail

---

# BLOCK 1: 部署目標與前提

**部署目標：**
- 把 LineBot Go backend 作為 Cloud Run 服務跑起來
- 接收 LINE Platform webhook，橋接 InternalAICopliot gRPC
- 生產環境 gRPC 走 TLS（`LINEBOT_INTERNAL_GRPC_INSECURE=false`）
- 不對外公開，LINE Platform 透過 channel secret 驗證身份

**前提條件：**
```text
prerequisites
├─ InternalAICopliot 已成功部署並取得 Service URL       ← 最重要
├─ gcloud SDK 已安裝且 auth login 完成
├─ docker 已安裝
├─ Artifact Registry repo 已存在 (docker-repo)
├─ Service Account 已建立
│  └─ linebot-backend@dailo-467502.iam.gserviceaccount.com
├─ IAM 綁定完成
│  └─ linebot-backend SA 取得 roles/run.invoker on internal-ai-copilot
└─ LINE Developer Console 設定
   ├─ Channel Secret 取得
   ├─ Channel Access Token 取得
   └─ Bot User ID 取得
```

---

# BLOCK 2: 完整部署流程圖

```text
前置: InternalAICopliot 已上線
  Service URL: https://internal-ai-copilot-XXXXX-de.a.run.app
  gRPC addr:   internal-ai-copilot-XXXXX-de.a.run.app:443
      │
      ▼
┌─────────────────────────────────────────────────┐
│  Step 1: Build Docker Image                     │
│                                                 │
│  docker build \                                 │
│    -f deployee/cloud_run/Dockerfile \           │
│    -t [REGISTRY]/linebot-backend:latest .       │
│                                                 │
│  Build stages:                                  │
│  golang:1.25-alpine (builder)                   │
│  └─ go mod download                             │
│  └─ go build → /app/linebot-server              │
│  alpine:3.21 (runtime)                          │
│  └─ COPY binary                                 │
│  └─ apk add ca-certificates tzdata             │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│  Step 2: Push to Artifact Registry              │
│                                                 │
│  docker push [REGISTRY]/linebot-backend:latest  │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│  Step 3: Deploy to Cloud Run                    │
│                                                 │
│  關鍵 env vars:                                  │
│  LINEBOT_INTERNAL_GRPC_ADDR=[URL]:443           │
│  LINEBOT_INTERNAL_GRPC_INSECURE=false           │
│  LINEBOT_LINE_CHANNEL_SECRET=xxx                │
│  LINEBOT_LINE_CHANNEL_ACCESS_TOKEN=xxx          │
│  LINEBOT_LINE_BOT_USER_ID=xxx                   │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
        Cloud Run 服務上線
        取得 Service URL
        格式: https://linebot-backend-XXXXX-de.a.run.app
                  │
                  ▼
        LINE Developer Console
        設定 Webhook URL:
        https://linebot-backend-XXXXX-de.a.run.app/api/line/webhook
```

## gRPC TLS 連線邏輯（對 InternalAICopliot）

```text
LineBot container 啟動
      │
      ▼
internalclient.NewService(grpcAddr, insecureConn)
      │
      ├─ insecureConn == true  (LINEBOT_INTERNAL_GRPC_INSECURE=true)
      │  └─▶ grpc.WithTransportCredentials(insecure.NewCredentials())
      │       └─ 用於 local dev，不做 TLS
      │
      └─ insecureConn == false (LINEBOT_INTERNAL_GRPC_INSECURE=false)
         └─▶ grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
              └─ 用於生產，系統根憑證做 TLS 握手
              └─ Cloud Run URL: [host]:443
```

## LINE Webhook 驗證流程

```text
POST /api/line/webhook
      │
      ▼
LineHandler.ServeHTTP
      │
      ├─ 驗 X-Line-Signature (HMAC-SHA256)
      │  ├─ 失敗 → 401
      │  └─ 成功 → 繼續
      │
      ▼
解析 LINE Events
      │
      ├─ source.UserID == LineBotUserID → 忽略（bot 自己的訊息）
      ├─ event.Type != "message"        → 忽略
      └─ message.Type != "text"         → 忽略
            │
            ▼
      gatekeeperUseCase.HandleLineTask
            │
            ▼
      task usecase → internalclient.LineTaskConsult (gRPC)
            │
            ▼
      lineReplyClient.Reply (LINE Messaging API)
```

---

# BLOCK 3: 技術補充

## 環境變數完整清單

| 變數名 | 必填 | 生產值 | 本機值 |
|--------|------|--------|--------|
| `PORT` | 自動注入 | `8080` | — |
| `LINEBOT_INTERNAL_GRPC_ADDR` | ✅ | `[internal URL]:443` | `localhost:9091` |
| `LINEBOT_INTERNAL_GRPC_INSECURE` | ✅ 生產改 | `false` | `true` |
| `LINEBOT_INTERNAL_APP_ID` | ✅ | `linebot-app` | `linebot-app` |
| `LINEBOT_INTERNAL_BUILDER_ID` | ✅ | `4` | `4` |
| `LINEBOT_LINE_CHANNEL_SECRET` | ✅ | LINE Console 取 | LINE Console 取 |
| `LINEBOT_LINE_CHANNEL_ACCESS_TOKEN` | ✅ | LINE Console 取 | LINE Console 取 |
| `LINEBOT_LINE_BOT_USER_ID` | ✅ | LINE Console 取 | LINE Console 取 |
| `LINEBOT_GOOGLE_CALENDAR_ENABLED` | 可選 | `true` / `false` | `false` |
| `LINEBOT_GOOGLE_CALENDAR_ID` | 啟用時必填 | calendar ID | — |
| `LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE` | 啟用時必填 | `/secrets/credentials/credentials.json` | 本機路徑 |
| `LINEBOT_GOOGLE_OAUTH_TOKEN_FILE` | 啟用時必填 | `/secrets/token/token.json` | 本機路徑 |

## Service Account 設定

```text
SA name: linebot-backend@dailo-467502.iam.gserviceaccount.com

IAM roles:
└─ roles/run.invoker on internal-ai-copilot (Cloud Run service level)
   └─ 允許 LineBot 呼叫 InternalAICopliot
   └─ 指令：
      gcloud run services add-iam-policy-binding internal-ai-copilot \
        --region asia-east1 \
        --member serviceAccount:linebot-backend@dailo-467502.iam.gserviceaccount.com \
        --role roles/run.invoker \
        --project dailo-467502
```

## Docker Image 結構

```text
Build stage (golang:1.25-alpine)
├─ WORKDIR /app
├─ COPY go.mod go.sum → go mod download
├─ COPY . .
└─ go build → /app/linebot-server

Runtime stage (alpine:3.21)
├─ apk add ca-certificates  ← gRPC TLS 握手需要
├─ apk add tzdata
├─ COPY binary from builder
├─ EXPOSE 8080
└─ CMD ["./linebot-server"]
```

> `ca-certificates` 對 LineBot 尤其重要：需要同時做 LINE API TLS 和 InternalAICopliot gRPC TLS。

## LINE Developer Console 設定步驟

```text
部署完成後
      │
      ▼
LINE Developer Console
      │
      ├─ Messaging API > Webhook settings
      │  └─ Webhook URL: https://[Cloud Run URL]/api/line/webhook
      │  └─ Use webhook: ON
      │
      └─ 驗證 Webhook（Verify 按鈕）
         └─ 預期: Success
```

## 已知限制

```text
限制
├─ linebot-backend 必須設 allUsers roles/run.invoker
│  └─ LINE Platform 無法帶 Google IAM token，安全邊界靠 HMAC channel secret
├─ Google Calendar OAuth token 需要先在本機跑一次 auth flow 產出 token.json
│  └─ token.json 需要想辦法帶入 Cloud Run（Secret Manager 或直接 env）
└─ gRPC timeout 預設 30 秒（Builder 處理較慢時可能 DeadlineExceeded）
```
