# Multi-stage Dockerfile for binderlm CLI
# Stage 1: Build binary
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build args for version injection
ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_DATE=unknown

# Compile statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o /bin/binderlm ./cmd/binderlm

# Stage 2: Minimal runtime environment
FROM alpine:3.20

# Install runtime SSL certs and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Create non-privileged user and workspace
RUN adduser -D -u 10001 binderlm \
    && mkdir -p /workspace \
    && chown -R binderlm:binderlm /workspace

# Copy compiled binary from builder stage
COPY --from=builder /bin/binderlm /usr/local/bin/binderlm

WORKDIR /workspace
USER binderlm:binderlm

ENTRYPOINT ["/usr/local/bin/binderlm"]
CMD ["--help"]
