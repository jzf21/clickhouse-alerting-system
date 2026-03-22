FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /alerting-system .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /alerting-system /usr/local/bin/alerting-system
COPY config.example.yaml /etc/alerting-system/config.yaml
EXPOSE 8080
ENTRYPOINT ["alerting-system"]
CMD ["--config", "/etc/alerting-system/config.yaml"]
