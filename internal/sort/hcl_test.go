package sort

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"os"

	hcl "github.com/hashicorp/hcl/v2"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/afero"
)

func TestIsSortable(t *testing.T) {
	/*********************************************************************/
	// Happy path test for isSortable() with a sortable file
	/*********************************************************************/

	t.Run("sortable file", func(t *testing.T) {
		testDir := t.TempDir()

		// Create a sortable file
		sortable, err := os.Create(filepath.Join(testDir, "foo.tf"))
		if err != nil {
			t.Fatalf("could not create sortable file: %v", err)
		}
		defer sortable.Close()

		// Get sortable file info
		sortableStat, err := sortable.Stat()
		if err != nil {
			t.Fatalf("could not stat sortable file: %v", err)
		}

		// Test sortable file
		expected := true
		if result := isSortable(sortableStat); result != expected {
			t.Errorf("isSortable() returned %t, expected %t", result, expected)
		}
	})

	/*********************************************************************/
	// Happy path test for isSortable() with a non-sortable file
	/*********************************************************************/

	t.Run("non-sortable file", func(t *testing.T) {
		testDir := t.TempDir()

		// Create a non-sortable file
		unsortable, err := os.Create(filepath.Join(testDir, "main.txt"))
		if err != nil {
			t.Fatalf("could not create non-sortable file: %v", err)
		}
		defer unsortable.Close()

		// Get sortable file info
		nonSortableStat, err := unsortable.Stat()
		if err != nil {
			t.Fatalf("could not stat non-sortable file: %v", err)
		}

		// Test sortable file
		expected := false
		if result := isSortable(nonSortableStat); result != expected {
			t.Errorf("isSortable() returned %t, expected %t", result, expected)
		}
	})
}

