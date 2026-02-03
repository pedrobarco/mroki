package diff

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestFieldNormalizer_Whitelist(t *testing.T) {
	tests := []struct {
		name           string
		includedFields []string
		input          string
		wantFields     []string // Fields that should exist
		dontWantFields []string // Fields that should NOT exist
	}{
		{
			name:           "include single top-level field",
			includedFields: []string{"name"},
			input:          `{"name":"John","age":30,"email":"john@example.com"}`,
			wantFields:     []string{"name"},
			dontWantFields: []string{"age", "email"},
		},
		{
			name:           "include multiple top-level fields",
			includedFields: []string{"name", "email"},
			input:          `{"name":"John","age":30,"email":"john@example.com"}`,
			wantFields:     []string{"name", "email"},
			dontWantFields: []string{"age"},
		},
		{
			name:           "include nested field",
			includedFields: []string{"user.name"},
			input:          `{"user":{"name":"John","age":30},"timestamp":"2024-01-01"}`,
			wantFields:     []string{"user.name"},
			dontWantFields: []string{"user.age", "timestamp"},
		},
		{
			name:           "include array wildcard field",
			includedFields: []string{"users.#.email"},
			input:          `{"users":[{"name":"John","email":"john@example.com"},{"name":"Jane","email":"jane@example.com"}]}`,
			wantFields:     []string{"users.0.email", "users.1.email"},
			dontWantFields: []string{"users.0.name", "users.1.name"},
		},
		{
			name:           "non-existent field",
			includedFields: []string{"nonexistent"},
			input:          `{"name":"John","age":30}`,
			wantFields:     []string{},
			dontWantFields: []string{"name", "age"},
		},
		{
			name:           "empty included fields",
			includedFields: []string{},
			input:          `{"name":"John","age":30}`,
			wantFields:     []string{"name", "age"},
			dontWantFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizer := NewFieldNormalizer(tt.includedFields, nil)
			result, err := normalizer.NormalizeBytes([]byte(tt.input))
			if err != nil {
				t.Fatalf("NormalizeBytes() error = %v", err)
			}

			// Check that wanted fields exist
			for _, field := range tt.wantFields {
				value := gjson.GetBytes(result, field)
				if !value.Exists() {
					t.Errorf("expected field %q to exist in result, but it doesn't. Result: %s", field, string(result))
				}
			}

			// Check that unwanted fields don't exist
			for _, field := range tt.dontWantFields {
				value := gjson.GetBytes(result, field)
				if value.Exists() {
					t.Errorf("expected field %q to NOT exist in result, but it does. Result: %s", field, string(result))
				}
			}
		})
	}
}

func TestFieldNormalizer_Blacklist(t *testing.T) {
	tests := []struct {
		name           string
		ignoredFields  []string
		input          string
		wantFields     []string // Fields that should exist
		dontWantFields []string // Fields that should NOT exist
	}{
		{
			name:           "ignore single top-level field",
			ignoredFields:  []string{"timestamp"},
			input:          `{"name":"John","age":30,"timestamp":"2024-01-01"}`,
			wantFields:     []string{"name", "age"},
			dontWantFields: []string{"timestamp"},
		},
		{
			name:           "ignore multiple top-level fields",
			ignoredFields:  []string{"timestamp", "request_id"},
			input:          `{"name":"John","timestamp":"2024-01-01","request_id":"abc123"}`,
			wantFields:     []string{"name"},
			dontWantFields: []string{"timestamp", "request_id"},
		},
		{
			name:           "ignore nested field",
			ignoredFields:  []string{"metadata.timestamp"},
			input:          `{"name":"John","metadata":{"timestamp":"2024-01-01","version":"v1"}}`,
			wantFields:     []string{"name", "metadata.version"},
			dontWantFields: []string{"metadata.timestamp"},
		},
		{
			name:           "ignore array wildcard field",
			ignoredFields:  []string{"users.#.created_at"},
			input:          `{"users":[{"name":"John","created_at":"2024-01-01"},{"name":"Jane","created_at":"2024-01-02"}]}`,
			wantFields:     []string{"users.0.name", "users.1.name"},
			dontWantFields: []string{"users.0.created_at", "users.1.created_at"},
		},
		{
			name:           "ignore non-existent field",
			ignoredFields:  []string{"nonexistent"},
			input:          `{"name":"John","age":30}`,
			wantFields:     []string{"name", "age"},
			dontWantFields: []string{},
		},
		{
			name:           "empty ignored fields",
			ignoredFields:  []string{},
			input:          `{"name":"John","age":30}`,
			wantFields:     []string{"name", "age"},
			dontWantFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizer := NewFieldNormalizer(nil, tt.ignoredFields)
			result, err := normalizer.NormalizeBytes([]byte(tt.input))
			if err != nil {
				t.Fatalf("NormalizeBytes() error = %v", err)
			}

			// Check that wanted fields exist
			for _, field := range tt.wantFields {
				value := gjson.GetBytes(result, field)
				if !value.Exists() {
					t.Errorf("expected field %q to exist in result, but it doesn't. Result: %s", field, string(result))
				}
			}

			// Check that unwanted fields don't exist
			for _, field := range tt.dontWantFields {
				value := gjson.GetBytes(result, field)
				if value.Exists() {
					t.Errorf("expected field %q to NOT exist in result, but it does. Result: %s", field, string(result))
				}
			}
		})
	}
}

