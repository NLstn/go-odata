# Compliance Test Patterns - Good vs Bad Examples

This document shows concrete examples of test patterns found in the compliance suite, categorizing them as good patterns to follow or bad patterns to avoid.

---

## Pattern 1: Status Code Validation

### ❌ BAD: Only checking status, not behavior
```go
// From multiple tests - just checks 200 OK
func testExample(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$filter=Price gt 100")
    if err != nil {
        return err
    }
    
    if resp.StatusCode == 200 {
        return nil  // ❌ Doesn't validate filter actually worked!
    }
    
    return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
```

### ✅ GOOD: Validating actual behavior
```go
// From 11.2.5.1_query_filter.go - validates filter worked
func testFilterGt(ctx *framework.TestContext) error {
    filter := url.QueryEscape("Price gt 100")
    resp, err := ctx.GET("/Products?$filter=" + filter)
    if err != nil {
        return err
    }
    
    if err := ctx.AssertStatusCode(resp, 200); err != nil {
        return err
    }
    
    var data map[string]interface{}
    if err := ctx.GetJSON(resp, &data); err != nil {
        return err
    }
    
    value, ok := data["value"].([]interface{})
    if !ok {
        return framework.NewError("value must be an array")
    }
    
    // ✅ Validates all returned entities match filter
    for _, item := range value {
        entity, ok := item.(map[string]interface{})
        if !ok {
            continue
        }
        
        price, ok := entity["Price"].(float64)
        if !ok {
            continue
        }
        
        if price <= 100 {
            return framework.NewError(fmt.Sprintf("Expected Price > 100, got Price=%f", price))
        }
    }
    
    return nil
}
```

**Why Good Pattern is Better:**
- Ensures filter actually filters data
- Catches bugs where server returns 200 but ignores filter
- Validates OData spec compliance, not just HTTP compliance

---

## Pattern 2: Handling Mandatory vs Optional Features

### ❌ BAD: Skipping mandatory features
```go
// From 11.4.6_relationships.go - $ref is mandatory!
func testReadEntityReference(ctx *framework.TestContext) error {
    resp, err := ctx.GET(productPath + "/Category/$ref")
    if err != nil {
        return err
    }
    
    if resp.StatusCode == 404 {
        return ctx.Skip("$ref operations not implemented")  // ❌ Should fail!
    }
    
    if err := ctx.AssertStatusCode(resp, 200); err != nil {
        return err
    }
    
    return ctx.AssertJSONField(resp, "@odata.id")
}
```

### ✅ GOOD: Failing on missing mandatory features
```go
func testReadEntityReference(ctx *framework.TestContext) error {
    resp, err := ctx.GET(productPath + "/Category/$ref")
    if err != nil {
        return err
    }
    
    // ✅ $ref is mandatory in OData v4
    if resp.StatusCode != 200 {
        return fmt.Errorf("$ref is mandatory in OData v4, got status %d", resp.StatusCode)
    }
    
    return ctx.AssertJSONField(resp, "@odata.id")
}
```

### ✅ GOOD: Properly skipping optional features
```go
// From 11.3.7_filter_geo_functions.go - geospatial is optional
func testGeoDistance(ctx *framework.TestContext) error {
    filter := url.QueryEscape("geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
    resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
    if err != nil {
        return err
    }
    
    switch resp.StatusCode {
    case 200:
        // ✅ When implemented, validate it works
        var result map[string]interface{}
        if err := json.Unmarshal(resp.Body, &result); err != nil {
            return fmt.Errorf("failed to parse response: %w", err)
        }
        if _, ok := result["value"]; !ok {
            return fmt.Errorf("response missing 'value' array")
        }
        return nil
    case 400, 404, 501:
        // ✅ Geospatial is optional, so skip is correct
        return ctx.Skip("geo.distance not implemented (optional feature)")
    default:
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
}
```

**Why Good Pattern is Better:**
- Distinguishes mandatory from optional features
- Helps identify actual compliance gaps
- Makes test results more meaningful

---

## Pattern 3: Required Header Validation

### ❌ BAD: Accepting missing required headers
```go
// From 8.2.5_header_location.go - Location is required!
func testLocationHeader(ctx *framework.TestContext) error {
    resp, err := ctx.POST("/Products", payload)
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 201 {
        return ctx.Skip("Entity creation not supported")
    }
    
    location := resp.Headers.Get("Location")
    if location != "" && !strings.Contains(location, "Products") {
        return framework.NewError("Location should contain entity URL")
    }
    
    return nil  // ❌ Passes even if location is empty!
}
```

