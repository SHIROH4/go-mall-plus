# =============================================================================
# Cart RPC - Binary Runtime Image
# =============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY cart ./cart
COPY etc ./etc

EXPOSE 8083

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

ENTRYPOINT ["./cart"]
CMD ["-f", "etc/cart.yaml"]
