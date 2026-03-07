# Entity Caching

go-odata supports optional per-replica caching of entity data to reduce round-trips to the
primary database for frequently-read, slowly-changing datasets.

## Cache Levels

| Level               | Behaviour                                                           |
|---------------------|---------------------------------------------------------------------|
| `CacheLevelNone`    | No caching (default). Every read queries the primary database.       |
| `CacheLevelFull`    | The entire entity dataset is loaded into a local file-based SQLite store (per service instance/replica). Reads are served from that store until the TTL expires or a write operation invalidates the cache. |

## When to Use Full Caching

`CacheLevelFull` is well-suited for small, slowly-changing lookup tables such as:

- Status codes / enumerations
- Country or region codes
- Product categories
- Configuration values

For entities that are updated frequently or are very large, caching provides little benefit and
may consume significant memory. In those cases, leave caching disabled (the default).

## Enabling Caching

Configure caching directly in `RegisterEntity` by passing `EntityCacheConfig`.

```go
service, err := odata.NewService(db)
if err != nil {
    log.Fatal(err)
}

// Enable full-dataset caching with a 10-minute TTL at registration time.
if err := service.RegisterEntity(&Category{}, odata.EntityCacheConfig{
    Level: odata.CacheLevelFull,
    TTL:   10 * time.Minute,
}); err != nil {
    log.Fatal(err)
}
```

### TTL (Time To Live)

`TTL` controls how long cached data is considered fresh. After the TTL elapses the next
read automatically refreshes the cache from the primary database.

- A zero `TTL` defaults to **5 minutes**.
- Use a shorter TTL (for example 1 minute) when the data changes regularly.
- Use a longer TTL (for example 1 hour) for truly static data.

### Passing `CacheLevelNone`

Passing `CacheLevelNone` to `RegisterEntity` is explicitly allowed and is a no-op.
This makes it straightforward to toggle caching via configuration without branching:

```go
service.RegisterEntity(&Category{}, odata.EntityCacheConfig{
    Level: odata.CacheLevelNone,
    TTL:   5 * time.Minute,
})
```

## Cache Invalidation

The cache is invalidated automatically whenever a write operation succeeds:

| HTTP method | Operation        | Cache invalidated? |
|-------------|------------------|--------------------|
| `POST`      | Create entity    | ✓                  |
| `PATCH`     | Update entity    | ✓                  |
| `PUT`       | Replace entity   | ✓                  |
| `DELETE`    | Delete entity    | ✓                  |
| `GET`       | Read collection  | ✗ (read-only)      |

After invalidation the very next read re-fetches the full dataset from the primary database
and repopulates the cache.

## OData Query Options and the Cache

When `CacheLevelFull` is active, standard OData query options continue to work:
`$filter`, `$orderby`, `$top`, `$skip`, `$count`, `$select`, and `$search`.

`$expand` remains available, but expansion queries run against the primary database so
navigation targets that are not present in the cache still resolve correctly.

## How It Works

1. On the first collection read after caching is enabled (or after invalidation / TTL
   expiry), go-odata loads all rows for the entity from the primary database and stores
    them in a private file-based SQLite cache local to that replica.
2. Subsequent reads are routed to this local cache until the TTL expires or a write
   invalidates the cache.
3. The cache is scoped to a single entity set — enabling caching for one entity has no
   effect on others.
4. The cache is held per-service instance. Horizontal scaling (multiple instances)
    means each instance maintains its own local copy. If you need distributed
   invalidation, refresh the TTL accordingly or handle invalidation in your deployment
   layer.

## Example — Category Lookup Table

```go
type Category struct {
    ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
    Name string `json:"Name"`
}

db.AutoMigrate(&Category{})

service, _ := odata.NewService(db)
service.RegisterEntity(&Category{}, odata.EntityCacheConfig{
    Level: odata.CacheLevelFull,
    TTL:   15 * time.Minute,
})

http.ListenAndServe(":8080", service)
```

All GET requests to `/Categories` (including `$filter`, `$orderby`, `$top`, `$skip`) will
be served from memory. POST, PATCH, PUT, and DELETE requests still reach the primary
database and automatically refresh the cache on success.
