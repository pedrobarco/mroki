package traffictesting

import (
	"testing"
)

func TestNewHTTPMethod(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "valid GET lowercase",
			input:     "get",
			wantValue: "GET",
			wantErr:   false,
		},
		{
			name:      "valid POST uppercase",
			input:     "POST",
			wantValue: "POST",
			wantErr:   false,
		},
		{
			name:      "valid PUT mixed case",
			input:     "Put",
			wantValue: "PUT",
			wantErr:   false,
		},
		{
			name:      "valid DELETE",
			input:     "delete",
			wantValue: "DELETE",
			wantErr:   false,
		},
		{
			name:      "valid PATCH",
			input:     "patch",
			wantValue: "PATCH",
			wantErr:   false,
		},
		{
			name:      "valid HEAD",
			input:     "head",
			wantValue: "HEAD",
			wantErr:   false,
		},
		{
			name:      "valid OPTIONS",
			input:     "options",
			wantValue: "OPTIONS",
			wantErr:   false,
		},
		{
			name:      "whitespace trimmed",
			input:     "  GET  ",
			wantValue: "GET",
			wantErr:   false,
		},
		{
			name:      "empty string rejected",
			input:     "",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "whitespace only rejected",
			input:     "   ",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "invalid method INVALID",
			input:     "INVALID",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "invalid method CONNECT",
			input:     "CONNECT",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "invalid method TRACE",
			input:     "TRACE",
			wantValue: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPMethod(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewHTTPMethod() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewHTTPMethod() unexpected error: %v", err)
				return
			}

			if got.String() != tt.wantValue {
				t.Errorf("NewHTTPMethod() = %v, want %v", got.String(), tt.wantValue)
			}
		})
	}
}

func TestHTTPMethodFactories(t *testing.T) {
	tests := []struct {
		name    string
		factory func() HTTPMethod
		want    string
	}{
		{"GET factory", GET, "GET"},
		{"POST factory", POST, "POST"},
		{"PUT factory", PUT, "PUT"},
		{"DELETE factory", DELETE, "DELETE"},
		{"PATCH factory", PATCH, "PATCH"},
		{"HEAD factory", HEAD, "HEAD"},
		{"OPTIONS factory", OPTIONS, "OPTIONS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := tt.factory()
			if method.String() != tt.want {
				t.Errorf("%s() = %v, want %v", tt.name, method.String(), tt.want)
			}
		})
	}
}

func TestHTTPMethodPredicates(t *testing.T) {
	tests := []struct {
		method    HTTPMethod
		predicate func(HTTPMethod) bool
		name      string
		want      bool
	}{
		{GET(), func(m HTTPMethod) bool { return m.IsGET() }, "GET.IsGET()", true},
		{GET(), func(m HTTPMethod) bool { return m.IsPOST() }, "GET.IsPOST()", false},
		{POST(), func(m HTTPMethod) bool { return m.IsPOST() }, "POST.IsPOST()", true},
		{POST(), func(m HTTPMethod) bool { return m.IsGET() }, "POST.IsGET()", false},
		{PUT(), func(m HTTPMethod) bool { return m.IsPUT() }, "PUT.IsPUT()", true},
		{DELETE(), func(m HTTPMethod) bool { return m.IsDELETE() }, "DELETE.IsDELETE()", true},
		{PATCH(), func(m HTTPMethod) bool { return m.IsPATCH() }, "PATCH.IsPATCH()", true},
		{HEAD(), func(m HTTPMethod) bool { return m.IsHEAD() }, "HEAD.IsHEAD()", true},
		{OPTIONS(), func(m HTTPMethod) bool { return m.IsOPTIONS() }, "OPTIONS.IsOPTIONS()", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.predicate(tt.method); got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestHTTPMethodEquals(t *testing.T) {
	get1 := GET()
	get2 := GET()
	post := POST()

	t.Run("same values are equal", func(t *testing.T) {
		if !get1.Equals(get2) {
			t.Error("Two GET methods should be equal")
		}
	})

	t.Run("different values are not equal", func(t *testing.T) {
		if get1.Equals(post) {
			t.Error("GET and POST methods should not be equal")
		}
	})
}
