# syntax=docker/dockerfile:1
FROM node:22-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /build
RUN apk add --no-cache gcc musl-dev
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY --from=frontend-builder /build/frontend/dist/index.html backend/web/spa/index.html
COPY --from=frontend-builder /build/frontend/dist/favicon.svg backend/web/spa/favicon.svg
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o videoserver .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg
WORKDIR /app
COPY --from=backend-builder /build/videoserver .
EXPOSE 8080
VOLUME ["/app/data"]
ENV PORT=:8080 DATA_DIR=/app/data
ENTRYPOINT ["/app/videoserver"]
