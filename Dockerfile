FROM golang:1.26.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /out/crm \
    ./cmd/crm

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S crm \
    && adduser -S -G crm crm \
    && mkdir -p /app/storage/uploads /app/storage/exports \
    && chown -R crm:crm /app

WORKDIR /app

COPY --from=builder /out/crm /app/crm
COPY migrations /app/migrations

USER crm

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O - http://127.0.0.1:8080/health/live || exit 1

ENTRYPOINT ["/app/crm"]
CMD ["api"]

