# Geospatial Functions

The go-odata library implements geospatial functions according to the OData v4 specification. Geospatial functions allow you to query and filter entities based on their geographic locations and spatial relationships.

## Overview

Geospatial functions are defined in the OData v4 specification:
- [OData v4.0 URL Conventions - Geospatial Functions](https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_GeospatialFunctions)

## Supported Functions

### geo.distance

Calculates the distance between two points. Returns the distance in meters.

**Syntax:**
```
geo.distance(point1, point2)
```

**Example:**
```http
GET /Locations?$filter=geo.distance(Position, geography'SRID=4326;POINT(-122.1 47.6)') lt 10000
```

This query returns all locations within 10,000 meters (10 km) of the specified point.

### geo.length

Calculates the length of a linestring. Returns the length in meters.

**Syntax:**
```
geo.length(linestring)
```

**Example:**
```http
GET /Routes?$filter=geo.length(Path) gt 5000
```

This query returns all routes longer than 5,000 meters.

### geo.intersects

Tests whether two geometries intersect. Returns true if they intersect, false otherwise.

**Syntax:**
```
geo.intersects(geometry1, geometry2)
```

**Example:**
```http
GET /Regions?$filter=geo.intersects(Area, geography'SRID=4326;POLYGON((0 0,10 0,10 10,0 10,0 0))')
```

This query returns all regions that intersect with the specified polygon.

## Geospatial Literals

OData supports two types of geospatial literals:

### Geography Literals

Geography types represent data in a round-earth coordinate system (WGS84 by default).

**Format:**
```
geography'SRID=4326;POINT(longitude latitude)'
```

**Examples:**
- Point: `geography'SRID=4326;POINT(-122.1 47.6)'`
- LineString: `geography'SRID=4326;LINESTRING(-122 47, -122.1 47.1, -122.2 47.2)'`
- Polygon: `geography'SRID=4326;POLYGON((0 0,10 0,10 10,0 10,0 0))'`

### Geometry Literals

Geometry types represent data in a flat-earth coordinate system.

**Format:**
```
geometry'POINT(x y)'
```

**Examples:**
- Point: `geometry'POINT(100 50)'`
- LineString: `geometry'LINESTRING(0 0, 10 10, 20 20)'`
- Polygon: `geometry'POLYGON((0 0,100 0,100 100,0 100,0 0))'`

## Combining with Other Filters

Geospatial functions can be combined with other filter expressions using logical operators:

```http
GET /Stores?$filter=Category eq 'Restaurant' and geo.distance(Location, geography'SRID=4326;POINT(-122.1 47.6)') lt 1000
```

This returns all restaurants within 1 km of the specified point.

## SQL Backend

The geospatial functions are translated to SQL using spatial extension functions:

- `geo.distance` → `ST_Distance(column, ST_GeomFromText(?))`
- `geo.length` → `ST_Length(column)`
- `geo.intersects` → `ST_Intersects(column, ST_GeomFromText(?))`

**Note:** To use geospatial functions, your database must support spatial extensions:
- SQLite: Install SpatiaLite extension
- PostgreSQL: Install PostGIS extension
- MySQL: Built-in spatial support

If spatial extensions are not available, queries will fail at the database level.

## Example Entity with Geospatial Properties

```go
type Store struct {
    ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
    Name     string  `json:"Name"`
    Location string  `json:"Location"` // In production, use a proper geospatial type
    Category string  `json:"Category"`
}

// With PostGIS, you might use:
import "github.com/paulmach/orb"

type Store struct {
    ID       uint        `json:"ID" gorm:"primaryKey" odata:"key"`
    Name     string      `json:"Name"`
    Location orb.Point   `json:"Location" gorm:"type:geography(Point,4326)"`
    Category string      `json:"Category"`
}
```

## Limitations

1. **Database Support Required**: Geospatial functions require database-level support through spatial extensions. Without these extensions, queries will fail.

2. **SRID Support**: The implementation assumes WGS84 (SRID 4326) for geography types. Other SRIDs may work depending on your database configuration.

3. **Geometry Types**: The current implementation supports basic geometry types (Point, LineString, Polygon). More complex types may require additional configuration.

4. **Performance**: Spatial queries can be slow without proper spatial indexing. Make sure to create spatial indexes on geospatial columns:

```sql
-- PostgreSQL with PostGIS
CREATE INDEX idx_store_location ON stores USING GIST (location);

-- SQLite with SpatiaLite
SELECT CreateSpatialIndex('stores', 'location');
```

## Testing

The library includes comprehensive tests for geospatial functions:

```bash
# Run geospatial function tests
go test ./internal/query -v -run TestGeo

# Run compliance tests (requires a running server with spatial support)
./compliance/run_compliance_tests.sh geo
```

## OData v4 Compliance

The geospatial function implementation follows the OData v4 specification. According to the spec, geospatial functions are **optional features**, meaning not all OData services are required to support them.

For full compliance details, see:
- [OData v4.0 Geospatial Functions Specification](https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_GeospatialFunctions)
