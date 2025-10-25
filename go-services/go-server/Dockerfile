FROM golang:1.23-alpine as builder

RUN apk add --no-cache git

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/orchestrator /usr/local/bin/

EXPOSE 8081 50051
CMD ["orchestrator"]