func TestIsStartOfComment(t *testing.T) {

	/*********************************************************************/
	// Happy path test for isStartOfComment() with a comment string
	/*********************************************************************/

	t.Run("comment string", func(t *testing.T) {
		testData := []string{
			"/***************************************************************",
			"/**",
			"/*",
			"#",
			"//",
			"#################################################################",
			"  #",
		}

		for _, s := range testData {
			result := isStartOfComment(s)
			expected := true
			if result != expected {
				t.Errorf("isStartOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})

	/*********************************************************************/
	// Sad path test for isStartOfComment() with non-comment string
	/*********************************************************************/

	t.Run("not comment string", func(t *testing.T) {
		testData := []string{
			"***************************************************************",
			"/",
			"   ",
			"",
			"this is not a comment",
			"also not a comment ## foo",
			"  foobar",
		}

		for _, s := range testData {
			result := isStartOfComment(s)
			expected := false
			if result != expected {
				t.Errorf("isStartOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})
}

func TestIsEndOfComment(t *testing.T) {

	/*********************************************************************/
	// Happy path test for isEndOfComment() with a closing comment string
	/*********************************************************************/

	t.Run("comment string", func(t *testing.T) {
		testData := []string{
			"***************************************************************/",
			"**/",
			"*/",
			"  */",
		}

		for _, s := range testData {
			result := isEndOfComment(s)
			expected := true
			if result != expected {
				t.Errorf("isEndOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})

	/*********************************************************************/
	// Sad path test for isEndOfComment() with non-comment string
	/*********************************************************************/

	t.Run("not comment string", func(t *testing.T) {
		testData := []string{
			"***************************************************************",
			"/",
			"   ",
			"",
			"#",
			"this is not a comment",
			"  foobar",
		}

		for _, s := range testData {
			result := isEndOfComment(s)
			expected := false
			if result != expected {
				t.Errorf("isEndOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})
}

func TestBlockListSorterLess(t *testing.T) {

	/*********************************************************************/
	// Different types: with sortByType enabled, blocks are ordered by
	// logical priority (data=4 < resource=5).
	/*********************************************************************/

	t.Run("different types ordered by priority", func(t *testing.T) {
		bs := BlockListSorter{
			blocks: []*hclsyntax.Block{
				{Type: "resource"},
				{Type: "data"},
			},
			sortByType: true,
		}
		// data (4) < resource (5), so Less(1,0) should be true
		if !bs.Less(1, 0) {
			t.Error("data should come before resource")
		}
		if bs.Less(0, 1) {
			t.Error("resource should not come before data")
		}
	})

	/*********************************************************************/
	// Different types with sortByType disabled: alphabetical ordering.
	/*********************************************************************/

	t.Run("different types ordered alphabetically when sortByType false", func(t *testing.T) {
		bs := BlockListSorter{
			blocks: []*hclsyntax.Block{
				{Type: "resource"},
				{Type: "data"},
			},
			sortByType: false,
		}
		// "data" < "resource" alphabetically
		if !bs.Less(1, 0) {
			t.Error("data should come before resource alphabetically")
		}
		if bs.Less(0, 1) {
			t.Error("resource should not come before data alphabetically")
		}
	})

	/*********************************************************************/
	// Full priority chain: terraform < variable < locals < data <
	// resource < module < import < moved < removed < check < output.
	/*********************************************************************/

	t.Run("full priority chain", func(t *testing.T) {
		orderedTypes := []string{
			"terraform", "variable", "locals", "data", "resource",
			"module", "import", "moved", "removed", "check", "output",
		}
		var blocks []*hclsyntax.Block
		for _, typ := range orderedTypes {
			blocks = append(blocks, &hclsyntax.Block{Type: typ})
		}
		bs := BlockListSorter{blocks: blocks, sortByType: true}

		for i := 0; i < len(orderedTypes)-1; i++ {
			if !bs.Less(i, i+1) {
				t.Errorf("%s (priority %d) should come before %s (priority %d)",
					orderedTypes[i], getBlockTypePriority(orderedTypes[i]),
					orderedTypes[i+1], getBlockTypePriority(orderedTypes[i+1]))
			}
		}
	})

	/*********************************************************************/
	// Same type, no labels on either block: len(0) < len(0) is false.
	/*********************************************************************/

	t.Run("same type no labels returns false", func(t *testing.T) {
		bs := BlockListSorter{
			blocks: []*hclsyntax.Block{
				{Type: "terraform"},
				{Type: "terraform"},
			},
			sortByType: true,
		}
		if bs.Less(0, 1) {
			t.Error("equal no-label blocks should not be Less")
		}
	})

	/*********************************************************************/
	// Same type, first label differs: label ordering determines result.
	/*********************************************************************/

	t.Run("same type first label differs", func(t *testing.T) {
		bs := BlockListSorter{
			blocks: []*hclsyntax.Block{
				{Type: "resource", Labels: []string{"aws_s3_bucket", "beta"}},
				{Type: "resource", Labels: []string{"aws_s3_bucket", "alpha"}},
			},
			sortByType: true,
		}
		// "alpha" < "beta", so block[1] should come before block[0]
		if !bs.Less(1, 0) {
			t.Error("alpha label should come before beta label")
		}
		if bs.Less(0, 1) {
			t.Error("beta label should not come before alpha label")
		}
	})

	/*********************************************************************/
	// Same type, same first label, block with fewer labels comes first.
	// Exercises the len(block1.Labels) < len(block2.Labels) return path.
	/*********************************************************************/

	t.Run("same type fewer labels comes first", func(t *testing.T) {
		bs := BlockListSorter{
			blocks: []*hclsyntax.Block{
				{Type: "resource", Labels: []string{"aws_s3_bucket"}},
				{Type: "resource", Labels: []string{"aws_s3_bucket", "extra"}},
			},
			sortByType: true,
		}
		if !bs.Less(0, 1) {
			t.Error("block with fewer labels should come first")
		}
		if bs.Less(1, 0) {
			t.Error("block with more labels should not come first")
		}
	})

	/*********************************************************************/
	// Same type, identical labels: not Less in either direction.
	/*********************************************************************/

	t.Run("same type identical labels returns false", func(t *testing.T) {
		bs := BlockListSorter{
			blocks: []*hclsyntax.Block{
				{Type: "resource", Labels: []string{"aws_s3_bucket", "my_bucket"}},
				{Type: "resource", Labels: []string{"aws_s3_bucket", "my_bucket"}},
			},
			sortByType: true,
		}
		if bs.Less(0, 1) || bs.Less(1, 0) {
			t.Error("identical blocks should not be Less in either direction")
		}
	})
}

func TestGetNodeComment(t *testing.T) {
	s := NewSorter(&Params{}, afero.NewMemMapFs())

	/*********************************************************************/
	// Block comment directly adjacent to the resource (no blank line).
	// The closing */ must not inject a spurious blank line.
	/*********************************************************************/

	t.Run("block comment adjacent to block", func(t *testing.T) {
		lines := []string{
			"/*",
			" * This is a block comment",
			" */",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		// startLine is the 0-indexed position of the resource line.
		expected := []string{
			"/*",
			" * This is a block comment",
			" */",
		}
		result := s.getNodeComment(lines, 3, "test.tf")
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getNodeComment() = %v, want %v", result, expected)
		}
	})

	/*********************************************************************/
	// Block comment separated from the resource by one blank line.
	// That single blank line must be preserved in the output.
	/*********************************************************************/

	t.Run("block comment with blank line before block", func(t *testing.T) {
		lines := []string{
			"/*",
			" * This is a block comment",
			" */",
			"",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		expected := []string{
			"/*",
			" * This is a block comment",
			" */",
			"",
		}
		result := s.getNodeComment(lines, 4, "test.tf")
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getNodeComment() = %v, want %v", result, expected)
		}
	})

	/*********************************************************************/
	// Consecutive single-line comments adjacent to the resource.
	// Both lines must be captured; no spurious blank line appended.
	/*********************************************************************/

	t.Run("consecutive single-line comments adjacent to block", func(t *testing.T) {
		lines := []string{
			"# First comment",
			"# Second comment",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		expected := []string{
			"# First comment",
			"# Second comment",
		}
		result := s.getNodeComment(lines, 2, "test.tf")
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getNodeComment() = %v, want %v", result, expected)
		}
	})

	/*********************************************************************/
	// Block with no preceding comment at all.
	// The function must return an empty slice (not nil panic).
	/*********************************************************************/

	t.Run("block with no comment", func(t *testing.T) {
		lines := []string{
			`variable "other" {`,
			"  default = 1",
			"}",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		result := s.getNodeComment(lines, 3, "test.tf")
		if len(result) != 0 {
			t.Errorf("getNodeComment() = %v, want empty slice", result)
		}
	})
}

func TestParseHclFileUsesInjectedFileSystem(t *testing.T) {
	memFS := afero.NewMemMapFs()

	// Write a minimal .tf file into the in-memory FS
	tfPath := "/test/main.tf"
	tfContent := []byte("resource \"null_resource\" \"example\" {}\n")
	if err := memFS.MkdirAll("/test", 0755); err != nil {
		t.Fatalf("could not create directory: %v", err)
	}
	if err := afero.WriteFile(memFS, tfPath, tfContent, 0644); err != nil {
		t.Fatalf("could not write tf file: %v", err)
	}

	s := NewSorter(&Params{}, memFS)
	body, err := s.parseHclFile(tfPath)
	if err != nil {
		t.Fatalf("parseHclFile() returned unexpected error: %v", err)
	}
	if body == nil {
		t.Fatal("parseHclFile() returned nil body, expected non-nil")
	}
}

// stubHCLBody is a minimal hcl.Body implementation whose concrete type is
// intentionally not *hclsyntax.Body, used to exercise the type-assertion
// error path in parseHclFile.
type stubHCLBody struct{}

func (s *stubHCLBody) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	return &hcl.BodyContent{}, nil
}
func (s *stubHCLBody) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	return &hcl.BodyContent{}, s, nil
}
func (s *stubHCLBody) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	return hcl.Attributes{}, nil
}
func (s *stubHCLBody) MissingItemRange() hcl.Range { return hcl.Range{} }

func TestParseHclFileNonHclsyntaxBody(t *testing.T) {
	// Save and restore the parse function only.
	origParseFn := hclParseFn
	t.Cleanup(func() {
		hclParseFn = origParseFn
	})

	// Use an in-memory filesystem so ReadFile succeeds.
	memFS := afero.NewMemMapFs()
	tfPath := "/test/stub.tf"
	if err := memFS.MkdirAll("/test", 0755); err != nil {
		t.Fatalf("could not create directory: %v", err)
	}
	if err := afero.WriteFile(memFS, tfPath, []byte("# stub\n"), 0644); err != nil {
		t.Fatalf("could not write stub file: %v", err)
	}

	// Inject a parser that returns a file whose body is not *hclsyntax.Body.
	hclParseFn = func(content []byte, filename string) (*hcl.File, hcl.Diagnostics) {
		return &hcl.File{Body: &stubHCLBody{}}, nil
	}

	s := NewSorter(&Params{}, memFS)
	_, err := s.parseHclFile(tfPath)
	if err == nil {
		t.Fatal("parseHclFile() expected an error for non-hclsyntax body, got nil")
	}
	if !strings.Contains(err.Error(), "*hclsyntax.Body") {
		t.Errorf("parseHclFile() error = %q; want it to mention *hclsyntax.Body", err.Error())
	}
}

func TestFindHeaderInLines(t *testing.T) {

	/*********************************************************************/
	// Multi-line block comment with double-asterisk close (**/). The
	// header pattern is a partial match ("Copyright"), not the full text.
	/*********************************************************************/

	t.Run("block comment with double asterisk close", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())

		lines := []string{
			"/**",
			" * Copyright (c) 2025 Example Corp",
			" **/",
			"",
			`resource "aws_instance" "foo" {`,
			"}",
		}

		header := s.findHeaderInLines(lines)
		expected := "/**\n * Copyright (c) 2025 Example Corp\n **/"
		if header != expected {
			t.Errorf("findHeaderInLines() = %q, want %q", header, expected)
		}
	})

	/*********************************************************************/
	// Block comment with standard close (*/). The header pattern matches
	// a substring within the header.
	/*********************************************************************/

	t.Run("block comment with standard close", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Apache License",
		}, afero.NewMemMapFs())

		lines := []string{
			"/**",
			" * Licensed under the Apache License, Version 2.0",
			" */",
			"",
			`resource "aws_instance" "foo" {`,
		}

		header := s.findHeaderInLines(lines)
		expected := "/**\n * Licensed under the Apache License, Version 2.0\n */"
		if header != expected {
			t.Errorf("findHeaderInLines() = %q, want %q", header, expected)
		}
	})

	/*********************************************************************/
	// Single-line hash comments as header.
	/*********************************************************************/

	t.Run("hash comment header", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "managed by Terraform",
		}, afero.NewMemMapFs())

		lines := []string{
			"# This file is managed by Terraform",
			"# Do not edit manually",
			"",
			`variable "name" {`,
		}

		header := s.findHeaderInLines(lines)
		expected := "# This file is managed by Terraform\n# Do not edit manually"
		if header != expected {
			t.Errorf("findHeaderInLines() = %q, want %q", header, expected)
		}
	})

	/*********************************************************************/
	// Pattern not found in the comment → no header detected.
	/*********************************************************************/

	t.Run("pattern not found returns empty", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "NONEXISTENT",
		}, afero.NewMemMapFs())

		lines := []string{
			"# A comment",
			`resource "aws_instance" "foo" {`,
		}

		header := s.findHeaderInLines(lines)
		if header != "" {
			t.Errorf("findHeaderInLines() = %q, want empty string", header)
		}
	})

	/*********************************************************************/
	// No comments at top of file → no header.
	/*********************************************************************/

	t.Run("no comments returns empty", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())

		lines := []string{
			`resource "aws_instance" "foo" {`,
			"}",
		}

		header := s.findHeaderInLines(lines)
		if header != "" {
			t.Errorf("findHeaderInLines() = %q, want empty string", header)
		}
	})

	/*********************************************************************/
	// header-end-pattern explicitly marks the end of the header, even
	// when there are more comments after it.
	/*********************************************************************/

	t.Run("header end pattern", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:        true,
			HeaderPattern:    "/**",
			HeaderEndPattern: "**/",
		}, afero.NewMemMapFs())

		lines := []string{
			"/**",
			" * Copyright (c) 2025",
			" **/",
			"# This is a block comment, not the header",
			`resource "aws_instance" "foo" {`,
		}

		header := s.findHeaderInLines(lines)
		expected := "/**\n * Copyright (c) 2025\n **/"
		if header != expected {
			t.Errorf("findHeaderInLines() = %q, want %q", header, expected)
		}
	})

	/*********************************************************************/
	// Leading blank lines before the header are skipped.
	/*********************************************************************/

	t.Run("leading blank lines skipped", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())

		lines := []string{
			"",
			"",
			"/**",
			" * Copyright 2025",
			" */",
			`resource "aws_instance" "foo" {`,
		}

		header := s.findHeaderInLines(lines)
		expected := "/**\n * Copyright 2025\n */"
		if header != expected {
			t.Errorf("findHeaderInLines() = %q, want %q", header, expected)
		}
	})

	/*********************************************************************/
	// YAML literal block pattern with trailing newline still matches.
	/*********************************************************************/

	t.Run("yaml trailing newline pattern still matches", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "/**\n * Copyright 2025\n */\n", // YAML | adds trailing newline
		}, afero.NewMemMapFs())

		lines := []string{
			"/**",
			" * Copyright 2025",
			" */",
			`resource "aws_instance" "foo" {`,
		}

		header := s.findHeaderInLines(lines)
		expected := "/**\n * Copyright 2025\n */"
		if header != expected {
			t.Errorf("findHeaderInLines() = %q, want %q", header, expected)
		}
	})
}

