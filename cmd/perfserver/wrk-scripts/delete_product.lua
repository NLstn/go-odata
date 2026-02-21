-- delete_product.lua
-- Benchmarks DELETE /Products(id) throughput.
-- Uses IDs in the range 5001-10000 (upper half of the seeded dataset).
-- First pass through each ID returns 204; subsequent passes return 404.
-- Both response paths are valid performance measurements.

local request_count = 0
local ID_MIN = 5001
local ID_MAX = 10000

function init(args)
    request_count = 0
end

function request()
    request_count = request_count + 1

    local product_id = ID_MIN + ((request_count - 1) % (ID_MAX - ID_MIN + 1))
    local path       = string.format("/Products(%d)", product_id)

    local headers = {
        ["Accept"] = "application/json",
    }

    return wrk.format("DELETE", path, headers, nil)
end
