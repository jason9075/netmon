# syntax=docker/dockerfile:1.6

########################################
# Builder
########################################
FROM golang:1.22-alpine AS builder
WORKDIR /src

# 安裝必要套件（可省略）
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod ./
RUN go mod download

COPY . .

# 針對樹莓派自動調整架構（若用 buildx，會傳入 TARGETARCH 與 TARGETVARIANT）
ARG TARGETARCH
ARG TARGETVARIANT
ENV CGO_ENABLED=0 GOOS=linux
# 當 TARGETARCH=arm 時，可用 GOARM
RUN if [ "$TARGETARCH" = "arm" ] && [ -n "$TARGETVARIANT" ]; then \
      export GOARM=${TARGETVARIANT#v}; \
      echo "Building for arm v$GOARM"; \
      go build -trimpath -ldflags='-s -w' -o /out/netmon ./cmd/netmon; \
    else \
      go build -trimpath -ldflags='-s -w' -o /out/netmon ./cmd/netmon; \
    fi

########################################
# Runner（distroless 更精簡也可）
########################################
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /out/netmon /app/netmon

# 預設環境變數
ENV URL=https://www.google.com/generate_204 \
    INTERVAL_SECONDS=10 \
    TIMEOUT_SECONDS=2 \
    LOG_PATH=/data/netlog.log

# 需要掛載 /data 做為持久化
VOLUME ["/data"]

ENTRYPOINT ["/app/netmon"]
