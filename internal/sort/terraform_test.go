package sort

import "testing"

func TestGetBlockTypePriority(t *testing.T) {
	tests := []struct {
		blockType string
		want      int
	}{
		{"terraform", 1},
		{"variable", 2},
		{"locals", 3},
		{"data", 4},
		{"resource", 5},
		{"module", 6},
		{"import", 7},
		{"moved", 8},
		{"removed", 9},
		{"check", 10},
		{"output", 11},
		{"unknown_type", defaultBlockTypePriority},
	}

	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			got := getBlockTypePriority(tt.blockType)
			if got != tt.want {
				t.Errorf("getBlockTypePriority(%q) = %d, want %d", tt.blockType, got, tt.want)
			}
		})
	}
}