func TestRemoveHeaderWithDetectedHeader(t *testing.T) {

	/*********************************************************************/
	// When a detected header is stored, removeHeader should strip exactly
	// those lines from the comment and return the remainder.
	/*********************************************************************/

	t.Run("strips detected header lines", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())

		// Pre-populate the detected header.
		s.detectedHeaders["test.tf"] = "/**\n * Copyright (c) 2025\n **/"

		comment := []string{
			"/**",
			" * Copyright (c) 2025",
			" **/",
			"",
			"# Block comment",
		}

		result := s.removeHeader(comment, "test.tf")
		expected := []string{"", "# Block comment"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("removeHeader() = %v, want %v", result, expected)
		}
	})

	/*********************************************************************/
	// When detected header matches the entire comment, result is empty.
	/*********************************************************************/

	t.Run("entire comment is header", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())

		s.detectedHeaders["test.tf"] = "/**\n * Copyright (c) 2025\n **/"

		comment := []string{
			"/**",
			" * Copyright (c) 2025",
			" **/",
			"",
		}

		result := s.removeHeader(comment, "test.tf")
		if len(result) != 0 {
			t.Errorf("removeHeader() = %v, want empty slice", result)
		}
	})

	/*********************************************************************/
	// When no detected header exists, falls back to legacy string replace.
	/*********************************************************************/

	t.Run("legacy fallback when no detected header", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "# This file is managed by Terraform",
		}, afero.NewMemMapFs())

		comment := []string{
			"# This file is managed by Terraform",
			"# Do not edit",
		}

		result := s.removeHeader(comment, "unknown.tf")
		expected := []string{"# Do not edit"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("removeHeader() = %v, want %v", result, expected)
		}
	})
}

