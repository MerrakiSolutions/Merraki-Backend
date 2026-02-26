# ---------- Stage 1: Build ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git (required for some Go deps)
RUN apk add --no-cache git

# Copy go mod files first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy entire project
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o main ./cmd/api


# ---------- Stage 2: Run ----------
FROM alpine:latest

WORKDIR /app

# Install certificates (important for HTTPS calls)
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/main .

# Expose your app port (change if needed)
EXPOSE 8000

# Run binary
CMD ["./main"]