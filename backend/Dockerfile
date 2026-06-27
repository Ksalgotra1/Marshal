FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN apk add --no-cache build-base
RUN CGO_ENABLED=1 GOOS=linux go build -o marshal ./cmd/marshal

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/marshal .
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./marshal"]
