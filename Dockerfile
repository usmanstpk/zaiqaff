# --------------------------------------------------------------------------
# STAGE 1: BUILDER
# Purpose: Compile the Go application into a static binary
# --------------------------------------------------------------------------
# Use the correct, updated Go version
FROM golang:1.25-alpine AS builder

# Set CGO_ENABLED=0 for a fully static build, essential for running on a minimal image
ENV CGO_ENABLED=0
WORKDIR /app

# Install git (often needed for go mod to fetch dependencies)
RUN apk add --no-cache git

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
# -ldflags "-s -w" reduces binary size by omitting debug info and symbol table
RUN go build -ldflags "-s -w" -o pocketbase ./main.go


# --------------------------------------------------------------------------
# STAGE 2: RUNTIME
# Purpose: Create a small, secure, production-ready image (minimal size)
# --------------------------------------------------------------------------
FROM alpine:latest

# Install CA certificates for HTTPS (essential for external services) 
# and tzdata for correct timezone handling
RUN apk add --no-cache ca-certificates tzdata

# Set the working directory for PocketBase
WORKDIR /pb

# Copy the statically built PocketBase binary from the builder stage
COPY --from=builder /app/pocketbase /pb/pocketbase

# CRUCIAL: Copy PocketBase runtime assets
# These directories are needed for the 'serve' command to run successfully
# (e.g., Admin UI, database migrations, and hook scripts).
COPY ./pb_public /pb/pb_public
COPY ./migrations /pb/migrations
COPY ./hooks /pb/hooks

# Create the PocketBase data directory (will contain your SQLite database)
RUN mkdir -p /pb/pb_data

# Expose the application port
EXPOSE 8080

# Start PocketBase correctly, listening on all interfaces
CMD ["/pb/pocketbase", "serve", "--http=0.0.0.0:8080"]