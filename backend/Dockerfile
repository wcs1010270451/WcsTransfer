FROM golang:1.24.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /wcstransfer-backend ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /wcstransfer-backend /usr/local/bin/wcstransfer-backend

EXPOSE 8080

CMD ["wcstransfer-backend"]
