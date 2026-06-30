FROM golang:1.25-trixie AS builder
WORKDIR /app
RUN apt-get update && apt-get install -y \
    pkg-config \
    libxml2-dev \
    libxslt1-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ksef-integration ./cmd
FROM debian:trixie-slim
WORKDIR /app
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libxml2 \
    libxslt1.1
COPY --from=builder /app/ksef-integration .
COPY xsd ./xsd
COPY ui ./ui
RUN mkdir -p /app/data
EXPOSE 8080
CMD ["./ksef-integration"]
