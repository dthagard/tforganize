package sort

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	gosort "sort"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Compare expression tokens and nested block structure, not whitespace or source
// ranges. Losing a new Terraform argument or changing a template must fail even
// when both versions still parse.
func terraformStructure(t *testing.T, source []byte, filename string) string {
	t.Helper()
	file, diagnostics := hclsyntax.ParseConfig(source, filename, hcl.InitialPos)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	var bodyStructure func(*hclsyntax.Body) string
	bodyStructure = func(body *hclsyntax.Body) string {
		parts := make([]string, 0, len(body.Attributes)+len(body.Blocks))
		for name, attribute := range body.Attributes {
			expression := attribute.Expr.Range().SliceBytes(source)
			tokens, diagnostics := hclsyntax.LexExpression(expression, filename, hcl.InitialPos)
			if diagnostics.HasErrors() {
				t.Fatal(diagnostics)
			}
			var encoded []string
			for _, token := range tokens {
				if token.Type != hclsyntax.TokenNewline && token.Type != hclsyntax.TokenEOF && token.Type != hclsyntax.TokenComment {
					encoded = append(encoded, token.Type.String()+":"+string(token.Bytes))
				}
			}
			value, err := json.Marshal(encoded)
			if err != nil {
				t.Fatal(err)
			}
			parts = append(parts, "attribute:"+name+":"+string(value))
		}
		for _, block := range body.Blocks {
			labels, err := json.Marshal(block.Labels)
			if err != nil {
				t.Fatal(err)
			}
			parts = append(parts, "block:"+block.Type+":"+string(labels)+":"+bodyStructure(block.Body))
		}
		gosort.Strings(parts)
		value, err := json.Marshal(parts)
		if err != nil {
			t.Fatal(err)
		}
		return string(value)
	}
	return bodyStructure(file.Body.(*hclsyntax.Body))
}

func TestTerraformLanguagePreservation(t *testing.T) {
	files, err := filepath.Glob("../../testdata/terraform/*/*.tf")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "../../testdata/terraform/latest/child/main.tf")
	for _, filename := range files {
		t.Run(filename, func(t *testing.T) {
			source, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			want := terraformStructure(t, source, filename)
			for _, settings := range []*Params{{}, {NoSortByType: true}, {RemoveComments: true}, {GroupByType: true}} {
				result, err := SortBytes(source, filename, settings)
				if err != nil {
					t.Fatalf("sorting with %+v: %v", settings, err)
				}
				if got := terraformStructure(t, result, filename); got != want {
					t.Fatalf("Terraform semantics changed with %+v\nwant: %s\ngot: %s", settings, want, got)
				}
				// Grouped byte output has no promised cross-file ordering.
				if settings.GroupByType {
					continue
				}
				again, err := SortBytes(result, filename, settings)
				if err != nil {
					t.Fatalf("sorting again with %+v: %v", settings, err)
				}
				if !bytes.Equal(result, again) {
					t.Fatalf("sorting is not idempotent with %+v\nfirst:\n%s\nsecond:\n%s", settings, result, again)
				}
			}
		})
	}
}
