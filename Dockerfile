# ---------- Stage 1: Build ----------
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build main app binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o migrate ./cmd/migrate

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o worker ./cmd/worker

# Build seed binary from its main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o seed ./cmd/seed/main.go

# ---------- Stage 2: Run ----------
FROM alpine:latest

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN apk add --no-cache ca-certificates

# Copy binaries
COPY --from=builder /app/api .
COPY --from=builder /app/migrate .
COPY --from=builder /app/worker .
COPY --from=builder /app/seed ./scripts/seed   

# Copy migrations folder
COPY --from=builder /app/migrations ./migrations

USER appuser

EXPOSE 8000

CMD ["./api"]