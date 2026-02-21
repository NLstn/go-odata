-- etag_conditional_get.lua
-- Benchmarks GET /Products(id) with an If-None-Match header.
-- Sends a deliberately non-matching ETag so the server always returns 200 with
-- the full entity body. This isolates the overhead of ETag comparison logic.
-- Cycles through products 1-1000.

local request_count = 0

function init(args)
    request_count = 0
end

function request()
    request_count = request_count + 1

    local product_id = (request_count % 1000) + 1
    local path       = string.format("/Products(%d)", product_id)

    -- Intentionally wrong ETag — server must compare and send full 200 response
    local headers = {
        ["Accept"]        = "application/json",
        ["If-None-Match"] = '"stale-etag-value"',
    }

    return wrk.format("GET", path, headers, nil)
end
