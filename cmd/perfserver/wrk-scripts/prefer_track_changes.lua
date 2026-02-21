-- prefer_track_changes.lua
-- Benchmarks GET /Products with the Prefer: odata.track-changes header.
-- The server should include a @odata.deltaLink in the response.
-- This measures the overhead of change-tracking setup on collection responses.

function init(args)
end

function request()
    local headers = {
        ["Accept"]  = "application/json",
        ["Prefer"]  = "odata.track-changes",
    }

    return wrk.format("GET", "/Products?$top=50", headers, nil)
end
