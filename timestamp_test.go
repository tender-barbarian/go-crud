package gocrud

import (
	"testing"
	"time"
)

func TestSetCreatedAt(t *testing.T) {
	testCases := []struct {
		name      string
		fieldName string
	}{
		{"created_at field", "created_at"},
		{"createdat field", "createdat"},
		{"created field", "created"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var timestamp time.Time
			m := map[string]any{
				tc.fieldName: &timestamp,
			}

			now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			setCreatedAt(m, now)

			if !timestamp.Equal(now) {
				t.Errorf("expected %v, got %v", now, timestamp)
			}
		})
	}
}

func TestSetCreatedAt_NonTimeField(t *testing.T) {
	var str string
	m := map[string]any{
		"created_at": &str,
	}

	now := time.Now()
	setCreatedAt(m, now)

	if str != "" {
		t.Error("non-time field should not be modified")
	}
}

func TestSetCreatedAt_FieldNotPresent(_ *testing.T) {
	m := map[string]any{
		"name": new(string),
	}

	now := time.Now()
	setCreatedAt(m, now) // Should not panic
}

func TestSetUpdatedAt(t *testing.T) {
	testCases := []struct {
		name      string
		fieldName string
	}{
		{"updated_at field", "updated_at"},
		{"updatedat field", "updatedat"},
		{"updated field", "updated"},
		{"modified_at field", "modified_at"},
		{"modifiedat field", "modifiedat"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var timestamp time.Time
			m := map[string]any{
				tc.fieldName: &timestamp,
			}

			now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			setUpdatedAt(m, now)

			if !timestamp.Equal(now) {
				t.Errorf("expected %v, got %v", now, timestamp)
			}
		})
	}
}

func TestSetUpdatedAt_NonTimeField(t *testing.T) {
	var str string
	m := map[string]any{
		"updated_at": &str,
	}

	now := time.Now()
	setUpdatedAt(m, now)

	if str != "" {
		t.Error("non-time field should not be modified")
	}
}

func TestRemoveCreatedAtFields(t *testing.T) {
	m := map[string]any{
		"created_at": &time.Time{},
		"createdat":  &time.Time{},
		"created":    &time.Time{},
		"name":       new(string),
		"updated_at": &time.Time{},
	}

	removeCreatedAtFields(m)

	for _, field := range []string{"created_at", "createdat", "created"} {
		if _, exists := m[field]; exists {
			t.Errorf("field %s should have been removed", field)
		}
	}

	if _, exists := m["name"]; !exists {
		t.Error("name field should not be removed")
	}
	if _, exists := m["updated_at"]; !exists {
		t.Error("updated_at field should not be removed")
	}
}
