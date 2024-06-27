FROM golang:1.22-alpine

ENV CGO_ENABLED=1

RUN apk add --no-cache \
    gcc \
    musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Use go run with CGO enabled
CMD CGO_ENABLED=1 go run main.go

EXPOSE 8080
