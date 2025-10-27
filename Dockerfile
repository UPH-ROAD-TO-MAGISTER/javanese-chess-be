# ────────────────────────────────
# Stage 1 — Build the Go binary
# ────────────────────────────────
FROM golang:1.24-alpine AS builder

# Enable Go modules and set working directory
WORKDIR /app

# Install build tools
RUN apk add --no-cache git

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary (static)
RUN go build -o /javanese-chess cmd/server/main.go


# ────────────────────────────────
# Stage 2 — Minimal runtime image
# ────────────────────────────────
FROM gcr.io/distroless/base-debian12

# Set working directory inside the container
WORKDIR /app

# Copy binary from builder
COPY --from=builder /javanese-chess /app/javanese-chess

# Copy swagger docs (static assets)
COPY --from=builder /app/docs /app/docs

# Copy any configuration files if needed
# COPY config.yml /app/config.yml

# Expose service port
EXPOSE 8080

# Run the app
ENTRYPOINT ["/app/javanese-chess"]