func TestAddHeaderWithDetectedHeader(t *testing.T) {

	/*********************************************************************/
	// When a detected header exists, addHeader should use it instead of
	// the raw HeaderPattern.
	/*********************************************************************/

	t.Run("uses detected header", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
			KeepHeader:    true,
		}, afero.NewMemMapFs())

		s.detectedHeaders["test.tf"] = "/**\n * Copyright (c) 2025\n **/"

		content := []byte("resource \"aws_instance\" \"a\" {}\n")
		result := s.addHeader(content, "test.tf")

		expected := "/**\n * Copyright (c) 2025\n **/\n\nresource \"aws_instance\" \"a\" {}\n"
		if string(result) != expected {
			t.Errorf("addHeader() = %q, want %q", string(result), expected)
		}
	})

	/*********************************************************************/
	// When no detected header exists, falls back to HeaderPattern.
	/*********************************************************************/

	t.Run("falls back to HeaderPattern", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "# Header\n",
			KeepHeader:    true,
		}, afero.NewMemMapFs())

		content := []byte("resource \"aws_instance\" \"a\" {}\n")
		result := s.addHeader(content, "unknown.tf")

		expected := "# Header\n\nresource \"aws_instance\" \"a\" {}\n"
		if string(result) != expected {
			t.Errorf("addHeader() = %q, want %q", string(result), expected)
		}
	})
}

