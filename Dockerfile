FROM golang:1.26.1-trixie AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o ksef-integration ./cmd
FROM debian:trixie-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /app/ksef-integration .
RUN mkdir -p /app/data
EXPOSE 8080
CMD ["./ksef-integration"]
