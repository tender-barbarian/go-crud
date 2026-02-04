package gocrud

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// NullTime represents a time.Time that can scan from both native time types
// (PostgreSQL) and string representations (SQLite).
type NullTime struct {
	Time  time.Time
	Valid bool
}

// Common time formats for parsing string timestamps
var timeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Scan implements the sql.Scanner interface.
// It handles time.Time, string, []byte, and nil values.
func (nt *NullTime) Scan(value any) error {
	if value == nil {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = v, true
		return nil
	case string:
		return nt.parseString(v)
	case []byte:
		return nt.parseString(string(v))
	default:
		return fmt.Errorf("cannot scan type %T into NullTime", value)
	}
}

func (nt *NullTime) parseString(s string) error {
	if s == "" {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}

	for _, format := range timeFormats {
		if t, err := time.Parse(format, s); err == nil {
			nt.Time, nt.Valid = t, true
			return nil
		}
	}

	return fmt.Errorf("cannot parse %q as time", s)
}

// Value implements the driver.Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}
