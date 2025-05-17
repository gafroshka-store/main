FROM golang:1.23-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /app

COPY . .

RUN go mod download && go build -o /app/main ./cmd/main/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/main /app/main
COPY --from=builder /app/config/config.yaml /app/config/config.yaml
COPY --from=builder /app/db/init.sql /app/db/init.sql

WORKDIR /app

EXPOSE 8080

CMD ["./main"]