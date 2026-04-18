# LineBot Backend — Cloud Run Deployment Log

記錄每次部署的過程、結果、踩坑與修正。

---

## 部署狀態總覽

```text
前置依賴
└─ InternalAICopliot 已部署並取得 Service URL       ✅ Done

部署流程
├─ [x] Step 0: 前置準備
│  ├─ [x] 取得 InternalAICopliot Service URL
│  ├─ [x] linebot-backend SA 建立
│  ├─ [x] roles/run.invoker 綁定到 internal-ai-copilot
│  └─ [ ] LINE Developer Console 取得憑證（已帶入 env vars）
│
├─ [x] Step 1: docker build
├─ [x] Step 2: docker push
├─ [x] Step 3: gcloud run deploy
├─ [x] Step 4: 驗證 /health endpoint
└─ [x] Step 5: LINE Webhook URL 設定 + Verify
```

---

## 部署記錄

<!-- 每次部署在下方新增一個 section -->

---

### Deploy #1

**日期：** 2026-04-17
**部署人：** evan
**Image tag：** `latest`
**InternalAICopliot URL（依賴）：** https://internal-ai-copilot-368821702422.asia-east1.run.app

#### 執行步驟結果

```text
Step 0: 前置準備
├─ InternalAICopliot Service URL 取得   ✅  → internal-ai-copilot-368821702422.asia-east1.run.app:443
├─ linebot-backend SA 建立              ✅
└─ run.invoker IAM 綁定                 ✅

Step 1: docker build
├─ 指令：docker build -f deployee/cloud_run/Dockerfile -t asia-east1-docker.pkg.dev/dailo-467502/docker-repo/linebot-backend:latest .
├─ 結果：✅  (44.9s, FINISHED)
└─ 備註：alpine/golang layers 已 cache，比 InternalAICopliot 快

Step 2: docker push
├─ 指令：docker push asia-east1-docker.pkg.dev/dailo-467502/docker-repo/linebot-backend:latest
├─ 結果：✅
└─ digest: sha256:b0e901abc37169b73825db99857cd682a4560a5eb564a40e4478cb0bfc8ceff5

Step 3: gcloud run deploy
├─ 結果：✅  revision: linebot-backend-00001-ltl
├─ Service URL：https://linebot-backend-368821702422.asia-east1.run.app
├─ LINEBOT_INTERNAL_GRPC_ADDR 使用值：internal-ai-copilot-368821702422.asia-east1.run.app:443
└─ LINEBOT_INTERNAL_GRPC_INSECURE：false

Step 4: 驗證 /health
├─ curl https://.../health (無 token)  → 403（預期，IAM 擋）
├─ CMD: curl -H "Authorization: Bearer $(..."  → 401（CMD 不展開 $()）
├─ PowerShell: 同指令                  → ✅ 空 body = 200 OK
└─ 備註：/health 無 body，空輸出即成功

Step 5: LINE Webhook 設定
├─ 第一次 Verify → 403（URL 重複：/api/line/webhook/api/line/webhook）
├─ 修正 URL 後第二次 Verify → 403（--no-allow-unauthenticated 擋 LINE Platform）
├─ 修正：add-iam-policy-binding allUsers roles/run.invoker
└─ 第三次 Verify → ✅ Success
```

#### 遇到的問題

```text
問題 1
├─ 現象：curl $(gcloud auth print-identity-token) → 401
├─ 根因：Windows CMD 不支援 $() 展開，token 未被替換
└─ 修正：改用 PowerShell 執行

問題 2
├─ 現象：LINE Verify → 403，URL 顯示 /api/line/webhook/api/line/webhook
├─ 根因：Webhook URL 輸入時路徑貼了兩次
└─ 修正：Edit 改為正確 URL

問題 3
├─ 現象：URL 修正後 LINE Verify 仍 403
├─ 根因：Cloud Run --no-allow-unauthenticated 在 IAM 層擋掉 LINE Platform 的匿名請求
│        LINE Platform 無法帶 Google IAM token
├─ 修正：add-iam-policy-binding allUsers roles/run.invoker
└─ 說明：安全邊界由 LineHandler 的 HMAC-SHA256 channel secret 驗證負責，
         這是 LINE webhook 的標準做法（同 Stripe/GitHub webhook 模式）
```

#### 端到端測試

```text
測試場景 1: LINE 訊息 → 任務建立
└─ Webhook Verify ✅，端到端訊息測試待進行

測試場景 2: Google Calendar 事件新增
└─ 未啟用（LINEBOT_GOOGLE_CALENDAR_ENABLED 未設定）
```

#### 部署後記錄

