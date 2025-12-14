package confluence

import (
	"reflect"
	"testing"
)

func TestHashString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{name: "Empty_String", input: ""},
		{name: "Simple_String", input: "hello"},
		{name: "Unicode", input: "日本語"},
		{name: "Long_String", input: "This is a longer string that should still produce a consistent hash value"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got1 := hashString(tt.input)
			got2 := hashString(tt.input)

			// Check consistency
			if got1 != got2 {
				t.Errorf("hashString not consistent: %q != %q", got1, got2)
			}

			// Check format (16 hex chars)
			if len(got1) != 16 {
				t.Errorf("hashString length = %d, want 16", len(got1))
			}

			// Check it's valid hex
			for _, c := range got1 {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("hashString contains invalid hex char: %c", c)
				}
			}
		})
	}
}

func TestHashString_Uniqueness(t *testing.T) {
	t.Parallel()
	values := []string{"a", "b", "hello", "Hello", "world", ""}
	hashes := make(map[string]string)

	for _, v := range values {
		hash := hashString(v)
		if existing, ok := hashes[hash]; ok && existing != v {
			t.Errorf("collision: %q and %q have same hash %q", v, existing, hash)
		}
		hashes[hash] = v
	}
}

func TestComputePageChecksums(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		page       map[string]any
		wantFields []string
	}{
		{
			name:       "Empty_Page",
			page:       map[string]any{},
			wantFields: []string{},
		},
		{
			name: "Title_Only",
			page: map[string]any{
				"title": "Test Page",
			},
			wantFields: []string{"title"},
		},
		{
			name: "Title_And_Version",
			page: map[string]any{
				"title": "Test Page",
				"version": map[string]any{
					"number": float64(1),
				},
			},
			wantFields: []string{"title", "version"},
		},
		{
			name: "Full_Page",
			page: map[string]any{
				"title": "Test Page",
				"body": map[string]any{
					"atlas_doc_format": map[string]any{
						"value": `{"type":"doc","version":1}`,
					},
				},
				"version": map[string]any{
					"number": float64(5),
				},
			},
			wantFields: []string{"title", "body", "version"},
		},
		{
			name: "Body_Without_ADF",
			page: map[string]any{
				"title": "Test Page",
				"body": map[string]any{
					"storage": map[string]any{
						"value": "<p>HTML content</p>",
					},
				},
			},
			wantFields: []string{"title"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputePageChecksums(tt.page)

			// Check expected fields are present
			for _, field := range tt.wantFields {
				if _, ok := got[field]; !ok {
					t.Errorf("missing checksum for field %q", field)
				}
			}

			// Check no extra fields
			if len(got) != len(tt.wantFields) {
				t.Errorf("got %d checksums, want %d", len(got), len(tt.wantFields))
			}
		})
	}
}

func TestComputePageChecksums_Consistency(t *testing.T) {
	t.Parallel()
	page := map[string]any{
		"title": "Test Page",
		"body": map[string]any{
			"atlas_doc_format": map[string]any{
				"value": `{"type":"doc"}`,
			},
		},
		"version": map[string]any{
			"number": float64(3),
		},
	}

	got1 := ComputePageChecksums(page)
	got2 := ComputePageChecksums(page)

	if !reflect.DeepEqual(got1, got2) {
		t.Error("ComputePageChecksums not consistent across calls")
	}
}

func TestFormatChecksums(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		checksums map[string]string
		wantStart string
		wantEnd   string
		wantLines int
	}{
		{
			name:      "Empty_Checksums",
			checksums: map[string]string{},
			wantStart: "__CHECKSUMS__",
			wantEnd:   "__END_CHECKSUMS__",
			wantLines: 2, // header + footer only
		},
		{
			name: "Single_Field",
			checksums: map[string]string{
				"title": "abc123def456",
			},
			wantStart: "__CHECKSUMS__",
			wantEnd:   "__END_CHECKSUMS__",
			wantLines: 3,
		},
		{
			name: "All_Fields",
			checksums: map[string]string{
				"title":   "aaa111bbb222",
				"body":    "ccc333ddd444",
				"version": "eee555fff666",
			},
			wantStart: "__CHECKSUMS__",
			wantEnd:   "__END_CHECKSUMS__",
			wantLines: 5,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatChecksums(tt.checksums)

			if got[:len(tt.wantStart)] != tt.wantStart {
				t.Errorf("FormatChecksums() should start with %q, got %q", tt.wantStart, got[:len(tt.wantStart)])
			}

			if got[len(got)-len(tt.wantEnd):] != tt.wantEnd {
				t.Errorf("FormatChecksums() should end with %q", tt.wantEnd)
			}

			// Check field contents
			for field, hash := range tt.checksums {
				expected := field + "=" + hash
				if !contains(got, expected) {
					t.Errorf("FormatChecksums() missing %q", expected)
				}
			}
		})
	}
}

func TestFormatChecksums_FieldOrder(t *testing.T) {
	t.Parallel()
	// ChecksumFields defines the expected order: title, body, version
	checksums := map[string]string{
		"version": "111",
		"title":   "222",
		"body":    "333",
	}

	got := FormatChecksums(checksums)

	// title should come before body, body before version
	titleIdx := indexOf(got, "title=")
	bodyIdx := indexOf(got, "body=")
	versionIdx := indexOf(got, "version=")

	if titleIdx > bodyIdx {
		t.Error("title should come before body in output")
	}
	if bodyIdx > versionIdx {
		t.Error("body should come before version in output")
	}
}

func TestChecksumFields(t *testing.T) {
	t.Parallel()
	// Verify the ChecksumFields slice contains expected fields
	expected := []string{"title", "body", "version"}
	if !reflect.DeepEqual(ChecksumFields, expected) {
		t.Errorf("ChecksumFields = %v, want %v", ChecksumFields, expected)
	}
}

// Helper functions

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
