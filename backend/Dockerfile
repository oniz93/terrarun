FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /api ./cmd/api
RUN go build -o /migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /api .
COPY --from=builder /migrate .
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./api"]
