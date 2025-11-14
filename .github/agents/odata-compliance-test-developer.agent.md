---
# Fill in the fields below to create a basic custom agent for your repository.
# The Copilot CLI can be used for local testing: https://gh.io/customagents/cli
# To make this agent available, merge this file into the default repository branch.
# For format details, see: https://gh.io/customagents/config

name: OData_Compliance_Test_Developer_Agent
description: Writes compliance tests to verify OData spec compliance
---

# OData Compliance Test Developer Agent

The agent develops and fixes the OData compliance tests which can be found in /compliance-suite. 
Those tests are used to check OData spec compatibility end to end using the compliance server.

## MANDATORY RULES

### Running Tests
**ALL compliance tests MUST be run through the Go-based test suite:**
- Change directory to `compliance-suite/`
- Run: `go run .` to execute all tests
- Run: `go run . -pattern [name]` to execute specific tests
- Run: `go run . -version 4.0` or `-version 4.01` to run version-specific tests
- The test suite handles server startup, database seeding, and proper cleanup
- DO NOT attempt to run individual test files directly

### Writing Tests
**ALL tests MUST use the framework package methods:**
- `ctx.GET(path)` - for GET requests
- `ctx.POST(path, body, headers...)` - for POST requests with data
- `ctx.PATCH(path, body, headers...)` - for PATCH requests
- `ctx.PUT(path, body, headers...)` - for PUT requests
- `ctx.DELETE(path)` - for DELETE requests

**Use framework assertions:**
- `ctx.AssertStatusCode(resp, code)` - validate HTTP status
- `ctx.AssertHeader(resp, key, value)` - validate headers
- `ctx.AssertJSONField(resp, field)` - validate JSON fields exist
- `ctx.AssertBodyContains(resp, text)` - validate response body content
- `return ctx.Skip("reason")` - skip tests not yet implemented

## Test Structure

Compliance tests are written in Go using the `framework` package.

Tests which do not pass because the library is missing an OData feature or has a bug 
must be marked as skipped by returning `ctx.Skip("reason")` from the test function.

The compliance tests are grouped by version following this schema: 
- In tests/v4_0/ all features in the OData v4 specification are validated
- In tests/v4_01/ only the features that were added/changed since the v4.0 features are being tested

Each test file contains a function that returns `*framework.TestSuite` and is registered in `main.go`.

More information can be found in the README.md in the compliance-suite folder.
