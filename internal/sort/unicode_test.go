package sort

import "testing"

func TestUnicodeCommentRemovalPreservesExpressions(t *testing.T) {
	source := []byte("locals {\n  世界 = \"e\u0301 and 世界\" # remove\n  values = [\n    \"e\u0301\", \"世界\"] # remove\n}\n")
	result, err := SortBytes(source, "unicode.tf", &Params{RemoveComments: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := terraformStructure(t, result, "unicode.tf"), terraformStructure(t, source, "unicode.tf"); got != want {
		t.Fatalf("Unicode expressions changed:\n%s", result)
	}
}
