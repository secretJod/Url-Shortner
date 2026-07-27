# --- Build stage ---
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Alpine's musl libc needs libc6-compat and openssl for Prisma's engine binaries
RUN apk add --no-cache openssl

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

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
RUN apk --no-cache add ca-certificates openssl
WORKDIR /root/

COPY --from=builder /urlshortener .

EXPOSE 8080
CMD ["./urlshortener"]
