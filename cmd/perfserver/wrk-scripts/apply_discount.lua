-- apply_discount.lua
-- Benchmarks POST /Products(id)/ApplyDiscount (bound OData action throughput)
-- Sends a discount percentage between 1-30%. Cycles through 1000 products.

local request_count = 0

function init(args)
    math.randomseed(os.time())
    request_count = 0
end

function request()
    request_count = request_count + 1

    local product_id = (request_count % 1000) + 1
    local discount   = math.random(1, 30)

    local body = string.format('{"percentage":%d}', discount)

    local headers = {
        ["Content-Type"] = "application/json",
        ["Accept"]       = "application/json",
    }

    local path = string.format("/Products(%d)/ApplyDiscount", product_id)
    return wrk.format("POST", path, headers, body)
end
