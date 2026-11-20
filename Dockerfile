FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install --package-lock-only --ignore-scripts && npm ci --ignore-scripts
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/frontend/dist/ ./static/dist/
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/smoi ./cmd/smoi

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S -g 10001 smoi && adduser -S -D -H -u 10001 -G smoi smoi && mkdir -p /data && chown smoi:smoi /data
COPY --from=backend /out/smoi /usr/local/bin/smoi
USER smoi
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/smoi"]
