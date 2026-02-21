-- conditional_patch.lua
-- Benchmarks PATCH /Products(id) with an If-Match: * conditional header.
-- If-Match: * succeeds as long as the entity exists (optimistic concurrency).
-- Measures the overhead of conditional update processing vs plain PATCH.
-- Cycles through product IDs 1-1000.

local request_count = 0

function init(args)
    math.randomseed(os.time())
    request_count = 0
end

function request()
    request_count = request_count + 1

    local product_id = (request_count % 1000) + 1
    local new_price  = math.random(100, 99900) / 100.0

    local body = string.format('{"Price":%.2f}', new_price)

    local headers = {
        ["Content-Type"] = "application/json",
        ["Accept"]       = "application/json",
        ["If-Match"]     = "*",
    }

    local path = string.format("/Products(%d)", product_id)
    return wrk.format("PATCH", path, headers, body)
end
