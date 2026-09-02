-- Authenticated cart-write workload for wrk.
-- AUTH_TOKEN is injected by run_cart_write_benchmark.sh and is never printed.

local token = os.getenv("AUTH_TOKEN")
if not token or token == "" then
  error("AUTH_TOKEN is required")
end

local headers = {
  ["Content-Type"] = "application/json",
  ["Authorization"] = "Bearer " .. token,
}

local body = [[{
  "product_id": 8,
  "product_name": "benchmark-product",
  "price": 149900,
  "image_url": "",
  "count": 1
}]]

request = function()
  return wrk.format("POST", "/api/cart/add", headers, body)
end
