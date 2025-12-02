FROM golang:1.24.2-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o benchmarking_go ./cmd/

# Use a smaller image for the final stage
FROM alpine:latest  

WORKDIR /root/

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /app/benchmarking_go .

# Command to run
ENTRYPOINT ["./benchmarking_go"]

# Default arguments (can be overridden at runtime)
CMD ["--help"]
