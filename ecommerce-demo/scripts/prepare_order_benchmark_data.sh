#!/bin/bash
# Seed an isolated product with ample stock for local order-write benchmarks.
# Only run against the disposable Kind test namespace.

set -euo pipefail

PRODUCT_ID="${PRODUCT_ID:-900001}"
STOCK="${STOCK:-10000}"

mysql_password=$(kubectl get secret ecommerce-secrets -n ecommerce \
  -o jsonpath='{.data.mysql-password}' | base64 --decode)

sql="
INSERT INTO product (id, name, \`desc\`, price, image_url, category_id)
VALUES (${PRODUCT_ID}, 'order-benchmark-product', 'local benchmark only', 100, '', 1)
ON DUPLICATE KEY UPDATE name=VALUES(name), \`desc\`=VALUES(\`desc\`), price=VALUES(price);
INSERT INTO stock (product_id, stock_num, version)
VALUES (${PRODUCT_ID}, ${STOCK}, 0)
ON DUPLICATE KEY UPDATE stock_num=VALUES(stock_num), version=0;
"

kubectl exec -i -n ecommerce mysql-0 -- \
  mysql -uroot -p"$mysql_password" ecommerce_demo -e "$sql" >/dev/null

kubectl exec -n ecommerce redis-0 -- \
  redis-cli -a redis123456 --no-auth-warning -c SET "{stock}:${PRODUCT_ID}" "$STOCK" >/dev/null

echo "Prepared benchmark product ${PRODUCT_ID} with stock ${STOCK} in MySQL and Redis."
