FROM golang:1.25.0-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o unshortened .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/unshortened .

# Copy static and template folders
COPY --from=builder /app/static ./static
COPY --from=builder /app/html ./html

# Create a non-root user and switch to it
RUN adduser -D appuser
USER appuser

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["./unshortened"]