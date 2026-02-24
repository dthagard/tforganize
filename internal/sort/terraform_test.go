package sort

import (
	"reflect"
	"testing"

	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestGetMetaArguments(t *testing.T) {
	tests := []struct {
		name      string
		blockType string
		wantPre   []string
		wantPost  []string
	}{
		/*********************************************************************/
		// moved block: requires from and to (in that order), no post args.
		/*********************************************************************/
		{
			name:      "moved block",
			blockType: "moved",
			wantPre:   []string{"from", "to"},
			wantPost:  []string{},
		},
		/*********************************************************************/
		// removed block: from attribute first, lifecycle block last.
		/*********************************************************************/
		{
			name:      "removed block",
			blockType: "removed",
			wantPre:   []string{"from"},
			wantPost:  []string{"lifecycle"},
		},
		/*********************************************************************/
		// import block: to and id are required before optional provider.
		/*********************************************************************/
		{
			name:      "import block",
			blockType: "import",
			wantPre:   []string{"to", "id", "provider"},
			wantPost:  []string{},
		},
		/*********************************************************************/
		// Unknown block type falls back to the default empty slices.
		/*********************************************************************/
		{
			name:      "unknown block type falls back to default",
			blockType: "unknown_block_type",
			wantPre:   []string{},
			wantPost:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &hclsyntax.Block{Type: tt.blockType}
			got := getMetaArguments(block)

			if !reflect.DeepEqual(got[0], tt.wantPre) {
				t.Errorf("getMetaArguments(%q) pre = %v, want %v", tt.blockType, got[0], tt.wantPre)
			}
			if !reflect.DeepEqual(got[1], tt.wantPost) {
				t.Errorf("getMetaArguments(%q) post = %v, want %v", tt.blockType, got[1], tt.wantPost)
			}
		})
	}
}
