# Build context: LineBot/Backend/
# Usage: docker build -f deployee/cloud_run/Dockerfile .

FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o linebot-server ./cmd/api

# ---- runtime ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/linebot-server .

EXPOSE 8080
CMD ["./linebot-server"]
