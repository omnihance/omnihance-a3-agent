# Frontend build stage
FROM node:22-alpine AS frontend-builder

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Set working directory
WORKDIR /app

# Copy frontend package files for dependency caching
COPY cmd/omnihance-a3-agent/omnihance-a3-agent-ui/package.json cmd/omnihance-a3-agent/omnihance-a3-agent-ui/pnpm-lock.yaml ./cmd/omnihance-a3-agent/omnihance-a3-agent-ui/

# Install frontend dependencies
WORKDIR /app/cmd/omnihance-a3-agent/omnihance-a3-agent-ui
RUN pnpm install --frozen-lockfile

# Copy frontend source code
WORKDIR /app
COPY cmd/omnihance-a3-agent/omnihance-a3-agent-ui ./cmd/omnihance-a3-agent/omnihance-a3-agent-ui

# Build frontend
WORKDIR /app/cmd/omnihance-a3-agent/omnihance-a3-agent-ui
RUN pnpm run build

# Go build stage
FROM golang:1.25-alpine AS builder

# Set necessary Go environment variables
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set working directory inside container
WORKDIR /app

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from frontend-builder stage
COPY --from=frontend-builder /app/cmd/omnihance-a3-agent/omnihance-a3-agent-ui/dist ./cmd/omnihance-a3-agent/omnihance-a3-agent-ui/dist

# Build the Go binary
RUN go build -ldflags="-w -s" -o omnihance-a3-agent ./cmd/omnihance-a3-agent

# Final stage
FROM alpine:3.19

# Set working directory in container
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/omnihance-a3-agent .

# Define port argument with default value
ARG PORT=8080

# Set PORT environment variable
ENV PORT=$PORT

# Set RUNNING_IN_DOCKER environment variable
ENV RUNNING_IN_DOCKER=true

# Expose the port
EXPOSE $PORT

# Run the application
ENTRYPOINT ["./omnihance-a3-agent"]
