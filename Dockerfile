# --------------------------------------------------------------------------
# STAGE 1: BUILDER
# This stage uses a full Go environment to compile the application.
# We use a recent Go image based on Alpine Linux for a smaller initial size.
# --------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

# Set the CGO_ENABLED flag to 0 for a static build, which works better with scratch.
ENV CGO_ENABLED=0

# Set the working directory for all subsequent commands
WORKDIR /app

# Copy the dependency files first (for better caching)
# If go.mod/go.sum don't change, Docker skips the time-consuming 'go mod download'.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code (including your main.go)
COPY . .

# Build the custom PocketBase executable
# -o pocketbase specifies the output file name.
RUN go build -ldflags "-s -w" -o pocketbase ./main.go


# --------------------------------------------------------------------------
# STAGE 2: FINAL
# This stage uses the 'scratch' image, which is the smallest possible base image
# (it's completely empty). It only contains the compiled binary for maximum security and minimal size.
# --------------------------------------------------------------------------
FROM scratch

# Set the working directory for the final container
WORKDIR /app

# Copy only the compiled binary from the 'builder' stage
COPY --from=builder /app/pocketbase /app/pocketbase

# IMPORTANT: Coolify expects the application to run the main binary
# This sets the command that executes when the container starts.
ENTRYPOINT ["/app/pocketbase"]

# Expose the standard PocketBase port for documentation (optional, but good practice)
EXPOSE 8080