### ✅ GOOD: Requiring mandatory headers
```go
func testLocationHeader(ctx *framework.TestContext) error {
    resp, err := ctx.POST("/Products", payload)
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 201 {
        return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
    }
    
    // ✅ OData v4 requires Location header in 201 responses
    location := resp.Headers.Get("Location")
    if location == "" {
        return fmt.Errorf("Location header is required for 201 responses")
    }
    
    if !strings.Contains(location, "Products") {
        return fmt.Errorf("Location header should contain entity URL, got: %s", location)
    }
    
    return nil
}
```

**Why Good Pattern is Better:**
- Enforces OData spec requirements
- Catches missing headers that clients depend on
- Validates complete HTTP response, not just body

---

## Pattern 4: Error Message Clarity

### ❌ BAD: Confusing error messages
```go
// From 11.2.13_type_casting.go - wrong error message
func testIsofFunction(ctx *framework.TestContext) error {
    if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
        return err  // Already skipped if no derived types
    }
    
    resp, err := ctx.GET("/Products?$filter=isof('Namespace.SpecialProduct')")
    if err != nil {
        return err
    }
    
    if resp.StatusCode == 200 {
        return nil
    }
    
    // ❌ This code is unreachable when correctly skipped above,
    // and error message is wrong (derived types are optional)
    if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
        return fmt.Errorf("specification violation: isof function must be supported (status: %d)", resp.StatusCode)
    }
    
    return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
```

### ✅ GOOD: Clear, accurate error messages
```go
func testIsofFunction(ctx *framework.TestContext) error {
    // Check if service declares derived types
    if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
        return err
    }
    
    resp, err := ctx.GET("/Products?$filter=isof('Namespace.SpecialProduct')")
    if err != nil {
        return err
    }
    
    // ✅ If derived types exist, isof MUST work
    if resp.StatusCode != 200 {
        return fmt.Errorf("isof function should work when derived types declared in metadata, got status %d", resp.StatusCode)
    }
    
    // ✅ Validate response structure
    var result map[string]interface{}
    if err := json.Unmarshal(resp.Body, &result); err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }
    
    if _, ok := result["value"]; !ok {
        return fmt.Errorf("response missing 'value' array")
    }
    
    return nil
}
```

**Why Good Pattern is Better:**
- Error messages accurately describe the problem
- Makes debugging easier
- Correctly identifies mandatory vs conditional requirements

---

## Pattern 5: Response Structure Validation

### ❌ BAD: Not validating response structure
```go
func testExpand(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$expand=Category")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    return nil  // ❌ Doesn't check if Category was actually expanded!
}
```

### ✅ GOOD: Validating expansion actually happened
```go
// From 11.2.5.6_query_expand.go
func testExpandBasic(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$expand=Descriptions&$top=1")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    if err := json.Unmarshal(resp.Body, &result); err != nil {
        return fmt.Errorf("failed to parse JSON: %w", err)
    }
    
    value, ok := result["value"].([]interface{})
    if !ok {
        return fmt.Errorf("response missing 'value' array")
    }
    
    if len(value) == 0 {
        return fmt.Errorf("response contains no items")
    }
    
    item, ok := value[0].(map[string]interface{})
    if !ok {
        return fmt.Errorf("first item is not an object")
    }
    
    // ✅ Verify field exists
    descriptions, ok := item["Descriptions"]
    if !ok {
        return fmt.Errorf("descriptions field is missing")
    }
    
    // ✅ Verify it's expanded (array), not just a link
    if _, ok := descriptions.([]interface{}); !ok {
        return fmt.Errorf("descriptions field is not an array (not properly expanded)")
    }
    
    return nil
}
```

**Why Good Pattern is Better:**
- Ensures $expand actually expands data
- Catches servers that return 200 but don't implement $expand
- Validates OData behavior, not just HTTP status

---

## Pattern 6: Select Validation

### ❌ BAD: Not checking what was excluded
```go
func testSelect(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$select=Name")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    json.Unmarshal(resp.Body, &result)
    
    value := result["value"].([]interface{})
    item := value[0].(map[string]interface{})
    
    // ✅ Checks selected field is present
    if _, ok := item["Name"]; !ok {
        return fmt.Errorf("Name field missing")
    }
    
    return nil  // ❌ But doesn't check if other fields were excluded!
}
```

### ✅ GOOD: Validating both inclusion and exclusion
```go
// From 11.2.5.2_query_select_orderby.go
func testSelectSingle(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$select=Name")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    json.Unmarshal(resp.Body, &result)
    
    value := result["value"].([]interface{})
    item := value[0].(map[string]interface{})
    
    // ✅ Verify selected field is present
    if _, ok := item["Name"]; !ok {
        return fmt.Errorf("selected field 'Name' is missing")
    }
    
    // ✅ Verify non-selected fields are absent
    for key := range item {
        // Allow metadata fields and ID
        if key == "@odata.context" || key == "@odata.etag" || 
           key == "@odata.id" || key == "ID" || key == "Name" {
            continue
        }
        // Any other field should not be present
        if key == "Description" || key == "Price" || key == "CategoryID" {
            return fmt.Errorf("response contains field '%s' which was not selected", key)
        }
    }
    
    return nil
}
```

