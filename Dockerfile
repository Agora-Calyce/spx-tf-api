# Build stage
FROM golang:1.25-alpine AS builder

# Install the required dependencies
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy the dependency files
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# CGO_ENABLED=0 for a static binary compatible with alpine
RUN CGO_ENABLED=0 GOOS=linux go build -v -o spx-tf-api ./cmd/spx-tf-api/main.go

# Final stage (lightweight image)
FROM alpine:3.24.1

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the binary from the build stage
COPY --from=builder /app/spx-tf-api .

# Port exposed by default by the Echo application
EXPOSE 8080

# Default environment variable (can be overridden at runtime)
# The user requires SPX_API_ENDPOINT to be set at runtime.
# The default value can be omitted or set to the one from the code.
ENV PORT=8080

# Startup command
CMD ["./spx-tf-api"]
