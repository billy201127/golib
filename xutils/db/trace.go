package db

import (
	"context"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/XSAM/otelsql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

var (
	driverName string
	once       sync.Once
	dbCache    sync.Map // dsn -> sqlx.SqlConn
)

// Initialize OTel driver
func initDriver() {
	once.Do(func() {
		var err error
		driverName, err = otelsql.Register(
			"mysql",
			// Mark database type
			otelsql.WithAttributes(semconv.DBSystemNameMySQL),
			// Ensure SQL text is written to span
			otelsql.WithSpanOptions(otelsql.SpanOptions{
				DisableQuery:   false, // Ensure SQL query statements are recorded
				DisableErrSkip: true,
			}),
			// Record SQL statements and parameters
			otelsql.WithAttributesGetter(func(ctx context.Context, method otelsql.Method, query string, args []driver.NamedValue) []attribute.KeyValue {
				// Build complete SQL statement
				completeSQL := buildCompleteSQL(query, args)

				attrs := []attribute.KeyValue{
					// Record complete SQL statement
					attribute.String("db.statement", completeSQL),
					// Record SQL method (SELECT, INSERT, UPDATE, DELETE, etc.)
					attribute.String("db.sql.method", string(method)),
				}

				return attrs
			}),
		)
		if err != nil {
			panic(err)
		}
	})
}

// GetDB returns sqlx.SqlConn with tracing enabled and caches the connection
func GetDB(dsn string) sqlx.SqlConn {
	initDriver()

	if val, ok := dbCache.Load(dsn); ok {
		return val.(sqlx.SqlConn)
	}

	conn := sqlx.NewSqlConn(driverName, dsn)
	dbCache.Store(dsn, conn)
	return conn
}

// buildCompleteSQL builds a complete SQL statement by replacing placeholders with actual values
func buildCompleteSQL(query string, args []driver.NamedValue) string {
	if len(args) == 0 {
		return query
	}

	// Convert driver.NamedValue to interface{} slice
	values := make([]interface{}, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}

	// Simple placeholder replacement for ? placeholders
	// This is a basic implementation - for production use, consider using a more robust SQL builder
	completeSQL := query
	placeholderRegex := regexp.MustCompile(`\?`)

	// Find all placeholder positions
	matches := placeholderRegex.FindAllStringIndex(completeSQL, -1)
	if len(matches) != len(values) {
		// If placeholder count doesn't match, return original query
		return query
	}

	// Replace placeholders from end to beginning to avoid index shifting
	for i := len(matches) - 1; i >= 0; i-- {
		if i < len(values) {
			start, end := matches[i][0], matches[i][1]
			var strValue string
			switch v := values[i].(type) {
			case string:
				strValue = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
			case nil:
				strValue = "NULL"
			default:
				strValue = fmt.Sprintf("%v", v)
			}
			completeSQL = completeSQL[:start] + strValue + completeSQL[end:]
		}
	}

	return completeSQL
}
