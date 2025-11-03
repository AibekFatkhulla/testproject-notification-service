FROM golang:1.23-bullseye AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    librdkafka-dev pkg-config ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY notification-service/go.mod notification-service/go.sum ./
RUN go mod download
COPY notification-service/ ./
RUN go build -o /bin/notification-service

FROM debian:bullseye-slim AS runner
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates librdkafka1 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /bin/notification-service /bin/notification-service
COPY --from=builder /app/db /app/db
ENV KAFKA_BOOTSTRAP_SERVERS=kafka:9092
CMD ["/bin/notification-service"]


