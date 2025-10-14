# OData v4 Compliance Test Report

This is a file to track OData V4 compliance test statuses against the go-odata library. It should not contain anything other
than a list of passing tests, a list of failing tests and a a list of the tests with partitial support or warnings.
All lists have to be ordered by the number of their test. 

**Important!** the devserver has to be restarted before every test run to ensure the database is seeded with the same data.

### ✅ Passing Tests (113/152)

| Test # | Feature | Status | Notes |
|--------|---------|--------|-------|
| 1 | Service Root Document | ✅ PASS | Returns proper JSON with `@odata.context` and `value` array |
| 2 | Metadata Document | ✅ PASS | Valid EDMX 4.0 XML structure |
| 3 | Entity Collection | ✅ PASS | Returns collection with proper `@odata.context` |
| 4 | Single Entity Retrieval | ✅ PASS | Correct context: `$metadata#Products/$entity` |
| 5 | $filter (basic) | ✅ PASS | `$filter=Price gt 500` works correctly |
| 6 | $select | ✅ PASS | Returns only requested properties |
| 7 | $orderby | ✅ PASS | Sorts results correctly (asc/desc) |
| 8 | $top and $skip | ✅ PASS | Pagination works as expected |
| 9 | $count=true | ✅ PASS | Returns `@odata.count` in response |
| 10 | $count Endpoint | ✅ PASS | Returns plain text integer "5" as per spec |
| 11 | Navigation Properties | ✅ PASS | `/Products(1)/Descriptions` works |
| 12 | $expand | ✅ PASS | Expands navigation properties inline |
| 13 | String Functions | ✅ PASS | `contains()` function works |
| 14 | Simple Filters | ✅ PASS | Single conditions work correctly |
| 14b | Complex Filters (AND) | ✅ PASS | Correctly filters with AND logic |
| 15 | HTTP Headers | ✅ PASS | `OData-Version: 4.0` header present |
| 16 | Composite Keys | ✅ PASS | Successfully retrieves entities with composite keys |
| 17 | Property Access | ✅ PASS | Returns property value in OData format |
| 18 | POST (Create) | ✅ PASS | Creates entities, returns 201 with created entity |
| 19 | Error Handling | ✅ PASS | Proper error format for missing entities |
| 20 | $format Query | ✅ PASS | Format query option works |
| 21 | Combined Queries | ✅ PASS | Multiple query options work together with `@odata.nextLink` |
| 22 | @odata.nextLink Navigation | ✅ PASS | Pagination links work correctly |
| 23 | String Comparison Filters | ✅ PASS | Category eq 'Electronics' works |
| 24 | OR Logical Operator | ✅ PASS | Correctly filters with OR logic |
| 25 | startswith() Function | ✅ PASS | String function works correctly |
| 26 | endswith() Function | ✅ PASS | String function works correctly |
| 27 | tolower() Function | ✅ PASS | String transformation function works ✨ FIXED |
| 28 | length() Function | ✅ PASS | String length function works ✨ FIXED |
| 29 | not Operator | ✅ PASS | Logical NOT operator works correctly ✨ FIXED |
| 30 | ne (Not Equal) Operator | ✅ PASS | Comparison operator works |
| 31 | Combined AND Filter | ✅ PASS | Price ranges work correctly |
| 32 | Multiple OrderBy | ✅ PASS | Multiple sort properties work |
| 33 | $expand with $select | ✅ PASS | Nested $select in expand works ✨ FIXED |
| 34 | $expand with $filter | ✅ PASS | Filter expanded collections |
| 35 | $expand with $top | ✅ PASS | Limit expanded collections |
| 36 | $expand on Collections | ✅ PASS | Expand works on entity collections |
| 37 | $skip=0 Edge Case | ✅ PASS | Zero skip handled correctly |
| 38 | $skip Beyond Records | ✅ PASS | Returns empty array when skipping beyond data |
| 39 | Case Sensitivity | ✅ PASS | Filters are case-sensitive (expected behavior) |
| 40 | substring() Function | ✅ PASS | Substring extraction works ✨ FIXED |
| 41 | Accept Header Metadata | ✅ PASS | Includes @odata.type annotations for odata.metadata=full ✨ FIXED |
| 42 | $select Validation | ✅ PASS | Returns error for non-existent properties ✨ FIXED |
| 43 | OrderBy Validation | ✅ PASS | Returns error for non-existent properties |
| 44 | OPTIONS Method (CORS) | ✅ PASS | OPTIONS method now supported ✨ FIXED |
| 45 | URL Encoding | ✅ PASS | Spaces in filter values handled correctly |
| 46 | Parentheses in Filters | ✅ PASS | Expression grouping supported |
| 47 | indexof() Function | ✅ PASS | Function now recognized and works |
| 48 | Date Functions | ✅ PASS | year(), month(), day() functions work with date properties ✨ FIXED |
| 49 | Navigation in $select | ✅ PASS | Returns navigation properties in select |
| 51 | ge Operator | ✅ PASS | Returns 2 products correctly ✨ FIXED |
| 52 | le Operator | ✅ PASS | Less than or equal operator works |
| 53 | null Comparison | ✅ PASS | Can compare properties with null |
| 54 | Nested $expand | ✅ PASS | Can expand navigation properties of expanded entities |
| 55 | $expand with $skip | ✅ PASS | Skip works in nested expand queries |
| 56 | $expand with $orderby | ✅ PASS | OrderBy works in nested expand queries |
| 57 | $expand with $levels | ✅ PASS | Accepts $levels parameter (recursive expansion) |
| 58 | If-Match Ignored on GET | ✅ PASS | If-Match header properly ignored for GET requests |
| 59 | Content-Type Header | ✅ PASS | Returns proper Content-Type with odata.metadata parameter |
| 60 | Arithmetic Operators | ✅ PASS | All arithmetic operators work (add, sub, mul, div, mod) - both function and infix syntax ✨ FIXED |
| 61 | concat() Function | ✅ PASS | String concatenation function works ✨ FIXED |
| 62 | trim() Function | ✅ PASS | String trim function works ✨ FIXED |
| 63 | toupper() Function | ✅ PASS | String transformation function works ✨ FIXED |
| 64 | in Operator | ✅ PASS | 'in' operator now working ✨ FIXED |
| 65 | has Operator | ✅ PASS | 'has' operator for enum flags now working ✨ FIXED |
| 66 | Math Functions | ✅ PASS | ceiling(), floor(), round() functions work ✨ FIXED |
| 67 | Lambda Operators | ✅ PASS | any() and all() lambda operators now working ✨ FIXED |
| 68 | HEAD Method | ✅ PASS | HEAD method now supported ✨ FIXED |
| 69 | If-None-Match | ✅ PASS | If-None-Match header supported (304 responses) ✨ FIXED |
| 70 | $batch Endpoint | ✅ PASS | $batch endpoint exists (POST only) |
| 71 | isof() Type Function | ✅ PASS | Type checking function works for both EDM types and entity types ✨ FIXED |
| 75 | Location Header on POST | ✅ PASS | Location header present on 201 Created responses |
| 77 | Allow Header (Collections) | ✅ PASS | Allow header lists GET, HEAD, POST, OPTIONS |
| 78 | Allow Header (Entity) | ✅ PASS | Allow header lists GET, HEAD, PATCH, PUT, DELETE, OPTIONS |
| 79 | $format=xml Entity Data | ✅ PASS | XML format correctly unsupported for entity data (deprecated in OData v4, only required for $metadata CSDL) |
| 80 | $metadata JSON Format | ✅ PASS | JSON metadata format supported |
| 81 | Query Option Case Sensitivity | ✅ PASS | Query options are case-sensitive (OData v4 spec) |
| 82 | Invalid Query Options | ✅ PASS | Returns error for invalid query options |
| 83 | DELETE Status Code | ✅ PASS | DELETE returns 204 No Content |
| 84 | PUT Status Code | ✅ PASS | PUT returns 204 No Content by default |
| 85 | PATCH Status Code | ✅ PASS | PATCH returns 204 No Content by default |
| 86 | POST Status Code | ✅ PASS | POST returns 201 Created |
| 87 | Error Format Compliance | ✅ PASS | Error follows OData v4 format (code, message) |
| 90 | Multiple OrderBy Directions | ✅ PASS | Multiple orderby with mixed directions works |
| 91 | $count with $filter | ✅ PASS | Returns correct count of 3 (Laptop, Office Chair, Smartphone with Price > 100) ✨ FIXED |
| 92 | Deep $expand | ✅ PASS | Multi-level expand support |
| 93 | Empty Result Format | ✅ PASS | Empty result set properly formatted with @odata.context and empty value array ✨ FIXED |
| 94 | Property Paths in Filter | ✅ PASS | Property path navigation works |
| 95 | $count endpoint with $filter | ✅ PASS | Returns correct count of 3 with filter applied ✨ FIXED |
| 96 | ETag on GET | ✅ PASS | ETag header present on GET responses |
| 97 | If-Match Validation | ✅ PASS | Returns 412 Precondition Failed for ETag mismatch |
| 99 | cast() Function | ✅ PASS | Type casting function now works ✨ FIXED |
| 109 | $apply with groupby | ✅ PASS | Data aggregation groupby works ✨ NEW |
| 110 | $apply with aggregate | ✅ PASS | Data aggregation aggregate works ✨ NEW |
| 111 | $apply with filter | ✅ PASS | Data aggregation filter transformation works ✨ NEW |
| 112 | $apply with compute | ✅ PASS | Data aggregation compute works ✨ NEW |
| 116 | Nested Collections | ✅ PASS | Collections within entities accessible |
| 120 | Delta Tokens | ✅ PASS | Delta tokens accepted (ignored but no error) ✨ NEW |
| 122 | Enum Type Support | ✅ PASS | Enum types now in metadata ✨ NEW |
| 126 | Prefer: return=minimal | ✅ PASS | Return minimal preference honored (204 response) ✨ NEW |
| 127 | Prefer: return=representation | ✅ PASS | Return representation preference honored (201 with body) ✨ NEW |
| 131 | $value on Primitives | ✅ PASS | Raw primitive value access works ✨ NEW |
| 132 | Raw Value Content-Type | ✅ PASS | text/plain content type for raw values works ✨ NEW |
| 133 | $skip without $orderby | ✅ PASS | Skip works without orderby (order undefined) |
| 134 | Complex Query Combination | ✅ PASS | Multiple query options work together |
| 135 | $filter with Navigation | ✅ PASS | Filter with navigation properties now works (any() function) ✨ FIXED |
| 137 | $select with Navigation Path | ✅ PASS | Path-based select with expand now works ✨ FIXED |
| 138 | Singleton Access | ✅ PASS | Singleton access works ✨ FIXED |
| 142 | Bound Action on Entity | ✅ PASS | Bound actions work ✨ NEW |
| 143 | $expand with $levels=max | ✅ PASS | Recursive expand to maximum depth works ✨ NEW |
| 146 | IEEE754Compatible | ✅ PASS | IEEE754Compatible format parameter accepted |
| 148 | $search with AND | ✅ PASS | Search AND operator works ✨ NEW |
| 149 | $search with OR | ✅ PASS | Search OR operator works ✨ NEW |
| 150 | $search with NOT | ✅ PASS | Search NOT operator works ✨ NEW |

