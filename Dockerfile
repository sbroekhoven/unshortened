FROM golang:1.27.3-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -trimpath -ldflags="-s -w" -o unshortened .

# Final stage
FROM alpine:3.24

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/unshortened .

# Copy template folder
COPY --from=builder /app/html ./html

# Create a non-root user and switch to it
RUN adduser -D appuser
USER appuser

# Expose the port the app runs on
ENV PORT=8080
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["./unshortened"]