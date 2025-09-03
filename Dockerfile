FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main ./cmd/api

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary
COPY --from=builder /app/main .

# Copy the site directory for static files
COPY --from=builder /app/site ./site

# Copy any env files
COPY --from=builder /app/.env* ./

# Set default port
ARG PORT=8080
ENV PORT=${PORT}

EXPOSE ${PORT}

CMD ["./main"]