### ⚠️ Partially Passing Tests (0/152)

No partially passing tests at this time.

### ❌ Failing Tests (30/152)

| Test # | Feature | Status | Issue | Priority |
|--------|---------|--------|-------|----------|
| 72 | $ref Entity References | ❌ FAIL | Returns full entities instead of just @odata.id references | Low |
| 73 | @odata.id Annotation | ❌ FAIL | Entity responses don't include @odata.id | Low |
| 74 | @odata.editLink Annotation | ❌ FAIL | Entity responses don't include @odata.editLink | Low |
| 76 | OData-EntityId Header | ❌ FAIL | OData-EntityId header not present | Low |
| 88 | null Literal in Filter | ❌ FAIL | null literal not supported in $filter expressions | Medium |
| 89 | Boolean Literal in Filter | ❌ FAIL | true/false literals not supported in $filter | Medium |
| 98 | Prefer: maxpagesize | ❌ FAIL | odata.maxpagesize preference not working | Medium |
| 100 | matchesPattern() Function | ❌ FAIL | Regex pattern matching not supported | Low |
| 101 | date() Function | ❌ FAIL | Extract date component not supported | Low |
| 102 | time() Function | ❌ FAIL | Extract time component not supported | Low |
| 103 | now() Function | ❌ FAIL | Current datetime function not supported | Medium |
| 104 | totaloffsetminutes() Function | ❌ FAIL | Timezone offset function not supported | Low |
| 105 | totalseconds() Function | ❌ FAIL | Duration to seconds not supported | Low |
| 106 | fractionalseconds() Function | ❌ FAIL | Fractional seconds extraction not supported | Low |
| 107 | mindatetime() Function | ❌ FAIL | Minimum datetime constant not supported | Low |
| 108 | maxdatetime() Function | ❌ FAIL | Maximum datetime constant not supported | Low |
| 113 | Parameter Aliases | ❌ FAIL | @param aliasing not supported | Medium |
| 114 | $it in Lambda | ❌ FAIL | $it reference in lambda not supported | High |
| 115 | Multiple $expand Paths | ❌ FAIL | Comma-separated expand paths not supported | Medium |
| 117 | $count on Navigation | ❌ FAIL | $count returns full data instead of integer count | Medium |
| 136 | $orderby with Navigation | ❌ FAIL | OrderBy with navigation paths not supported | Medium |
| 118 | Deep Navigation Paths | ❌ FAIL | Multi-level navigation not supported | Low |
| 119 | Delta Links | ❌ FAIL | Change tracking delta links not supported | Low |
| 121 | Prefer: track-changes | ❌ FAIL | Track changes preference not supported | Low |
| 123 | Complex Type Properties | ❌ FAIL | Complex types not in metadata | Medium |
| 124 | Collection of Primitives | ❌ FAIL | Primitive type collections not supported | Medium |
| 125 | Collection of Complex Types | ❌ FAIL | Complex type collections not supported | Medium |
| 128 | Async Request Processing | ❌ FAIL | Async processing (202 responses) not supported | Low |
| 129 | respond-async Preference | ❌ FAIL | Respond async preference not supported | Low |
| 130 | $ref POST | ❌ FAIL | Entity reference creation not supported | Medium |
| 139 | Function Import | ❌ FAIL | Unbound functions not supported | High |
| 140 | Action Import | ❌ FAIL | Unbound actions not supported | High |
| 141 | Bound Function on Entity | ❌ FAIL | Bound functions not supported | High |
| 144 | $expand with * | ❌ FAIL | Expand all navigation properties not supported | Medium |
| 145 | $select with * | ❌ FAIL | Select all properties not supported | Medium |
| 147 | Media Entity $value | ❌ FAIL | No media entities in service | Low |