**Why Good Pattern is Better:**
- Ensures $select actually limits fields
- Catches servers that return all fields regardless of $select
- Validates projection behavior correctly

---

## Pattern 7: OrderBy Validation

### ❌ BAD: Not checking actual order
```go
func testOrderBy(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$orderby=Price asc")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    return nil  // ❌ Doesn't verify results are actually ordered!
}
```

### ✅ GOOD: Validating sort order
```go
// From 11.2.5.2_query_select_orderby.go
func testOrderbyAsc(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$orderby=Price asc")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    json.Unmarshal(resp.Body, &result)
    
    value := result["value"].([]interface{})
    
    // ✅ Extract prices and verify ascending order
    var prices []float64
    for _, item := range value {
        entity := item.(map[string]interface{})
        if price, ok := entity["Price"].(float64); ok {
            prices = append(prices, price)
        }
    }
    
    // ✅ Verify prices are sorted
    for i := 1; i < len(prices); i++ {
        if prices[i] < prices[i-1] {
            return fmt.Errorf("prices not in ascending order: %v", prices)
        }
    }
    
    return nil
}
```

**Why Good Pattern is Better:**
- Ensures $orderby actually sorts data
- Catches servers that accept query but don't implement sorting
- Validates correct sort direction

---

## Pattern 8: Handling Invalid Input

### ❌ BAD: Not testing error cases
```go
func testFilter(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$filter=Price gt 100")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    return nil
    // ❌ Doesn't test what happens with invalid filters
}
```

### ✅ GOOD: Testing both valid and invalid input
```go
// Good test suite should include both:

func testFilterValid(ctx *framework.TestContext) error {
    resp, err := ctx.GET("/Products?$filter=Price gt 100")
    if err != nil {
        return err
    }
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    // Validate results...
    return nil
}

func testFilterInvalidSyntax(ctx *framework.TestContext) error {
    // ✅ Test invalid filter syntax
    resp, err := ctx.GET("/Products?$filter=InvalidSyntax")
    if err != nil {
        return err
    }
    
    // ✅ Should return 400 Bad Request
    if resp.StatusCode != 400 {
        return fmt.Errorf("invalid filter syntax should return 400, got %d", resp.StatusCode)
    }
    
    return nil
}

func testFilterNonExistentProperty(ctx *framework.TestContext) error {
    // ✅ Test filter on non-existent property
    resp, err := ctx.GET("/Products?$filter=NonExistent eq 'value'")
    if err != nil {
        return err
    }
    
    // ✅ Should return 400 Bad Request
    if resp.StatusCode != 400 {
        return fmt.Errorf("filter on non-existent property should return 400, got %d", resp.StatusCode)
    }
    
    return nil
}
```

**Why Good Pattern is Better:**
- Tests both happy path and error cases
- Ensures proper error handling
- Validates error response codes match spec

---

## Summary: Test Quality Checklist

When writing or reviewing tests, ensure they:

- [ ] ✅ Validate behavior, not just HTTP status
- [ ] ✅ Distinguish mandatory vs optional features
- [ ] ✅ Check response structure matches spec
- [ ] ✅ Verify feature actually works (filter filters, sort sorts, etc.)
- [ ] ✅ Test error cases and edge cases
- [ ] ✅ Have clear, accurate error messages
- [ ] ✅ Validate required headers are present
- [ ] ✅ Check both what should be included AND excluded
- [ ] ✅ Use appropriate assertions from framework
- [ ] ✅ Document assumptions and spec references

---

## Quick Reference: When to Skip vs Fail

| Scenario | Action | Reason |
|----------|--------|--------|
| Mandatory feature returns 404 | **FAIL** | Required by OData v4 |
| Optional feature returns 404 | **SKIP** | Optional per spec |
| Feature returns 200 but doesn't work | **FAIL** | Broken implementation |
| Feature returns 400 for valid input | **FAIL** | Invalid error code |
| Feature returns 500 | **FAIL** | Server error |
| Conditional feature missing (e.g., isof without derived types) | **SKIP** | Not applicable |
| Feature returns 501 (Not Implemented) | **SKIP** or **FAIL** | Depends if mandatory |

---

## Recommended Reading Order

1. Read this document first to understand patterns
2. Review `COMPLIANCE_TEST_ISSUES.md` for specific issues
3. Use `COMPLIANCE_TEST_FIXES_CHECKLIST.md` to prioritize work
4. Apply good patterns when fixing tests
5. Run tests after each change to verify
