# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X 'github.com/saviotito/currency-router/internal/version.Version=v0.2.0'" \
    -o /out/smartpath-fx \
    ./cmd/server/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/smartpath-fx /smartpath-fx
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/smartpath-fx"]
