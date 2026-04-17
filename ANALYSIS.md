Now I have a thorough understanding of the library. Here is the gap analysis:

---

## OData v4 / v4.01 Feature Gap Analysis

### Filter / Query Expression Functions

| Missing Feature | Spec | Notes |
|---|---|---|
| `divby` arithmetic operator | v4.01 | Decimal/floating-point division; library has `div` (integer) but not `divby` |
| `matchesPattern(str, pattern)` | v4.01 | POSIX regex pattern matching on strings |
| `fractionalseconds(datetime)` | v4.0 | Extracts sub-second component from DateTimeOffset/TimeOfDay |
| `totaloffsetminutes(datetime)` | v4.0 | Returns UTC offset in minutes from a DateTimeOffset |
| `totalseconds(duration)` | v4.0 | Returns total seconds of an Edm.Duration value |
| `mindatetime()` / `maxdatetime()` | v4.0 | Minimum/maximum DateTimeOffset literals (used in $apply aggregations) |

### URL Conventions

| Missing Feature | Spec | Notes |
|---|---|---|
| `$crossjoin(Set1,Set2,...)` | v4.0 | Cartesian product across entity sets; endpoint entirely absent |
| `$expand=*` wildcard | v4.01 | Expand all navigation properties in one expression |
| `$select=*` explicit wildcard | v4.01 | Explicitly select all declared properties (implied by omission, but `*` is not parsed) |

### `$apply` Transformations

| Missing Feature | Spec | Notes |
|---|---|---|
| `rollup()` | v4.0 aggregation ext. | Subtotals at multiple aggregation levels within `groupby` |
| `from()` clause in aggregate | v4.0 aggregation ext. | Aggregate from a related collection path; parsed but navigation-based execution not yet supported at SQL-builder layer |
| `nest()` transformation | v4.0 aggregation ext. | Nests a transformation result as a sub-collection; parsed but sub-query nesting not yet executed at SQL-builder layer |

### Metadata / EDMX (CSDL)

| Missing Feature | Spec | Notes |
|---|---|---|
| Abstract entity types (`Abstract="true"`) | v4.0 | No way to mark a Go struct as an abstract OData entity type |
| Entity type inheritance (`BaseType` attribute) | v4.0 | Deriving one EntityType from another; absent from metadata model |
| TypeDefinition elements | v4.0 | Custom named types aliasing primitives (e.g., `type Weight = Edm.Decimal`); only enums have underlying-type support |
| Navigation properties on complex types | v4.0 | Complex types can have nav properties per spec; this path is not handled |

### Response Format

| Missing Feature | Spec | Notes |
|---|---|---|
| XML/Atom format for data responses | v4.0 | `$format=atom` / `Accept: application/atom+xml`; XML is supported only for `$metadata`, not entity/collection responses |
| `@odata.nextLink` inside expanded collections | v4.0 | Server-driven paging within `$expand` results when the expanded collection is truncated |

### `Prefer` Header

| Missing Feature | Spec | Notes |
|---|---|---|

### HTTP Operations

| Missing Feature | Spec | Notes |
|---|---|---|
| Deep update | v4.0 | Updating related entities inline in a single `PATCH` (as opposed to `@odata.bind` which only links, not modifies) |
| Upsert via `PUT` to non-existent key | v4.0 | `PUT` to a resource that doesn't exist should create it (upsert semantics); currently `PUT` requires pre-existing entity |

### OData v4.01-Specific Gaps (beyond above)

| Missing Feature | Notes |
|---|---|
| Optional `$` prefix on query options | v4.01 allows writing `filter=...` instead of `$filter=...`; library requires the `$` prefix |
| `continue-on-error` outside batch | v4.01 allows this preference on non-batch collection modifications |
| `Edm.Untyped` as first-class property type | Appears only in function return contexts; not registerable as a property type on entity structs |

---

**Summary:** The library is comprehensive for standard OData v4 usage. The most impactful gaps are: `$crossjoin`, XML/Atom data format, entity type inheritance, abstract types, TypeDefinitions, the `divby` / `matchesPattern` filter additions, `rollup()` in `$apply`, and nested paging via `@odata.nextLink` in expanded collections.