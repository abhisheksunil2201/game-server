FROM golang:1.23.1 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go app
RUN go build -o main cmd/api/main.go

# Start a new stage from scratch
FROM alpine:latest  

# Set the working directory for the runtime
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Expose the port the app will run on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
