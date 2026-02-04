package gocrud

import (
	"testing"
	"time"
)

func TestNullTime_Scan_TimeValue(t *testing.T) {
	now := time.Now()
	var nt NullTime

	err := nt.Scan(now)
	if err != nil {
		t.Fatal(err)
	}

	if !nt.Valid {
		t.Error("expected Valid to be true")
	}
	if !nt.Time.Equal(now) {
		t.Errorf("expected %v, got %v", now, nt.Time)
	}
}

func TestNullTime_Scan_String(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "RFC3339",
			input:    "2024-01-15T10:30:00Z",
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "SQLite format",
			input:    "2024-01-15 10:30:00",
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "Date only",
			input:    "2024-01-15",
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var nt NullTime
			err := nt.Scan(tc.input)
			if err != nil {
				t.Fatal(err)
			}

			if !nt.Valid {
				t.Error("expected Valid to be true")
			}
			if !nt.Time.Equal(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, nt.Time)
			}
		})
	}
}

func TestNullTime_Scan_Bytes(t *testing.T) {
	var nt NullTime
	err := nt.Scan([]byte("2024-01-15 10:30:00"))
	if err != nil {
		t.Fatal(err)
	}

	if !nt.Valid {
		t.Error("expected Valid to be true")
	}
}

func TestNullTime_Scan_Nil(t *testing.T) {
	var nt NullTime
	nt.Valid = true // Set to true first

	err := nt.Scan(nil)
	if err != nil {
		t.Fatal(err)
	}

	if nt.Valid {
		t.Error("expected Valid to be false after scanning nil")
	}
}

func TestNullTime_Scan_EmptyString(t *testing.T) {
	var nt NullTime

	err := nt.Scan("")
	if err != nil {
		t.Fatal(err)
	}

	if nt.Valid {
		t.Error("expected Valid to be false for empty string")
	}
}

func TestNullTime_Scan_InvalidString(t *testing.T) {
	var nt NullTime

	err := nt.Scan("not a time")
	if err == nil {
		t.Error("expected error for invalid time string")
	}
}

func TestNullTime_Scan_UnsupportedType(t *testing.T) {
	var nt NullTime

	err := nt.Scan(12345)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestNullTime_Value_Valid(t *testing.T) {
	now := time.Now()
	nt := NullTime{Time: now, Valid: true}

	val, err := nt.Value()
	if err != nil {
		t.Fatal(err)
	}

	if val != now {
		t.Errorf("expected %v, got %v", now, val)
	}
}

func TestNullTime_Value_Invalid(t *testing.T) {
	nt := NullTime{Valid: false}

	val, err := nt.Value()
	if err != nil {
		t.Fatal(err)
	}

	if val != nil {
		t.Error("expected nil for invalid NullTime")
	}
}
