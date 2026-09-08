FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY gateway/go.mod gateway/go.sum ./
RUN go mod download
COPY gateway/ .
RUN go build -o omni-router main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/omni-router .
COPY --from=builder /app/config ./config
EXPOSE 8787
CMD ["./omni-router"]
