# PostgreSQL Performance Quick Reference

This guide provides quick reference for optimizing go-odata performance with PostgreSQL.

## Recommended Database Indexes

Add these indexes to your entity models for optimal performance:

```go
type Product struct {
    // Add 'index' tag for frequently queried columns
    Price      float64   `gorm:"not null;index" odata:"required"`
    CategoryID *uint     `gorm:"index" odata:"nullable"`
    CreatedAt  time.Time `gorm:"not null;index"`
    Name       string    `gorm:"not null;index" odata:"searchable"`
    // ... other fields
}
```

### When to Add Indexes

| Column Type | Add Index If | Expected Benefit |
|-------------|--------------|------------------|
| **Price/Numeric** | Used in `$filter` or `$orderby` | +15-20% throughput |
| **Foreign Keys** | Used in `$expand` or joins | Faster joins |
| **Timestamps** | Used for temporal queries | +10-15% throughput |
| **Text Fields** | Used with `$search` or `$filter` | Faster searches |

## Connection Pool Settings

Configure connection pooling for production:

```go
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    PrepareStmt: true,  // Enable prepared statement cache
})

sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(50)                  // Max connections
sqlDB.SetMaxIdleConns(25)                  // Idle connections
sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Connection lifetime
sqlDB.SetConnMaxIdleTime(1 * time.Minute)  // Idle timeout
```

**Recommended Settings:**
- `MaxOpenConns`: 25-100 (depends on your DB server capacity)
- `MaxIdleConns`: 50% of MaxOpenConns
- `ConnMaxLifetime`: 5-15 minutes
- `ConnMaxIdleTime`: 1-5 minutes

## Performance Benchmarks (PostgreSQL 16)

Based on load testing with 10,000 products, 100 categories:

| Operation Type | Throughput | p50 Latency | Rating |
|---------------|------------|-------------|--------|
| Service Document | 56k req/sec | 1.7ms | ⭐⭐⭐⭐⭐ |
| Metadata | 49k req/sec | 1.9ms | ⭐⭐⭐⭐⭐ |
| Single Entity | 1.4k req/sec | 43ms | ⭐⭐⭐⭐ |
| Simple Collection (100) | 820 req/sec | 134ms | ⭐⭐⭐ |
| Filter + OrderBy | 401 req/sec | 235ms | ⭐⭐⭐ |
| Complex Query | 420 req/sec | 267ms | ⭐⭐⭐ |

**Hardware:** GitHub Actions runner (2 CPU cores, 7GB RAM)  
**Concurrency:** 100 connections, 10 threads

## Quick Optimization Checklist

### ✅ High Priority (Easy Wins)

- [ ] Add indexes on filtered/sorted columns (Price, Status, etc.)
- [ ] Add indexes on foreign keys (CategoryID, etc.)
- [ ] Configure connection pooling (see settings above)
- [ ] Use `$select` to fetch only needed columns
- [ ] Use `$top` to limit response sizes

### ✅ Medium Priority

- [ ] Implement response caching for rarely-changed data
- [ ] Monitor slow queries with `EXPLAIN ANALYZE`
- [ ] Enable prepared statements in GORM
- [ ] Consider read replicas for read-heavy workloads

### ✅ Low Priority (Advanced)

- [ ] Use alternative JSON serializer (jsoniter)
- [ ] Implement application-level caching (Redis)
- [ ] Use PgBouncer for connection pooling
- [ ] Partition large tables (>1M rows)

## Common Performance Issues

### Issue: Slow Collection Queries (>500ms)

**Symptoms:** High latency fetching entity collections  
**Solutions:**
1. Add index on filtered/sorted columns
2. Reduce `$top` page size (max 500 recommended)
3. Use `$select` to fetch fewer columns
4. Check query plan with `EXPLAIN ANALYZE`

### Issue: High Memory Usage

**Symptoms:** Memory grows continuously  
**Solutions:**
1. Limit `$top` to reasonable values (≤500)
2. Implement pagination with `$skip`
3. Use streaming for large responses
4. Check for memory leaks with pprof

### Issue: Connection Pool Exhaustion

**Symptoms:** "too many connections" errors  
**Solutions:**
1. Increase PostgreSQL `max_connections`
2. Reduce app `MaxOpenConns` setting
3. Use PgBouncer connection pooler
4. Check for connection leaks (not closing transactions)

### Issue: Slow Filtering/Sorting

**Symptoms:** `$filter` and `$orderby` are slow  
**Solutions:**
1. Add index on filtered/sorted column
2. Use composite indexes for multi-column filters
3. Avoid `LIKE '%term%'` (can't use index)
4. Consider full-text search for text searches

## Monitoring Queries

### View Active Queries

```sql
-- See currently running queries
SELECT pid, now() - query_start as duration, query 
FROM pg_stat_activity 
WHERE state = 'active' 
ORDER BY duration DESC;
```

### Analyze Query Performance

```sql
-- Explain query execution
EXPLAIN ANALYZE SELECT * FROM products WHERE price > 500 ORDER BY price DESC LIMIT 100;
```

### Monitor Index Usage

```sql
-- Check index usage statistics
SELECT 
    schemaname, tablename, indexname,
    idx_scan as scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes 
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;
```

### Find Slow Queries

Enable slow query logging in `postgresql.conf`:

```
log_min_duration_statement = 1000  # Log queries > 1 second
log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '
```

## Production Deployment

### Database Configuration

```ini
# postgresql.conf - Recommended settings for go-odata
max_connections = 200
shared_buffers = 256MB              # 25% of RAM
effective_cache_size = 1GB          # 50-75% of RAM
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1              # For SSD storage
work_mem = 4MB
```

### Index Maintenance

```sql
-- Rebuild indexes monthly
REINDEX TABLE products;

-- Update statistics weekly
ANALYZE products;

-- Check for bloat
SELECT 
    schemaname, tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

## Performance Testing

Run performance tests against your own data:

```bash
# Start the performance server with PostgreSQL
cd cmd/perfserver
go run . -db postgres -dsn "postgresql://user:pass@localhost:5432/dbname?sslmode=disable"

# In another terminal, run load tests
./run_load_tests.sh --db postgres --dsn "postgresql://user:pass@localhost:5432/dbname?sslmode=disable"

# View results
ls -lh load-test-results/
```

## Additional Resources

- **Full Analysis:** See [PERFORMANCE_ANALYSIS_POSTGRESQL.md](../PERFORMANCE_ANALYSIS_POSTGRESQL.md)
- **Improvements:** See [PERFORMANCE_IMPROVEMENTS.md](../PERFORMANCE_IMPROVEMENTS.md)
- **Testing Guide:** See [documentation/testing.md](../documentation/testing.md)
- **PostgreSQL Tuning:** https://pgtune.leopard.in.ua/

---

**Need Help?**
- Check the [GitHub Issues](https://github.com/NLstn/go-odata/issues)
- Review existing performance discussions
- Open a new issue with your performance question
