package sqlite

import (
	"testing"
)

func TestConvertWebsearchToFTS5(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple term",
			input:    "dragon",
			expected: "dragon",
		},
		{
			name:     "multiple terms",
			input:    "red dragon",
			expected: "red AND dragon",
		},
		{
			name:     "explicit AND",
			input:    "dragon AND sword",
			expected: "dragon AND sword",
		},
		{
			name:     "explicit OR",
			input:    "dragon OR sword",
			expected: "dragon OR sword",
		},
		{
			name:     "negation",
			input:    "dragon -fire",
			expected: "dragon AND NOT fire",
		},
		{
			name:     "phrase",
			input:    `"red dragon"`,
			expected: `"red dragon"`,
		},
		{
			name:     "phrase with other term",
			input:    `"red dragon" castle`,
			expected: `"red dragon" AND castle`,
		},
		{
			name:     "prefix search",
			input:    "dragon*",
			expected: "dragon*",
		},
		{
			name:     "complex query",
			input:    `"red dragon" -fire castle OR tower`,
			expected: `"red dragon" AND NOT fire AND castle OR tower`,
		},
		{
			name:     "NOT operator",
			input:    "dragon NOT fire",
			expected: "dragon NOT fire",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertWebsearchToFTS5(tt.input)
			if result != tt.expected {
				t.Errorf("convertWebsearchToFTS5(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
