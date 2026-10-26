# syntax=docker/dockerfile:1
FROM golang:1.27-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/gestion-immo ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /out/gestion-immo /app/gestion-immo
COPY migrations /app/migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/gestion-immo"]
CMD ["server"]
