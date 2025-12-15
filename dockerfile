# Step 1: Build Stage
FROM golang:1.25 AS builder

WORKDIR /app

# Copy go.mod and go.sum first (dependency caching)
COPY go.* ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the Go app
RUN go build -o server ./cmd

# Step 2: Run Stage
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy compiled binary
COPY --from=builder /app/server .

# Expose port (Render will set $PORT)
EXPOSE 8080

# Run server
CMD ["./server"]
