# =============================================================================
# Gateway - Binary Runtime Image
# =============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY gateway ./gateway
COPY etc ./etc
COPY cert ./cert

EXPOSE 8888

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8888/health || exit 1

ENTRYPOINT ["./gateway"]
CMD ["-f", "etc/gateway.yaml"]
