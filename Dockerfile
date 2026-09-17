FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -o lattepos ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Jakarta

COPY --from=builder /app/lattepos .

EXPOSE 8000

ENTRYPOINT ["./lattepos"]