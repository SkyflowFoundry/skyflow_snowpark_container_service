# Use the official Golang image as a build stage
FROM golang:1.18-alpine AS build

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules files
COPY go.mod go.sum ./

# Download and cache the Go modules
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Use a minimal base image to package the built binary
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /root/

# Copy the binary from the build stage
COPY --from=build /app/main .

COPY credentials.json /root/credentials.json

# Expose the port the application runs on
EXPOSE 8080

# Command to run the application
CMD ["./main"]
