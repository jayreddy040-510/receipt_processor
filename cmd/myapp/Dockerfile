# builder stage
FROM golang:1.21-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./cmd/myapp/

# main stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /main .

CMD ["./main"]
