# --------------------------------------------------------------------------
# STAGE 1: BUILDER
# --------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

ENV CGO_ENABLED=0
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags "-s -w" -o pocketbase ./main.go


# --------------------------------------------------------------------------
# STAGE 2: RUNTIME
# --------------------------------------------------------------------------
FROM alpine:latest

# Install CA certificates for HTTPS (Firebase REQUIRED)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /pb

# Copy PocketBase binary
COPY --from=builder /app/pocketbase /pb/pocketbase

# Create PocketBase data directory
RUN mkdir -p /pb/pb_data

# Expose PocketBase port
EXPOSE 8080

# Start PocketBase correctly
CMD ["/pb/pocketbase", "serve", "--http=0.0.0.0:8080"]
