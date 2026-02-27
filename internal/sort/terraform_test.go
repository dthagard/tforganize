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
		{
			name:      "resource block",
			blockType: "resource",
			wantPre:   []string{"count", "for_each", "provider"},
			wantPost:  []string{"provisioner", "lifecycle", "depends_on", "triggers_replace"},
		},
		{
			name:      "variable block",
			blockType: "variable",
			wantPre:   []string{"description", "type", "default", "nullable", "sensitive"},
			wantPost:  []string{"validation"},
		},
		{
			name:      "module block",
			blockType: "module",
			wantPre:   []string{"source", "version", "providers", "count", "for_each"},
			wantPost:  []string{"depends_on"},
		},
		{
			name:      "data block",
			blockType: "data",
			wantPre:   []string{"count", "for_each", "provider"},
			wantPost:  []string{"provisioner", "depends_on"},
		},
		{
			name:      "terraform block",
			blockType: "terraform",
			wantPre:   []string{"required_version", "required_providers"},
			wantPost:  []string{},
		},
		{
			name:      "dynamic block",
			blockType: "dynamic",
			wantPre:   []string{"for_each"},
			wantPost:  []string{},
		},
		{
			name:      "local block",
			blockType: "local",
			wantPre:   []string{},
			wantPost:  []string{},
		},
		{
			name:      "check block",
			blockType: "check",
			wantPre:   []string{"data"},
			wantPost:  []string{},
		},
		/*********************************************************************/
		// output block: description and value first, sensitive/precondition/depends_on last.
		/*********************************************************************/
		{
			name:      "output block",
			blockType: "output",
			wantPre:   []string{"description", "value"},
			wantPost:  []string{"sensitive", "precondition", "depends_on"},
		},
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
