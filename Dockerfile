# Build stage for Go backend
FROM golang:1.21-alpine AS go-builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY server/go.mod server/go.sum ./server/
COPY pkg/api-types/go.mod ./pkg/api-types/

# Download dependencies
WORKDIR /build/server
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
WORKDIR /build
COPY server/ ./server/
COPY pkg/ ./pkg/

# Build the application
WORKDIR /build/server
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o supacontrol main.go

# Build stage for React frontend
FROM node:18-alpine AS ui-builder

WORKDIR /build

# Copy package files
COPY ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY ui/ ./

# Build the frontend
RUN npm run build

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary
COPY --from=go-builder /build/server/supacontrol .

# Copy migrations
COPY --from=go-builder /build/server/internal/db/migrations ./internal/db/migrations

# Copy the built UI
COPY --from=ui-builder /build/dist ../ui/dist

# Expose port
EXPOSE 8091

# Run the application
CMD ["./supacontrol"]
