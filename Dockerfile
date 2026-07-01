# Build Stage
FROM golang:1.25-alpine3.21 AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all services
RUN go build -o /bin/auth-service ./services/auth-service
RUN go build -o /bin/api-gateway ./services/api-gateway
RUN go build -o /bin/ai-gateway ./services/ai-gateway
RUN go build -o /bin/billing-service ./services/billing-service
RUN go build -o /bin/notification-service ./services/notification-service
RUN go build -o /bin/subscription-worker ./services/subscription-worker
RUN go build -o /bin/umkm-accounting ./apps/umkm/accounting
RUN go build -o /bin/umkm-business ./apps/umkm/business
RUN go build -o /bin/umkm-chatbot ./apps/umkm/chatbot
RUN go build -o /bin/umkm-automation ./apps/umkm/automation
RUN go build -o /bin/campaign-api ./apps/campaign/api
RUN go build -o /bin/wa-gateway ./services/wa-gateway
RUN go build -o /bin/wa-cloud-api ./services/wa-cloud-api
# Final Stage
FROM alpine:3.21

WORKDIR /app

# ponytail: ca-certificates for TLS to external APIs (Meta, Xendit, Telegram, LLM);
# wget for docker-compose healthchecks (test: ["CMD","wget",...]).
RUN apk add --no-cache ca-certificates wget

# Copy binaries from builder
COPY --from=builder /bin/auth-service /usr/local/bin/
COPY --from=builder /bin/api-gateway /usr/local/bin/
COPY --from=builder /bin/ai-gateway /usr/local/bin/
COPY --from=builder /bin/billing-service /usr/local/bin/
COPY --from=builder /bin/notification-service /usr/local/bin/
COPY --from=builder /bin/subscription-worker /usr/local/bin/
COPY --from=builder /bin/umkm-accounting /usr/local/bin/
COPY --from=builder /bin/umkm-business /usr/local/bin/
COPY --from=builder /bin/umkm-chatbot /usr/local/bin/
COPY --from=builder /bin/umkm-automation /usr/local/bin/
COPY --from=builder /bin/campaign-api /usr/local/bin/
COPY --from=builder /bin/wa-gateway /usr/local/bin/
COPY --from=builder /bin/wa-cloud-api /usr/local/bin/

# Default entrypoint (can be overridden by docker-compose)
COPY --from=builder /app/shared/migrations /app/shared/migrations
CMD ["sh"]
