package sort

import (
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func evaluatedPolicies(t *testing.T, source []byte) map[string]cty.Value {
	t.Helper()
	file, diagnostics := hclsyntax.ParseConfig(source, "main.tf", hcl.InitialPos)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	context := &hcl.EvalContext{
		Functions: map[string]function.Function{"jsonencode": stdlib.JSONEncodeFunc},
	}
	policies := make(map[string]cty.Value)
	for _, block := range file.Body.(*hclsyntax.Body).Blocks {
		if block.Type != "locals" {
			continue
		}
		for name, attribute := range block.Body.Attributes {
			value, diagnostics := attribute.Expr.Value(context)
			if diagnostics.HasErrors() {
				t.Fatal(diagnostics)
			}
			decoded, err := stdlib.JSONDecode(value)
			if err != nil {
				t.Fatalf("%s is not valid JSON: %v", name, err)
			}
			policies[name] = decoded
		}
	}
	return policies
}

func TestIAMJSONPreservation(t *testing.T) {
	source, err := os.ReadFile("../../testdata/terraform/iam/main.tf")
	if err != nil {
		t.Fatal(err)
	}
	want := evaluatedPolicies(t, source)
	for name, settings := range map[string]Params{
		"default":                  {},
		"alphabetical":             {NoSortByType: true},
		"remove comments":          {RemoveComments: true},
		"strip sections":           {StripSectionComments: true},
		"grouped":                  {GroupByType: true},
		"grouped without comments": {GroupByType: true, RemoveComments: true},
		"compact":                  {CompactEmptyBlocks: true},
		"combined":                 {RemoveComments: true, CompactEmptyBlocks: true, StripSectionComments: true},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := SortBytes(source, "main.tf", &settings)
			if err != nil {
				t.Fatal(err)
			}
			if settings.RemoveComments {
				tokens, diagnostics := hclsyntax.LexConfig(result, "main.tf", hcl.InitialPos)
				if diagnostics.HasErrors() {
					t.Fatal(diagnostics)
				}
				for _, token := range tokens {
					if token.Type == hclsyntax.TokenComment {
						t.Errorf("HCL comment retained: %s", token.Bytes)
					}
				}
			}
			got := evaluatedPolicies(t, result)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("evaluated policy JSON changed\nwant: %#v\ngot: %#v", want, got)
			}
			again, err := SortBytes(result, "main.tf", &settings)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(evaluatedPolicies(t, again), want) {
				t.Error("evaluated policy JSON changed on second sort")
			}
		})
	}
}
