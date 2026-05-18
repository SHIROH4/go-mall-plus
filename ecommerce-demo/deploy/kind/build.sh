#!/bin/bash
# =============================================================================
# Build Script: Compile Go binaries and build Docker images
# For Kind deployment (binary-only images)
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
KIND_DIR="$SCRIPT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Service list
SERVICES=("gateway" "user" "product" "cart" "order" "payment" "address" "stock")
ORDER_WORKERS=("order-delay" "order-cron" "order-dlq")

# Output directory
OUTPUT_DIR="$KIND_DIR/output"

# =============================================================================
# Build all Go binaries
# =============================================================================
build_binaries() {
    log_info "Building Go binaries..."

    mkdir -p "$OUTPUT_DIR"

    cd "$PROJECT_ROOT"

    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64

    # Gateway
    log_info "Building gateway..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/gateway" ./app/gateway/gateway.go

    # User
    log_info "Building user..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/user" ./app/user/user.go

    # Product
    log_info "Building product..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/product" ./app/product/product.go

    # Cart
    log_info "Building cart..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/cart" ./app/cart/cart.go

    # Order
    log_info "Building order..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/order" ./app/order/order.go

    # Order Workers
    log_info "Building order-delay..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/order-delay" ./app/order/cmd/delay/main.go

    log_info "Building order-cron..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/order-cron" ./app/order/cmd/cron/main.go

    log_info "Building order-dlq..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/order-dlq" ./app/order/cmd/dlq/main.go

    # Payment
    log_info "Building payment..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/payment" ./app/payment/payment.go

    # Address
    log_info "Building address..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/address" ./app/address/address.go

    # Stock
    log_info "Building stock..."
    go build -ldflags='-w -s' -o "$OUTPUT_DIR/stock" ./app/stock/stock.go

    log_info "All binaries built successfully!"
}

# =============================================================================
# Copy configs for Docker build
# =============================================================================
prepare_docker_context() {
    log_info "Preparing Docker build context..."

    local context_dir="$KIND_DIR/context"
    mkdir -p "$context_dir"

    # Copy binaries
    for svc in "${SERVICES[@]}" "${ORDER_WORKERS[@]}"; do
        if [ -f "$OUTPUT_DIR/$svc" ]; then
            cp "$OUTPUT_DIR/$svc" "$context_dir/"
        fi
    done

    # Copy configs
    for svc in "${SERVICES[@]}"; do
        if [ -d "$PROJECT_ROOT/app/$svc/etc" ]; then
            mkdir -p "$context_dir/etc"
            cp -r "$PROJECT_ROOT/app/$svc/etc" "$context_dir/etc/$svc"
        fi
    done

    # Copy gateway cert (if exists)
    if [ -d "$PROJECT_ROOT/deploy/cert" ]; then
        mkdir -p "$context_dir/cert"
        cp -r "$PROJECT_ROOT/deploy/cert"/* "$context_dir/cert/" 2>/dev/null || true
    fi

    # Copy order etc to order workers
    if [ -d "$context_dir/etc/order" ]; then
        cp "$context_dir/etc/order"/* "$context_dir/" 2>/dev/null || true
    fi

    log_info "Docker context prepared at: $context_dir"
}

# =============================================================================
# Build Docker images for Kind
# =============================================================================
build_docker_images() {
    log_info "Building Docker images for Kind..."

    local context_dir="$KIND_DIR/context"
    local docker_dir="$KIND_DIR/docker"

    for svc in "${SERVICES[@]}" "${ORDER_WORKERS[@]}"; do
        log_info "Building image: ecommerce-$svc:latest"

        # Create temporary context for each service
        local temp_dir=$(mktemp -d)
        cp "$context_dir/$svc" "$temp_dir/" 2>/dev/null || true

        # Copy service-specific etc config
        if [ -d "$context_dir/etc/$svc" ]; then
            mkdir -p "$temp_dir/etc"
            cp -r "$context_dir/etc/$svc"/* "$temp_dir/etc/"
        fi

        # For order workers, use the order etc
        if [[ "$svc" == order-* ]]; then
            mkdir -p "$temp_dir/etc"
            cp -r "$context_dir/etc/order"/* "$temp_dir/etc/"
        fi

        # Copy cert for gateway
        if [ "$svc" == "gateway" ] && [ -d "$context_dir/cert" ]; then
            mkdir -p "$temp_dir/cert"
            cp -r "$context_dir/cert"/* "$temp_dir/cert/"
        fi

        # Build image
        docker build -f "$docker_dir/${svc}.dockerfile" -t "ecommerce-$svc:latest" "$temp_dir"

        # Cleanup
        rm -rf "$temp_dir"

        log_info "Image built: ecommerce-$svc:latest"
    done

    # Load images into Kind cluster
    load_into_kind
}

# =============================================================================
# Load images into Kind cluster
# =============================================================================
load_into_kind() {
    log_info "Loading images into Kind cluster..."

    for svc in "${SERVICES[@]}" "${ORDER_WORKERS[@]}"; do
        log_info "Loading ecommerce-$svc:latest into Kind..."
        kind load docker-image "ecommerce-$svc:latest" --name ecommerce-cluster 2>/dev/null || \
            log_warn "Kind cluster not found or not running. Run deploy-kind.sh first."
    done

    log_info "Images loaded into Kind!"
}

# =============================================================================
# Full build pipeline
# =============================================================================
full_build() {
    log_info "Starting full build pipeline..."
    build_binaries
    prepare_docker_context
    build_docker_images
    log_info "Build pipeline completed!"
}

# =============================================================================
# Quick rebuild (skip Go compilation)
# =============================================================================
quick_build() {
    if [ ! -d "$OUTPUT_DIR" ] || [ -z "$(ls -A $OUTPUT_DIR 2>/dev/null)" ]; then
        log_error "No binaries found. Run './build.sh full' first."
        exit 1
    fi

    log_info "Quick rebuild (Docker only)..."
    prepare_docker_context
    build_docker_images
}

# =============================================================================
# Help
# =============================================================================
help() {
    cat << EOF

===============================================
  E-commerce Microservices - Build Script
===============================================

Usage: ./build.sh [command]

Commands:
  full         Full build: compile binaries + Docker images
  binaries     Build Go binaries only
  docker       Build Docker images only (requires binaries)
  quick        Quick rebuild Docker images
  load         Load images into Kind cluster
  clean        Clean build artifacts

===============================================

EOF
}

case "${1:-help}" in
    full)
        full_build
        ;;
    binaries)
        build_binaries
        ;;
    docker)
        prepare_docker_context
        build_docker_images
        ;;
    quick)
        quick_build
        ;;
    load)
        load_into_kind
        ;;
    clean)
        log_info "Cleaning build artifacts..."
        rm -rf "$KIND_DIR/output" "$KIND_DIR/context"
        log_info "Cleaned!"
        ;;
    help|*)
        help
        ;;
esac
