# =============================================================================
# Order DLQ Consumer - Binary Runtime Image
# =============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY order-dlq ./order-dlq
COPY etc ./etc

ENTRYPOINT ["./order-dlq"]
CMD ["-f", "etc/order.yaml"]
