# LineBot Backend — Cloud Run Deployment Plan

---

# BLOCK 1: 這個服務是什麼

這個 Go backend 看起來像一個「**LINE 訊息入口 + 任務派送橋接器**」。
核心價值不是 AI，而是把 LINE 使用者的自然語言訊息橋接到 InternalAICopliot 做解析，再把結果回寫 Google Calendar。

它的依賴鏈：
- **上游**：LINE Platform（webhook push）
- **下游 AI**：InternalAICopliot gRPC（解析意圖）
- **下游日曆**：Google Calendar API（新增/查詢/刪除事件）

關鍵設計選擇：
- 純 HTTP server（不需要 h2c，自己不提供 gRPC）
- 對 InternalAICopliot 的呼叫走 gRPC + TLS（Cloud Run service-to-service）
- LINE Channel Secret / Access Token 透過環境變數注入
- 不允許未認證請求（`--no-allow-unauthenticated`），LINE Platform webhook 透過 channel secret 驗證

---

# BLOCK 2: Cloud Run 架構圖

```text
LINE Platform
      │  POST /api/line/webhook
      │  (HMAC-SHA256 X-Line-Signature)
      ▼
┌──────────────────────────────────────────────────────┐
│  Cloud Run: linebot-backend                          │
│                                                      │
│  PORT (8080, from Cloud Run env)                     │
│      │                                               │
│      ▼                                               │
│  httpMux                                             │
│  ├─ GET  /health                                     │
│  ├─ POST /api/tasks      (直接建立任務)               │
│  └─ POST /api/line/webhook (LINE webhook handler)   │
│                                                      │
│  gatekeeper → task usecase → internalclient         │
└────────┬───────────────────────────────┬─────────────┘
         │ gRPC + TLS (443)              │ HTTPS
         ▼                               ▼
┌─────────────────────┐     ┌──────────────────────────┐
│  InternalAICopliot  │     │  Google Calendar API     │
│  Cloud Run (gRPC)   │     │  (OAuth 2.0)             │
└─────────────────────┘     └──────────────────────────┘
```

**IAM 邊界：**
```text
linebot-backend SA
├─ roles/run.invoker on internal-ai-copilot   → 呼叫 InternalAICopliot
└─ (Google Calendar 走 OAuth token，不是 IAM)
```

**Artifact Registry 路徑：**
```text
asia-east1-docker.pkg.dev/dailo-467502/docker-repo/linebot-backend:latest
```

---

# BLOCK 3: 補充細節

## 部署順序依賴

```text
必須先完成
└─ InternalAICopliot 部署上線
   └─ 取得 Service URL
      └─ 作為 LINEBOT_INTERNAL_GRPC_ADDR
         格式: internal-ai-copilot-XXXXX-de.a.run.app:443
```

> `LINEBOT_INTERNAL_GRPC_INSECURE` 在 Cloud Run 部署時必須設為 `false`（預設 `true` 只給 local dev 用）。

## 必要環境變數

| 變數名 | 說明 | 範例 |
|--------|------|------|
| `PORT` | Cloud Run 自動注入 | `8080` |
| `LINEBOT_INTERNAL_GRPC_ADDR` | InternalAICopliot URL | `internal-ai-copilot-xxx.a.run.app:443` |
| `LINEBOT_INTERNAL_GRPC_INSECURE` | 生產設 false | `false` |
| `LINEBOT_LINE_CHANNEL_SECRET` | LINE channel secret | `abc123...` |
| `LINEBOT_LINE_CHANNEL_ACCESS_TOKEN` | LINE access token | `Bearer xxx...` |
| `LINEBOT_LINE_BOT_USER_ID` | Bot user ID | `U1234...` |
| `LINEBOT_GOOGLE_CALENDAR_ENABLED` | 是否啟用日曆 | `true` / `false` |

## Build → Push → Deploy 步驟

```text
Step 1: docker build
└─ context: LineBot/Backend/
└─ Dockerfile: deployee/cloud_run/Dockerfile
└─ tag: asia-east1-docker.pkg.dev/dailo-467502/docker-repo/linebot-backend:latest

Step 2: docker push

Step 3: gcloud run deploy
└─ --service-account linebot-backend@dailo-467502.iam.gserviceaccount.com
└─ --no-allow-unauthenticated
└─ --port 8080
```

## 注意事項

> Google Calendar OAuth credentials 如有使用，需透過 Secret Manager volume mount 或直接環境變數注入。
> LINE webhook URL 需在 LINE Developer Console 設定為 Cloud Run Service URL + `/api/line/webhook`。
> 第一次部署後記得測試 `/health` 確認服務正常。
