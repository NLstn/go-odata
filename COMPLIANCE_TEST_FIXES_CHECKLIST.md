# Compliance Test Fixes - Actionable Checklist

## Critical Fixes (Must Do First)

### 1. Entity References Tests (`11.2.15_entity_references.go`)
**Problem:** Tests skip when $ref returns 404, but $ref is mandatory in OData v4

**Files to modify:**
- [ ] `tests/v4_0/11.2.15_entity_references.go`
- [ ] `tests/v4_0/11.4.6_relationships.go` (uses $ref)

**Changes needed:**
```go
// OLD (line ~58-60):
if resp.StatusCode == 404 {
    return ctx.Skip("$ref operations not implemented")
}

// NEW:
if resp.StatusCode != 200 {
    return fmt.Errorf("$ref is mandatory in OData v4, got status %d", resp.StatusCode)
}
```

**Tests to fix:**
- [ ] test_read_entity_reference (relationships.go line 45-68)
- [ ] test_read_collection_references (relationships.go line 70-94)
- [ ] test_create_entity_reference (relationships.go line 96+)
- [ ] Similar tests in entity_references.go

---

### 2. Location Header Test (`8.2.5_header_location.go`)
**Problem:** Test passes when Location header is missing, but spec requires it

**File to modify:**
- [ ] `tests/v4_0/8.2.5_header_location.go`

**Changes needed:**
```go
// OLD (line 38-43):
location := resp.Headers.Get("Location")
if location != "" && !strings.Contains(location, "Products") {
    return framework.NewError("Location should contain entity URL")
}
return nil  // Passes even if location is empty!

// NEW:
location := resp.Headers.Get("Location")
if location == "" {
    return fmt.Errorf("Location header is required for 201 responses per OData v4 spec")
}
if !strings.Contains(location, "Products") {
    return fmt.Errorf("Location header should contain entity URL, got: %s", location)
}
return nil
```

---

### 3. Type Casting Error Messages (`11.2.13_type_casting.go`)
**Problem:** Tests claim "specification violation" but derived types are optional

**File to modify:**
- [ ] `tests/v4_0/11.2.13_type_casting.go`

**Changes needed:**
```go
// OLD (multiple locations, e.g., line 44):
if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
    return fmt.Errorf("specification violation: isof function must be supported (status: %d)", resp.StatusCode)
}

// NEW:
if resp.StatusCode != 200 {
    return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
}
```

**Apply to all 9 type casting tests**

---

### 4. Empty Path Segments Test (`11.1_resource_path.go`)
**Problem:** Test skips when server returns 200 for invalid URL with empty segments

**File to modify:**
- [ ] `tests/v4_0/11.1_resource_path.go`

**Changes needed:**
```go
// OLD (line 396-398):
if resp.StatusCode == 200 {
    return ctx.Skip("server normalizes empty segments; behavior under review")
}

// NEW:
if resp.StatusCode == 200 {
    return fmt.Errorf("server accepted invalid URL with empty path segments (should return 400/404)")
}
```

---

## High Priority Fixes

### 5. Stream Properties Test Diagnostics (`11.2.12_stream_properties.go`)
**Problem:** Test failing with 500 error but doesn't show response details

**File to modify:**
- [ ] `tests/v4_0/11.2.12_stream_properties.go`

**Changes needed:**
```go
// At line 191, improve error message:
// OLD:
return fmt.Errorf("unexpected status: %d", resp.StatusCode)

// NEW:
return fmt.Errorf("unexpected status %d updating media entity. Response: %s", 
    resp.StatusCode, string(resp.Body))
```

---

### 6. Geospatial Tests Content Validation (`11.3.7_filter_geo_functions.go`)
**Problem:** Tests accept 200 without validating filter actually worked

**File to modify:**
- [ ] `tests/v4_0/11.3.7_filter_geo_functions.go`

**Changes needed:**
```go
// For all geo tests (lines 72-88, 91-105, 108-122, etc.):
// OLD:
case 200:
    return nil

// NEW:
case 200:
    var result map[string]interface{}
    if err := json.Unmarshal(resp.Body, &result); err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }
    if _, ok := result["value"]; !ok {
        return fmt.Errorf("response missing 'value' array")
    }
    return nil
```

**Apply to 6 geospatial tests**

---

### 7. Singleton Test Error Message (`11.2.16_singleton_operations.go`)
**Problem:** Claims singletons are mandatory but they're optional

**File to modify:**
- [ ] `tests/v4_0/11.2.16_singleton_operations.go`

