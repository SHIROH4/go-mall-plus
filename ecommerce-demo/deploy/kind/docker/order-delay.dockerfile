# =============================================================================
# Order Delay Consumer - Binary Runtime Image
# =============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY order-delay ./order-delay
COPY etc ./etc

ENTRYPOINT ["./order-delay"]
CMD ["-f", "etc/order.yaml"]
