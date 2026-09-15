# --- Frontend build stage (Phase 6) ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

# Copy package files first for better layer caching
COPY frontend/package*.json ./
RUN npm install

# Copy all frontend source and build
COPY frontend/ ./
RUN npm run build

# --- Backend build stage ---
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Alpine's musl libc needs libc6-compat and openssl for Prisma's engine binaries
RUN apk add --no-cache openssl

COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ ./

# IMPORTANT: generate the Prisma client HERE, inside the build container,
# not on your host machine. This ensures the query-engine binary that gets
# embedded matches the container's actual platform (Alpine/musl), not
# whatever OS you're developing on (e.g. macOS). This was the cause of the
# "ensure: no binary found" runtime error.
RUN go run github.com/steebchen/prisma-client-go prefetch
RUN go run github.com/steebchen/prisma-client-go generate

RUN CGO_ENABLED=0 GOOS=linux go build -o /urlshortener ./cmd/api

# --- Run stage ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates openssl wget && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /home/appuser/

COPY --from=builder /urlshortener .

# Copy frontend build (Phase 6)
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

RUN chown -R appuser:appgroup /home/appuser
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

CMD ["./urlshortener"]