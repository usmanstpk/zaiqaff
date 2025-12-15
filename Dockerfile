# BUILD STAGE
# Use a full Go environment to compile the application
FROM golang:1.22-alpine AS builder

# Set working directory for the build stage
WORKDIR /app

# Copy go module files and download dependencies
# This speeds up subsequent builds if dependencies haven't changed
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the Go binary
# -a: force rebuild packages that are already up-to-date
# -tags: include the "sqlite_omit_footer" build tag for static compilation (optional, but good practice)
# -o: output file name
RUN go build -o /pb -ldflags "-s -w"

# RUNTIME STAGE
# Use a minimal Alpine Linux image for the final container
FROM alpine:3.18

# Install necessary packages for PocketBase
# For a full install, we need ca-certificates and sqlite3
RUN apk add --no-cache ca-certificates sqlite-libs

# Set working directory for the application
WORKDIR /pb

# Copy the built PocketBase binary from the builder stage
COPY --from=builder /pb /pb

# Copy the default PocketBase UI and any other files your extension needs
# Create required directories
RUN mkdir -p pb_data pb_public migrations

# Make sure the binary is executable
RUN chmod +x /pb

# Expose the default port PocketBase runs on
EXPOSE 8080

# Run the PocketBase application
# The binary is executed directly to run the extension
ENTRYPOINT ["/pb", "serve", "--http", "0.0.0.0:8080", "--dir", "pb_data"]