func TestFieldNormalizer_HybridStrategy(t *testing.T) {
	// When both includedFields and ignoredFields are set, it's a hybrid strategy:
	// 1. First include only the specified fields (whitelist)
	// 2. Then exclude from the included fields (blacklist on top)
	input := `{"name":"John","age":30,"email":"john@example.com"}`

	normalizer := NewFieldNormalizer(
		[]string{"name", "email"}, // whitelist: include name and email
		[]string{"email"},         // blacklist: then exclude email
	)

	result, err := normalizer.NormalizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("NormalizeBytes() error = %v", err)
	}

	// name should exist (included and not excluded)
	nameValue := gjson.GetBytes(result, "name")
	if !nameValue.Exists() {
		t.Errorf("expected name to exist (included), but it doesn't. Result: %s", string(result))
	}

	// email should NOT exist (included but then excluded)
	emailValue := gjson.GetBytes(result, "email")
	if emailValue.Exists() {
		t.Errorf("expected email to NOT exist (excluded after include), but it does. Result: %s", string(result))
	}

	// age should not exist (not included in whitelist)
	ageValue := gjson.GetBytes(result, "age")
	if ageValue.Exists() {
		t.Errorf("expected age to NOT exist (not included), but it does. Result: %s", string(result))
	}
}

func TestFieldNormalizer_HybridStrategy_NestedObjects(t *testing.T) {
	// Hybrid: include entire "user" object, but exclude sensitive field
	input := `{"user":{"name":"John","email":"john@example.com","ssn":"123-45-6789"},"timestamp":"2024-01-01"}`

	normalizer := NewFieldNormalizer(
		[]string{"user"},     // include entire user object
		[]string{"user.ssn"}, // but exclude sensitive SSN
	)

	result, err := normalizer.NormalizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("NormalizeBytes() error = %v", err)
	}

	// user.name and user.email should exist
	if !gjson.GetBytes(result, "user.name").Exists() {
		t.Errorf("expected user.name to exist. Result: %s", string(result))
	}
	if !gjson.GetBytes(result, "user.email").Exists() {
		t.Errorf("expected user.email to exist. Result: %s", string(result))
	}

	// user.ssn should NOT exist (excluded)
	if gjson.GetBytes(result, "user.ssn").Exists() {
		t.Errorf("expected user.ssn to NOT exist (excluded). Result: %s", string(result))
	}

	// timestamp should NOT exist (not included)
	if gjson.GetBytes(result, "timestamp").Exists() {
		t.Errorf("expected timestamp to NOT exist (not included). Result: %s", string(result))
	}
}

