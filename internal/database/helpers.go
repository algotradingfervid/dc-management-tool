package database

import (
	"context"
	"database/sql"
)

// ctx returns a background context for sqlc calls.
func ctx() context.Context {
	return context.Background()
}

// nullStringFromStr converts a string to sql.NullString (valid when non-empty).
func nullStringFromStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullFloat64 converts a float64 to sql.NullFloat64 (always valid).
func nullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: f, Valid: true}
}

// nullInt64FromInt converts an int to sql.NullInt64 (valid when non-zero).
func nullInt64FromInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

// nullInt64FromPtr converts a *int to sql.NullInt64.
func nullInt64FromPtr(p *int) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}
