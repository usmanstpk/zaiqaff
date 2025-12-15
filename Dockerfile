# BUILD STAGE
# Use a modern Go image for building the binary
FROM golang:1.24-alpine AS builder

# Arguments passed by Coolify (these are for cache busting and environment config)
ARG FCM_SERVICE_ACCOUNT_JSON
ARG FIREBASE_PROJECT_ID
ARG GCLOUD_PROJECT
ARG GOOGLE_CLOUD_PROJECT
ARG COOLIFY_URL
ARG COOLIFY_FQDN
ARG COOLIFY_BRANCH
ARG COOLIFY_RESOURCE_UUID

# ⚡️ CRITICAL CACHE BUST STEP ⚡️
# We use this build arg which changes on every deploy to ensure 
# the 'go mod download' step is never served from a stale Docker cache.
# This fixes the problem of the compiler using an old PocketBase version.
ARG COOLIFY_BUILD_SECRETS_HASH

WORKDIR /app

# Copy go module files and download dependencies
# The ARGS above ensure this step and all subsequent steps are re-run on every deploy
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the Go binary (output file is /pb)
RUN CGO_ENABLED=0 go build -o /pb -ldflags "-s -w"

# RUNTIME STAGE
# Use a minimal Alpine image for the final, small runtime container
FROM alpine:3.18

# Copy build arguments to runtime stage to allow setting env vars later if needed
ARG FCM_SERVICE_ACCOUNT_JSON
ARG FIREBASE_PROJECT_ID
ARG GCLOUD_PROJECT
ARG GOOGLE_CLOUD_PROJECT
ARG COOLIFY_URL
ARG COOLIFY_FQDN
ARG COOLIFY_BRANCH
ARG COOLIFY_RESOURCE_UUID

# Install necessary packages for PocketBase
RUN apk add --no-cache ca-certificates sqlite-libs

# Set the working directory for the application data
WORKDIR /app

# Copy the built PocketBase binary to a safe, distinct path (/app/pb)
COPY --from=builder /pb /app/pb

# Create required directories for data, static files, and migrations
RUN mkdir -p pb_data pb_public migrations

# Make sure the binary is executable
RUN chmod +x /app/pb

# The port PocketBase will listen on
EXPOSE 8080

# The command to start the application
# It now points to the correct binary path: /app/pb
ENTRYPOINT ["/app/pb", "serve", "--http", "0.0.0.0:8080", "--dir", "pb_data"]