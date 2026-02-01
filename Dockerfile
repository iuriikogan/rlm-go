# Use Go for the build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app
COPY go.mod ./
# If go.sum exists, copy it too
# COPY go.sum ./
RUN go mod download

COPY . .
RUN go build -o rlm-server main.go

# Use a slim Debian image for the final stage to include Python
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y python3 python3-pip && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/rlm-server .

# Expose port 8080
EXPOSE 8080

# Command to run the service
CMD ["./rlm-server"]
