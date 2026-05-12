# =============================================================================
# Order Cron Job - Binary Runtime Image
# =============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY order-cron ./order-cron
COPY etc ./etc

ENTRYPOINT ["./order-cron"]
CMD ["-f", "etc/order.yaml"]
