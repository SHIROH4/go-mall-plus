# =============================================================================
# Stock RPC - Binary Runtime Image
# =============================================================================
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY stock ./stock
COPY etc ./etc
EXPOSE 8086
ENTRYPOINT ["./stock"]
CMD ["-f", "etc/stock.yaml"]
