-- patch_product.lua
-- Benchmarks PATCH /Products(id) (partial update throughput)
-- Cycles through product IDs 1-1000 so every request hits an existing entity.

local request_count = 0

function init(args)
    math.randomseed(os.time())
    request_count = 0
end

function request()
    request_count = request_count + 1

    -- Cycle through the first 1000 products
    local product_id = (request_count % 1000) + 1
    local new_price  = math.random(100, 99900) / 100.0

    local body = string.format('{"Price":%.2f}', new_price)

    local headers = {
        ["Content-Type"] = "application/json",
        ["Accept"]       = "application/json",
    }

    local path = string.format("/Products(%d)", product_id)
    return wrk.format("PATCH", path, headers, body)
end
