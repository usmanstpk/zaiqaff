# BUILD STAGE
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the Go binary (output file is /pb)
RUN go build -o /pb -ldflags "-s -w"

# RUNTIME STAGE
FROM alpine:3.18

# Install necessary packages for PocketBase
RUN apk add --no-cache ca-certificates sqlite-libs

# FIX 1: Use a safe working directory that won't conflict with the binary path
WORKDIR /app

# FIX 2: Copy the built PocketBase binary to a safe, distinct path (/app/pb)
COPY --from=builder /pb /app/pb

# Create required directories (now inside /app)
RUN mkdir -p pb_data pb_public migrations

# FIX 3: Make sure the binary is executable
RUN chmod +x /app/pb

EXPOSE 8080

# FIX 4: The ENTRYPOINT must now point to the correct path: /app/pb
ENTRYPOINT ["/app/pb", "serve", "--http", "0.0.0.0:8080", "--dir", "pb_data"]