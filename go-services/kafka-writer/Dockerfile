
FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o kafka-writer .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/kafka-writer .

EXPOSE 50052

CMD ["./kafka-writer"]