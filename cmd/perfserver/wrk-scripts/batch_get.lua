-- batch_get.lua
-- Benchmarks POST /$batch with a JSON batch body containing 5 GET requests.
-- Tests the overhead of batch request parsing, dispatching, and response assembly.
-- Each batch cycles through different starting product IDs.

local request_count = 0
local BATCH_SIZE    = 5

function init(args)
    request_count = 0
end

function build_batch(start_id)
    local parts = {}
    for i = 0, BATCH_SIZE - 1 do
        local product_id = ((start_id + i - 1) % 1000) + 1
        table.insert(parts, string.format(
            '{"id":"%d","method":"GET","url":"Products(%d)"}',
            i + 1, product_id
        ))
    end
    return '{"requests":[' .. table.concat(parts, ",") .. "]}"
end

function request()
    request_count = request_count + 1

    local start_id = request_count * BATCH_SIZE
    local body     = build_batch(start_id)

    local headers = {
        ["Content-Type"] = "application/json",
        ["Accept"]       = "application/json",
        ["OData-Version"] = "4.0",
    }

    return wrk.format("POST", "/$batch", headers, body)
end
