package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	gormSpanKey      = "odata:gorm:span"
	gormStartTimeKey = "odata:gorm:start"
)

// RegisterGORMCallbacks registers GORM callbacks for database query tracing.
// This should be called after GORM is initialized and observability is configured.
func RegisterGORMCallbacks(db *gorm.DB, cfg *Config) error {
	if cfg == nil || cfg.TracerProvider == nil || !cfg.EnableDetailedDBTracing {
		return nil
	}

	tracer := cfg.Tracer()

	// Query callbacks
	if err := db.Callback().Query().Before("gorm:query").Register("odata:before_query", beforeQuery(tracer)); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register("odata:after_query", afterQuery(tracer, cfg)); err != nil {
		return err
	}

	// Create callbacks
	if err := db.Callback().Create().Before("gorm:create").Register("odata:before_create", beforeCreate(tracer)); err != nil {
		return err
	}
	if err := db.Callback().Create().After("gorm:create").Register("odata:after_create", afterCreate(tracer, cfg)); err != nil {
		return err
	}

	// Update callbacks
	if err := db.Callback().Update().Before("gorm:update").Register("odata:before_update", beforeUpdate(tracer)); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("odata:after_update", afterUpdate(tracer, cfg)); err != nil {
		return err
	}

	// Delete callbacks
	if err := db.Callback().Delete().Before("gorm:delete").Register("odata:before_delete", beforeDelete(tracer)); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("odata:after_delete", afterDelete(tracer, cfg)); err != nil {
		return err
	}

	// Row callbacks
	if err := db.Callback().Row().Before("gorm:row").Register("odata:before_row", beforeRow(tracer)); err != nil {
		return err
	}
	if err := db.Callback().Row().After("gorm:row").Register("odata:after_row", afterRow(tracer, cfg)); err != nil {
		return err
	}

	// Raw callbacks
	if err := db.Callback().Raw().Before("gorm:raw").Register("odata:before_raw", beforeRaw(tracer)); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("odata:after_raw", afterRaw(tracer, cfg)); err != nil {
		return err
	}

	return nil
}

func beforeQuery(tracer *Tracer) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startSpan(db, tracer, "db.query")
	}
}

func afterQuery(tracer *Tracer, cfg *Config) func(*gorm.DB) {
	return func(db *gorm.DB) {
		endSpan(db, tracer, cfg, "SELECT")
	}
}

func beforeCreate(tracer *Tracer) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startSpan(db, tracer, "db.create")
	}
}

func afterCreate(tracer *Tracer, cfg *Config) func(*gorm.DB) {
	return func(db *gorm.DB) {
		endSpan(db, tracer, cfg, "INSERT")
	}
}

func beforeUpdate(tracer *Tracer) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startSpan(db, tracer, "db.update")
	}
}

func afterUpdate(tracer *Tracer, cfg *Config) func(*gorm.DB) {
	return func(db *gorm.DB) {
		endSpan(db, tracer, cfg, "UPDATE")
	}
}

func beforeDelete(tracer *Tracer) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startSpan(db, tracer, "db.delete")
	}
}

func afterDelete(tracer *Tracer, cfg *Config) func(*gorm.DB) {
	return func(db *gorm.DB) {
		endSpan(db, tracer, cfg, "DELETE")
	}
}

func beforeRow(tracer *Tracer) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startSpan(db, tracer, "db.row")
	}
}

func afterRow(tracer *Tracer, cfg *Config) func(*gorm.DB) {
	return func(db *gorm.DB) {
		endSpan(db, tracer, cfg, "ROW")
	}
}

func beforeRaw(tracer *Tracer) func(*gorm.DB) {
	return func(db *gorm.DB) {
		startSpan(db, tracer, "db.raw")
	}
}

func afterRaw(tracer *Tracer, cfg *Config) func(*gorm.DB) {
	return func(db *gorm.DB) {
		endSpan(db, tracer, cfg, "RAW")
	}
}

func startSpan(db *gorm.DB, tracer *Tracer, spanName string) {
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, span := tracer.StartSpan(ctx, spanName,
		attribute.String("db.system", "gorm"),
	)

	db.Statement.Context = ctx
	db.InstanceSet(gormSpanKey, span)
	db.InstanceSet(gormStartTimeKey, time.Now())
}

func endSpan(db *gorm.DB, tracer *Tracer, cfg *Config, operation string) {
	spanVal, ok := db.InstanceGet(gormSpanKey)
	if !ok {
		return
	}

	span, ok := spanVal.(trace.Span)
	if !ok {
		return
	}
	defer span.End()

	// Add SQL statement info
	if db.Statement != nil {
		tableName := db.Statement.Table
		if tableName != "" {
			span.SetAttributes(attribute.String("db.sql.table", tableName))
		}
		span.SetAttributes(attribute.Int64("db.rows_affected", db.RowsAffected))
	}

	// Record error if any
	if db.Error != nil {
		tracer.RecordError(span, db.Error)
		span.SetStatus(codes.Error, db.Error.Error())
	}

	// Record metrics
	if startTimeVal, ok := db.InstanceGet(gormStartTimeKey); ok {
		if startTime, ok := startTimeVal.(time.Time); ok {
			duration := time.Since(startTime)
			cfg.Metrics().RecordDBQuery(db.Statement.Context, operation, duration)
		}
	}
}
