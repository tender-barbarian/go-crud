package gocrud

import (
	"time"
)

// createdAtFields contains recognized field names for created_at timestamp
var createdAtFields = map[string]bool{
	"created_at": true,
	"createdat":  true,
	"created":    true,
}

// updatedAtFields contains recognized field names for updated_at timestamp
var updatedAtFields = map[string]bool{
	"updated_at":  true,
	"updatedat":   true,
	"updated":     true,
	"modified_at": true,
	"modifiedat":  true,
}

// setCreatedAt sets the created_at field to current time if it exists in the map
// and the underlying value is a *time.Time pointer
func setCreatedAt(m map[string]any, now time.Time) {
	for fieldName := range createdAtFields {
		if ptr, ok := m[fieldName]; ok {
			if timePtr, ok := ptr.(*time.Time); ok {
				*timePtr = now
			}
		}
	}
}

// setUpdatedAt sets the updated_at field to current time if it exists in the map
// and the underlying value is a *time.Time pointer
func setUpdatedAt(m map[string]any, now time.Time) {
	for fieldName := range updatedAtFields {
		if ptr, ok := m[fieldName]; ok {
			if timePtr, ok := ptr.(*time.Time); ok {
				*timePtr = now
			}
		}
	}
}

// removeCreatedAtFields removes created_at fields from the map
// Used during Update to prevent overwriting the original creation timestamp
func removeCreatedAtFields(m map[string]any) {
	for fieldName := range createdAtFields {
		delete(m, fieldName)
	}
}
