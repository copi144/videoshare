# syntax=docker/dockerfile:1
FROM golang:alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o videoserver ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg
WORKDIR /app
COPY --from=builder /build/videoserver .
EXPOSE 8080
VOLUME ["/app/data"]
ENV PORT=:8080 DATA_DIR=/app/data
ENTRYPOINT ["/app/videoserver"]
