package diff

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// FieldNormalizer handles field filtering for JSON comparison.
// It always uses a hybrid approach:
// 1. If IncludedFields is set, extract only those fields (whitelist)
// 2. If IgnoredFields is set, remove those fields from the result (blacklist)
// 3. If both are set, include first, then exclude from included
// 4. If neither is set, return data unchanged
type FieldNormalizer struct {
	includedFields []string
	ignoredFields  []string
}

// NewFieldNormalizer creates a normalizer with the specified field filters.
func NewFieldNormalizer(includedFields, ignoredFields []string) *FieldNormalizer {
	return &FieldNormalizer{
		includedFields: includedFields,
		ignoredFields:  ignoredFields,
	}
}

// NormalizeBytes filters JSON fields using a hybrid approach:
// 1. Apply whitelist if includedFields is set (keep only these fields)
// 2. Apply blacklist if ignoredFields is set (remove these fields)
// This allows for flexible combinations like "include user object but exclude user.ssn"
func (fn *FieldNormalizer) NormalizeBytes(data []byte) ([]byte, error) {
	result := data

	// Step 1: Apply whitelist if specified
	if len(fn.includedFields) > 0 {
		var err error
		result, err = fn.applyWhitelist(result)
		if err != nil {
			return nil, err
		}
	}

	// Step 2: Apply blacklist if specified (on top of whitelist result)
	if len(fn.ignoredFields) > 0 {
		var err error
		result, err = fn.applyBlacklist(result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// applyWhitelist extracts only the specified fields (whitelist strategy).
// This is more performant when keeping few fields (5.9x faster in benchmarks).
func (fn *FieldNormalizer) applyWhitelist(data []byte) ([]byte, error) {
	// Start with empty JSON object
	result := []byte("{}")

	for _, path := range fn.includedFields {
		// Handle wildcard paths (e.g., "users.#.email")
		if strings.Contains(path, ".#.") {
			// Array wildcard - need to extract from all array elements
			result = fn.extractArrayField(data, result, path)
			continue
		}

		// Simple field path
		value := gjson.GetBytes(data, path)
		if !value.Exists() {
			continue
		}

		// Set the field in result
		var err error
		result, err = sjson.SetRawBytes(result, path, []byte(value.Raw))
		if err != nil {
			return nil, fmt.Errorf("failed to set field %s: %w", path, err)
		}
	}

	return result, nil
}

// applyBlacklist removes the specified fields (blacklist strategy).
// This is more flexible when keeping most fields.
func (fn *FieldNormalizer) applyBlacklist(data []byte) ([]byte, error) {
	result := data

	for _, path := range fn.ignoredFields {
		// Handle wildcard paths (e.g., "users.#.created_at")
		if strings.Contains(path, ".#.") {
			result = fn.deleteArrayField(result, path)
			continue
		}

		// Simple field path
		var err error
		result, err = sjson.DeleteBytes(result, path)
		if err != nil {
			// DeleteBytes returns error if path doesn't exist, which is fine
			// Just continue to next field
			continue
		}
	}

	return result, nil
}

// deleteArrayField handles wildcard deletion like "users.#.created_at".
// It removes the field from all array elements.
func (fn *FieldNormalizer) deleteArrayField(data []byte, path string) []byte {
	// Split path into: arrayPath + "#" + fieldPath
	// Example: "users.#.created_at" -> "users", "created_at"
	parts := strings.Split(path, ".#.")
	if len(parts) != 2 {
		return data
	}

	arrayPath := parts[0]
	fieldPath := parts[1]

	// Get the array
	arrayResult := gjson.GetBytes(data, arrayPath)
	if !arrayResult.Exists() || !arrayResult.IsArray() {
		return data
	}

	result := data

	// Delete field from each array element
	// We need to iterate and delete: arrayPath[i].fieldPath
	arrayResult.ForEach(func(key, value gjson.Result) bool {
		fullPath := fmt.Sprintf("%s.%s.%s", arrayPath, key.String(), fieldPath)
		result, _ = sjson.DeleteBytes(result, fullPath)
		return true // continue iteration
	})

	return result
}

// extractArrayField handles wildcard paths like "users.#.email".
// It extracts the field from all array elements and adds them to the result.
func (fn *FieldNormalizer) extractArrayField(data, result []byte, path string) []byte {
	// Split path into: arrayPath + "#" + fieldPath
	// Example: "users.#.email" -> "users", "email"
	parts := strings.Split(path, ".#.")
	if len(parts) != 2 {
		return result
	}

	arrayPath := parts[0]
	fieldPath := parts[1]

	// Get the array
	arrayResult := gjson.GetBytes(data, arrayPath)
	if !arrayResult.Exists() || !arrayResult.IsArray() {
		return result
	}

	// Extract field from each array element
	var extracted []any
	arrayResult.ForEach(func(key, value gjson.Result) bool {
		fieldValue := gjson.Get(value.Raw, fieldPath)
		if fieldValue.Exists() {
			extracted = append(extracted, fieldValue.Value())
		}
		return true // continue iteration
	})

	// Build the array structure in result
	// We need to reconstruct: arrayPath with extracted values
	for i, val := range extracted {
		fullPath := fmt.Sprintf("%s.%d.%s", arrayPath, i, fieldPath)
		result, _ = sjson.SetBytes(result, fullPath, val)
	}

	return result
}
