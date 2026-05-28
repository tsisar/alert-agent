FROM golang:1.25-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w \
        -X github.com/tsisar/alert-agent/internal/version.Version=${VERSION} \
        -X github.com/tsisar/alert-agent/internal/version.Commit=${COMMIT} \
        -X github.com/tsisar/alert-agent/internal/version.Date=${DATE}" \
    -o /app/alert-agent ./cmd/alert-agent

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/alert-agent /usr/local/bin/alert-agent
COPY configs/ /etc/alert-agent/

ENTRYPOINT ["alert-agent"]
CMD ["serve"]