**Changes needed:**
```go
// Find error messages claiming non-compliance for 404:
// OLD:
if resp.StatusCode == 404 {
    return fmt.Errorf("status code 404 indicates singleton property access is not implemented and is non-compliant")
}

// NEW:
if resp.StatusCode == 404 {
    return ctx.Skip("singleton properties not implemented (optional feature)")
}
```

---

## Medium Priority Improvements

### 8. Expand Tests Content Validation (`11.2.5.6_query_expand.go`)
**Problem:** Tests check if field is expanded but don't validate expanded content

**File to modify:**
- [ ] `tests/v4_0/11.2.5.6_query_expand.go`

**Enhancement needed:**
Add validation that expanded entities have proper structure:
```go
// After verifying expanded array exists, add:
if len(descArray) > 0 {
    desc, ok := descArray[0].(map[string]interface{})
    if !ok {
        return fmt.Errorf("expanded item is not an object")
    }
    if _, ok := desc["ID"]; !ok {
        return fmt.Errorf("expanded entity missing ID field")
    }
}
```

---

### 9. Deep Insert Result Verification (`11.4.7_deep_insert.go`)
**Problem:** Tests accept 201 but don't verify related entities were created

**File to modify:**
- [ ] `tests/v4_0/11.4.7_deep_insert.go`

**Enhancement needed:**
After successful deep insert:
1. Fetch the created entity with $expand
2. Verify related entities were created
3. Verify relationships are established

---

### 10. Navigation Property Operations ($ref usage)
**File to modify:**
- [ ] `tests/v4_0/11.4.6.1_navigation_property_operations.go`

**Problem:** Similar to Issue #1 - too lenient with 404 responses

**Changes needed:**
Review all tests that use $ref and ensure they fail (not skip) when mandatory features return 404.

---

## Documentation Improvements

### 11. Add Test Categories Documentation
**New file to create:**
- [ ] `compliance-suite/MANDATORY_VS_OPTIONAL.md`

**Content needed:**
Document which OData v4 features are:
- Mandatory (must fail if not implemented)
- Optional (should skip if not implemented)
- Conditional (mandatory if certain other features present)

### 12. Update README with Testing Philosophy
**File to modify:**
- [ ] `compliance-suite/README.md`

**Add section:**
```markdown
## Test Validation Levels

Our tests validate at three levels:

1. **Status Code:** HTTP response is correct
2. **Structure:** Response JSON/XML has correct structure
3. **Behavior:** Feature actually works per OData spec

All tests should aim for Level 3 validation.

## Mandatory vs Optional Features

Tests for mandatory OData v4 features should FAIL if not implemented.
Tests for optional features should SKIP if not implemented.
See MANDATORY_VS_OPTIONAL.md for details.
```

---

## Testing Checklist Summary

### By Priority:

**Critical (Do First):**
- [ ] Fix 6+ $ref tests to fail instead of skip
- [ ] Fix Location header test
- [ ] Fix type casting error messages (9 tests)
- [ ] Fix empty path segments test

**High Priority:**
- [ ] Add diagnostics to failing stream test
- [ ] Add content validation to geo tests (6 tests)
- [ ] Fix singleton error message

**Medium Priority:**
- [ ] Enhance expand test validation
- [ ] Add deep insert verification
- [ ] Review navigation property tests

**Documentation:**
- [ ] Create mandatory vs optional features doc
- [ ] Update README with testing philosophy

---

## Validation Checklist

After making each fix, verify:

1. [ ] Test still passes when feature works correctly
2. [ ] Test fails when feature doesn't work
3. [ ] Test skips only for truly optional features
4. [ ] Error messages are clear and accurate
5. [ ] Test validates behavior, not just status codes

---

## Testing the Fixes

To test your changes:

```bash
cd compliance-suite

# Run all tests
go run .

# Run specific test suite
go run . -pattern "entity_references"

# Run with verbose output
go run . -verbose

# Run with debug output
go run . -debug
```

Expected outcomes after fixes:
- Some previously passing tests will now fail (exposing library bugs)
- Some previously skipped tests will now fail (exposing library gaps)
- All tests will have clearer, more accurate error messages
- Pass rate may drop but will reflect true compliance more accurately

---

## Notes

- **Don't fix library bugs** - this checklist is only for TEST CODE fixes
- **Focus on test quality** - making tests stricter and more accurate
- **Some tests will fail** - that's expected and good! It exposes real issues
- **Document assumptions** - add comments explaining why features are mandatory/optional

---

## Estimated Impact

After implementing all critical fixes:
- Expected pass rate: ~85-90% (down from 96%)
- Failed tests will point to real library compliance gaps
- Skipped tests will only be truly optional features
- Test suite will be a reliable OData v4 compliance validator
