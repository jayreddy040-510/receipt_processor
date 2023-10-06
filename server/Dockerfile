# builder stage
FROM golang:1.21-alpine as builder

WORKDIR /server

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# main stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /server/main .

CMD ["./main"]