```text
Service URL: https://linebot-backend-368821702422.asia-east1.run.app
LINE Webhook URL: https://linebot-backend-368821702422.asia-east1.run.app/api/line/webhook
Image digest: sha256:b0e901abc37169b73825db99857cd682a4560a5eb564a40e4478cb0bfc8ceff5
Revision: linebot-backend-00001-ltl
```

---

### Deploy #2

**日期：** 2026-04-17
**部署人：** evan
**Image tag：** `latest`
**修正原因：** 加入 Cloud Run service-to-service OIDC token（修 PermissionDenied）

#### 執行步驟結果

```text
Step 0: go mod tidy
└─ 結果：✅（無輸出，idtoken/grpcoauth 已在 google.golang.org/api + grpc 內）

Step 1: docker build
├─ 結果：✅  (46.6s)
└─ 備註：go.sum 有變動，builder layer cache miss；alpine layers 全 CACHED

Step 2: docker push
├─ 結果：✅
└─ digest: sha256:ff5678418bd80459d29f17d8eed8866cfc7e4cf81c906e63d076f50395e261a0

Step 3: gcloud run deploy
├─ 結果：✅  revision: linebot-backend-00002-d7g
└─ Service URL：https://linebot-backend-368821702422.asia-east1.run.app
```

#### 修正內容

```text
修正: internalclient/service.go
├─ 新增 google.golang.org/api/idtoken
├─ 新增 google.golang.org/grpc/credentials/oauth (grpcoauth)
└─ insecureConn == false 時
   ├─ 從 grpcAddr 取 host，組 audience = "https://" + host
   ├─ idtoken.NewTokenSource(ctx, audience) 取 OIDC token source
   └─ grpc.WithPerRPCCredentials(grpcoauth.TokenSource{...}) 掛到每個呼叫

說明：Cloud Run --no-allow-unauthenticated 要求 caller 帶 OIDC token
      只有 IAM 綁定 (roles/run.invoker) 不夠，還需要在 request 帶 Bearer token
```

#### 部署後記錄

```text
Service URL: https://linebot-backend-368821702422.asia-east1.run.app
Revision: linebot-backend-00002-d7g
Image digest: sha256:ff5678418bd80459d29f17d8eed8866cfc7e4cf81c906e63d076f50395e261a0
```

---

---

### Deploy #3

**日期：** 2026-04-17
**部署人：** evan
**Image tag：** `latest`（同 Deploy #2，無新 image）
**修正原因：** 掛入 Google Calendar OAuth credentials + 開啟 Calendar 功能

#### 執行步驟結果

```text
Step 0: 前置準備
├─ Secret Manager API 啟用                          ✅（首次啟用）
├─ gcloud secrets create linebot-google-credentials ✅
├─ gcloud secrets create linebot-google-token       ✅
├─ IAM secretAccessor 綁定（credentials）           ✅
└─ IAM secretAccessor 綁定（token）                 ✅

Step 1: gcloud run deploy（含 --set-secrets）
├─ 第一次嘗試 → ❌
│  └─ 錯誤：Cannot update secret at [/secrets/token.json]
│            because a different secret is already mounted
│            in the same directory.
│  └─ 根因：兩個 secret 不能掛在同一個父目錄 /secrets/
├─ 修正：改為 /secrets/credentials/ 和 /secrets/token/ 各自獨立目錄
└─ 第二次嘗試 → ✅  revision: linebot-backend-00003-tv8

Secret 掛載路徑：
├─ /secrets/credentials/credentials.json = linebot-google-credentials:latest
└─ /secrets/token/token.json             = linebot-google-token:latest
```

#### 新增環境變數

```text
LINEBOT_GOOGLE_CALENDAR_ENABLED=true
LINEBOT_GOOGLE_CALENDAR_ID=1f6bf7e594574a49f33d8bed52056beefa9da404c3e5459faf71ad3fd6bdcf0d@group.calendar.google.com
LINEBOT_GOOGLE_OAUTH_CREDENTIALS_FILE=/secrets/credentials/credentials.json
LINEBOT_GOOGLE_OAUTH_TOKEN_FILE=/secrets/token/token.json
```

#### 遇到的問題

```text
問題 1
├─ 現象：Deployment failed - Cannot update secret in same directory
├─ 根因：Cloud Run 限制每個父目錄只能掛一個 secret volume
└─ 修正：credentials 掛 /secrets/credentials/，token 掛 /secrets/token/
```

#### 部署後記錄

```text
Service URL: https://linebot-backend-368821702422.asia-east1.run.app
Revision: linebot-backend-00003-tv8
Google Calendar ID: 1f6bf7e594574a49f33d8bed52056beefa9da404c3e5459faf71ad3fd6bdcf0d@group.calendar.google.com
```

---

<!-- 複製上方 ### Deploy #N section 新增下一次部署 -->
