-- post_product.lua
-- Benchmarks POST /Products (entity creation throughput)
-- Each virtual thread generates unique product names to avoid conflicts.

local thread_id = 0
local request_count = 0

-- wrk calls setup() once per thread before the test starts
function setup(thread)
    thread:set("tid", thread_id)
    thread_id = thread_id + 1
end

function init(args)
    -- Retrieve the per-thread ID set in setup()
    local tid = tonumber(wrk.thread:get("tid")) or 0
    request_count = tid * 10000000
    math.randomseed(tid + os.time())
end

function request()
    request_count = request_count + 1

    local price       = math.random(100, 99900) / 100.0
    local category_id = math.random(1, 100)
    local status      = 1  -- InStock

    local body = string.format(
        '{"Name":"LoadTest-%d","Price":%.2f,"CategoryID":%d,"Status":%d}',
        request_count, price, category_id, status
    )

    local headers = {
        ["Content-Type"] = "application/json",
        ["Accept"]       = "application/json",
    }

    return wrk.format("POST", "/Products", headers, body)
end
