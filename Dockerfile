FROM golang:1.23-alpine AS builder

WORKDIR /aiproxy

COPY ./ ./

RUN apk add --no-cache curl

RUN sh common/tiktoken/assest.sh

RUN go install github.com/swaggo/swag/cmd/swag@latest

RUN swag init

RUN go build -trimpath -tags "jsoniter" -ldflags "-s -w" -o aiproxy

FROM alpine:latest

RUN mkdir -p /aiproxy

WORKDIR /aiproxy

VOLUME /aiproxy

RUN apk add --no-cache ca-certificates tzdata ffmpeg curl && \
    rm -rf /var/cache/apk/*

COPY --from=builder /aiproxy/aiproxy /usr/local/bin/aiproxy

ENV PUID=0 PGID=0 UMASK=022

ENV FFPROBE_ENABLED=true

EXPOSE 3000

ENTRYPOINT ["aiproxy"]