func TestFieldNormalizer_HybridStrategy_ArrayWildcard(t *testing.T) {
	// Hybrid: include users array, but exclude timestamps from each user
	input := `{"users":[{"name":"John","email":"john@example.com","created_at":"2024-01-01"},{"name":"Jane","email":"jane@example.com","created_at":"2024-01-02"}],"metadata":{"version":"v1"}}`

	normalizer := NewFieldNormalizer(
		[]string{"users"},              // include entire users array
		[]string{"users.#.created_at"}, // but exclude created_at from each user
	)

	result, err := normalizer.NormalizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("NormalizeBytes() error = %v", err)
	}

	// users should exist with name and email
	if !gjson.GetBytes(result, "users.0.name").Exists() {
		t.Errorf("expected users.0.name to exist. Result: %s", string(result))
	}
	if !gjson.GetBytes(result, "users.0.email").Exists() {
		t.Errorf("expected users.0.email to exist. Result: %s", string(result))
	}

	// created_at should NOT exist
	if gjson.GetBytes(result, "users.0.created_at").Exists() {
		t.Errorf("expected users.0.created_at to NOT exist. Result: %s", string(result))
	}
	if gjson.GetBytes(result, "users.1.created_at").Exists() {
		t.Errorf("expected users.1.created_at to NOT exist. Result: %s", string(result))
	}

	// metadata should NOT exist (not included)
	if gjson.GetBytes(result, "metadata").Exists() {
		t.Errorf("expected metadata to NOT exist (not included). Result: %s", string(result))
	}
}

func TestFieldNormalizer_NoFiltering(t *testing.T) {
	input := `{"name":"John","age":30,"email":"john@example.com"}`

	normalizer := NewFieldNormalizer(nil, nil)
	result, err := normalizer.NormalizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("NormalizeBytes() error = %v", err)
	}

	// Result should be identical to input when no filtering
	if string(result) != input {
		t.Errorf("expected no changes, but got: %s", string(result))
	}
}

func TestFieldNormalizer_ComplexNesting(t *testing.T) {
	input := `{
		"user": {
			"name": "John",
			"profile": {
				"age": 30,
				"email": "john@example.com"
			}
		},
		"timestamp": "2024-01-01"
	}`

	normalizer := NewFieldNormalizer(
		[]string{"user.profile.email", "user.name"},
		nil,
	)

	result, err := normalizer.NormalizeBytes([]byte(input))
	if err != nil {
		t.Fatalf("NormalizeBytes() error = %v", err)
	}

	// Check wanted fields exist
	if !gjson.GetBytes(result, "user.profile.email").Exists() {
		t.Errorf("expected user.profile.email to exist. Result: %s", string(result))
	}
	if !gjson.GetBytes(result, "user.name").Exists() {
		t.Errorf("expected user.name to exist. Result: %s", string(result))
	}

	// Check unwanted fields don't exist
	if gjson.GetBytes(result, "user.profile.age").Exists() {
		t.Errorf("expected user.profile.age to NOT exist. Result: %s", string(result))
	}
	if gjson.GetBytes(result, "timestamp").Exists() {
		t.Errorf("expected timestamp to NOT exist. Result: %s", string(result))
	}
}

// Benchmark tests
func BenchmarkFieldNormalizer_Whitelist(b *testing.B) {
	input := []byte(`{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","phone":"555-1234","country":"USA","timestamp":"2024-01-01","request_id":"abc123"}`)
	normalizer := NewFieldNormalizer([]string{"name", "email"}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.NormalizeBytes(input)
	}
}

func BenchmarkFieldNormalizer_Blacklist(b *testing.B) {
	input := []byte(`{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","phone":"555-1234","country":"USA","timestamp":"2024-01-01","request_id":"abc123"}`)
	normalizer := NewFieldNormalizer(nil, []string{"timestamp", "request_id"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.NormalizeBytes(input)
	}
}

func BenchmarkFieldNormalizer_NoFiltering(b *testing.B) {
	input := []byte(`{"name":"John","age":30,"email":"john@example.com","address":"123 Main St","phone":"555-1234","country":"USA","timestamp":"2024-01-01","request_id":"abc123"}`)
	normalizer := NewFieldNormalizer(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.NormalizeBytes(input)
	}
}