func TestRemoveLeadingEmptyLines(t *testing.T) {

	/*********************************************************************/
	// Happy path test for removeLeadingEmptyLines() with a leading empty lines
	/*********************************************************************/

	t.Run("with leading lines", func(t *testing.T) {
		testData := []string{
			"",
			"",
			"/******************************************",
			"  Mandatory firewall rules",
			" *****************************************/",
		}

		expected := []string{
			"/******************************************",
			"  Mandatory firewall rules",
			" *****************************************/",
		}

		result := removeLeadingEmptyLines(testData)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("removeLeadingEmptyLines returned %v, expected %v\n", result, expected)
		}
	})
}

func TestRemoveHeaderPreservesBlankLineSeparator(t *testing.T) {
	t.Run("detected header preserves blank line before remaining comments", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())
		s.detectedHeaders["test.tf"] = "/**\n * Copyright (c) 2025\n **/"
		comment := []string{"/**", " * Copyright (c) 2025", " **/", "", "# Block comment"}
		result := s.removeHeader(comment, "test.tf")
		expected := []string{"", "# Block comment"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("removeHeader() = %v, want %v", result, expected)
		}
	})

	t.Run("detected header entire comment no extra blank line", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())
		s.detectedHeaders["test.tf"] = "/**\n * Copyright (c) 2025\n **/"
		comment := []string{"/**", " * Copyright (c) 2025", " **/"}
		result := s.removeHeader(comment, "test.tf")
		if len(result) != 0 {
			t.Errorf("removeHeader() = %v, want empty slice", result)
		}
	})

	t.Run("legacy fallback preserves blank line before remaining comments", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "# This file is managed by Terraform",
		}, afero.NewMemMapFs())
		comment := []string{"# This file is managed by Terraform", "", "# Do not edit"}
		result := s.removeHeader(comment, "unknown.tf")
		expected := []string{"", "# Do not edit"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("removeHeader() = %v, want %v", result, expected)
		}
	})

	t.Run("legacy fallback entire comment no extra blank line", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			HeaderPattern: "# Only line",
		}, afero.NewMemMapFs())
		comment := []string{"# Only line"}
		result := s.removeHeader(comment, "unknown.tf")
		if len(result) != 0 {
			t.Errorf("removeHeader() = %v, want empty slice", result)
		}
	})

	t.Run("detected header with keep-header does not add blank line", func(t *testing.T) {
		s := NewSorter(&Params{
			HasHeader:     true,
			KeepHeader:    true,
			HeaderPattern: "Copyright",
		}, afero.NewMemMapFs())
		s.detectedHeaders["test.tf"] = "/**\n * Copyright (c) 2025\n **/"
		comment := []string{"/**", " * Copyright (c) 2025", " **/", "", "# Block comment"}
		result := s.removeHeader(comment, "test.tf")
		expected := []string{"# Block comment"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("removeHeader() = %v, want %v", result, expected)
		}
	})
}
