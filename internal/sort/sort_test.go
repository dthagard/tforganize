package sort

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// TestSortNilSettings verifies that Sort does not panic when settings is nil
// and returns an error for a nonexistent target.
func TestSortNilSettings(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Sort panicked with nil settings: %v", r)
		}
	}()
	err := Sort("nonexistent-path-that-does-not-exist", nil)
	if err == nil {
		t.Fatal("expected an error for nonexistent target, got nil")
	}
}

// TestSortInlineDoesNotMutateSettings verifies that Sort with Inline=true does
// not overwrite the caller's Params.OutputDir.
func TestSortInlineDoesNotMutateSettings(t *testing.T) {
	// Use an in-memory filesystem so the Sorter can actually process a file.
	memFS := afero.NewMemMapFs()

	const target = "/testinline"
	_ = memFS.MkdirAll(target, 0755)
	_ = afero.WriteFile(memFS, filepath.Join(target, "main.tf"), []byte("resource \"aws_s3_bucket\" \"b\" {}\n"), 0644)

	settings := &Params{Inline: true}
	s := NewSorter(settings, memFS)
	if err := s.run(target); err != nil {
		t.Fatalf("Sort returned unexpected error: %v", err)
	}

	if settings.OutputDir != "" {
		t.Fatalf("Sort mutated caller's OutputDir: got %q, want empty string", settings.OutputDir)
	}
}

// TestSortErrorPaths verifies that Sort returns errors (not panics or silently
// continues) for each known fatal condition.
func TestSortErrorPaths(t *testing.T) {
	t.Run("nonexistent target", func(t *testing.T) {
		err := Sort("nonexistent-path-xyz-does-not-exist", nil)
		if err == nil {
			t.Fatal("expected error for nonexistent target, got nil")
		}
	})

	t.Run("inline conflicts with group-by-type", func(t *testing.T) {
		err := Sort(".", &Params{Inline: true, GroupByType: true})
		if err == nil {
			t.Fatal("expected error when inline conflicts with group-by-type, got nil")
		}
	})

	t.Run("inline conflicts with output-dir", func(t *testing.T) {
		err := Sort(".", &Params{Inline: true, OutputDir: "/some/dir"})
		if err == nil {
			t.Fatal("expected error when inline conflicts with output-dir, got nil")
		}
	})

	t.Run("keep-header without has-header", func(t *testing.T) {
		err := Sort(".", &Params{KeepHeader: true, HasHeader: false, HeaderPattern: "# header"})
		if err == nil {
			t.Fatal("expected error when keep-header is set but has-header is false, got nil")
		}
	})

	t.Run("keep-header with empty header-pattern", func(t *testing.T) {
		err := Sort(".", &Params{KeepHeader: true, HasHeader: true, HeaderPattern: ""})
		if err == nil {
			t.Fatal("expected error when keep-header is set but header-pattern is empty, got nil")
		}
	})

	t.Run("keep-header with valid has-header and header-pattern does not error on validation", func(t *testing.T) {
		memFS := afero.NewMemMapFs()

		const target = "/valid-keep-header"
		_ = memFS.MkdirAll(target, 0755)
		_ = afero.WriteFile(memFS, filepath.Join(target, "main.tf"), []byte("resource \"aws_s3_bucket\" \"b\" {}\n"), 0644)

		s := NewSorter(&Params{KeepHeader: true, HasHeader: true, HeaderPattern: "# header"}, memFS)
		err := s.run(target)
		// The validation should pass; any error here is from processing, not param validation.
		if err != nil && err.Error() == "keep-header requires has-header=true and a non-empty header-pattern" {
			t.Fatal("Sort incorrectly rejected valid keep-header params")
		}
	})

	t.Run("sortFiles failure via stub", func(t *testing.T) {
		memFS := afero.NewMemMapFs()

		const target = "/bad-hcl"
		_ = memFS.MkdirAll(target, 0755)
		_ = afero.WriteFile(memFS, filepath.Join(target, "bad.tf"), []byte("this is not { valid HCL\n"), 0644)

		s := NewSorter(nil, memFS)
		err := s.run(target)
		if err == nil {
			t.Fatal("expected error when sortFiles encounters invalid HCL, got nil")
		}
	})
}

const (
	sortedDir   = "sorted"
	testDataDir = "testdata"
	unsortedDir = "unsorted"
)

// testAFS is the afero helper used by golden-file tests.
var testAFS = &afero.Afero{Fs: afero.NewOsFs()}

func TestSortFile(t *testing.T) {
	/*********************************************************************/
	// Happy path test for sortFile() with a single file
	/*********************************************************************/

	t.Run("single file", func(t *testing.T) {
		path := filepath.Join(testDataDir, "single_file")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Happy path test for sortFile() with a multiple files
	/*********************************************************************/

	t.Run("multiple files", func(t *testing.T) {
		path := filepath.Join(testDataDir, "multiple_files")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Happy path test for sortFile() with a multiple files with headers
	/*********************************************************************/

	t.Run("multiple files with headers", func(t *testing.T) {
		path := filepath.Join(testDataDir, "multiple_files_with_headers")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Test that pre-meta arguments are sorted in their defined canonical
	// order (not alphabetically) across variable, resource, and module
	// block types. Also validates that the sort comparator returns false
	// (not true) when neither key is found, preserving strict weak ordering.
	/*********************************************************************/

	t.Run("meta arg ordering", func(t *testing.T) {
		path := filepath.Join(testDataDir, "meta_arg_ordering")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Regression test for issue #5: multiple nested blocks of the same
	// type (no labels) must all be preserved in the output. Previously,
	// only the first block was emitted and all subsequent blocks were
	// silently replaced with a copy of the first.
	/*********************************************************************/

	t.Run("multiple nested blocks", func(t *testing.T) {
		path := filepath.Join(testDataDir, "multiple_nested_blocks")
		testSortFile(path, t)
	})

	t.Run("data blocks", func(t *testing.T) {
		path := filepath.Join(testDataDir, "data_blocks")
		testSortFile(path, t)
	})

	t.Run("module blocks", func(t *testing.T) {
		path := filepath.Join(testDataDir, "module_blocks")
		testSortFile(path, t)
	})

	t.Run("terraform block", func(t *testing.T) {
		path := filepath.Join(testDataDir, "terraform_block")
		testSortFile(path, t)
	})

	t.Run("locals block", func(t *testing.T) {
		path := filepath.Join(testDataDir, "locals_block")
		testSortFile(path, t)
	})

	t.Run("heredoc syntax", func(t *testing.T) {
		path := filepath.Join(testDataDir, "heredoc_syntax")
		testSortFile(path, t)
	})

	t.Run("import and check blocks", func(t *testing.T) {
		path := filepath.Join(testDataDir, "import_check_blocks")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Multi-line header with double-asterisk close (**/) and a partial
	// header-pattern ("Copyright"). This is the HIGH-severity bug from
	// the improvements file: previously, a partial pattern caused
	// removeHeader to leave comment fragments and addHeader to prepend
	// only the partial pattern, producing invalid HCL.
	/*********************************************************************/

	t.Run("multiline header double asterisk", func(t *testing.T) {
		path := filepath.Join(testDataDir, "multiline_header_double_asterisk")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Multi-line header with --header-end-pattern set. This verifies that
	// the header is bounded by the start/end patterns and a block comment
	// after the header is not treated as part of it.
	/*********************************************************************/

	t.Run("header end pattern", func(t *testing.T) {
		path := filepath.Join(testDataDir, "header_end_pattern")
		testSortFile(path, t)
	})
}

// testSortFile is a helper function for TestSortFile
func testSortFile(path string, t *testing.T) {
	t.Helper()

	p := &Params{}
	if err := setParams(path, p); err != nil {
		t.Fatalf("could not set params: %v", err)
	}
	p.OutputDir = filepath.Join("./scratch", path)

	// Get the files in the unsorted directory
	unsortedFiles, err := testAFS.ReadDir(filepath.Join(path, unsortedDir))
	if err != nil {
		t.Fatalf("could not read unsorted directory: %v", err)
	}

	// Check each file if sorted correctly
	for _, file := range unsortedFiles {
		fileName := file.Name()

		// Read the sorted file
		sortedFilePath := filepath.Join(path, sortedDir, fileName)
		sortedBytes, err := testAFS.ReadFile(sortedFilePath)
		if err != nil {
			t.Fatalf("could not read expected sorted file %s: %v", sortedFilePath, err)
		}

		// Sort the unsorted file
		unsortedFilePath := filepath.Join(path, unsortedDir, fileName)
		s := NewSorter(p, afero.NewOsFs())
		results, err := s.sortFile(unsortedFilePath)
		if err != nil {
			t.Fatalf("sortFile(%s) returned unexpected error: %v", unsortedFilePath, err)
		}

		// Compare the sorted file to the expected sorted file
		if !reflect.DeepEqual(results[fileName], sortedBytes) {
			t.Errorf("sortFile(%s) did not match the expected sorted file", fileName)
		}
	}
}

// TestGroupByTypeFileRouting verifies that sortFile routes each block type to
// the expected output file when --group-by-type is enabled.
//
// Specifically it asserts:
//   - import blocks  → imports.tf
//   - check blocks   → checks.tf
//   - moved blocks   → main.tf
//   - removed blocks → main.tf
func TestGroupByTypeFileRouting(t *testing.T) {
	memFS := afero.NewMemMapFs()

	const inputPath = "/testrouting/main.tf"
	_ = memFS.MkdirAll("/testrouting", 0755)
	_ = afero.WriteFile(memFS, inputPath, []byte(`check "health_check" {
  assert {
    condition     = true
    error_message = "Health check failed"
  }
}

import {
  to = aws_instance.example
  id = "i-abcdef0123456789"
}

moved {
  from = aws_instance.old
  to   = aws_instance.new
}

removed {
  from = aws_instance.removed
  lifecycle {
    destroy = false
  }
}
`), 0644)

	s := NewSorter(&Params{GroupByType: true}, memFS)
	results, err := s.sortFile(inputPath)
	if err != nil {
		t.Fatalf("sortFile returned unexpected error: %v", err)
	}

	wantKeys := map[string]string{
		"checks.tf":  "check",
		"imports.tf": "import",
		"main.tf":    "moved",
	}

	for file, blockType := range wantKeys {
		content, ok := results[file]
		if !ok {
			t.Errorf("expected output key %q not found in results; got keys: %v", file, mapKeys(results))
			continue
		}
		if !strings.Contains(string(content), blockType) {
			t.Errorf("%s: expected to contain %q block, got:\n%s", file, blockType, string(content))
		}
	}

	// removed also routes to main.tf alongside moved
	if content, ok := results["main.tf"]; ok {
		if !strings.Contains(string(content), "removed") {
			t.Errorf("main.tf: expected to contain removed block, got:\n%s", string(content))
		}
	}
}

// mapKeys returns the keys of a map[string][]byte as a slice for error messages.
func mapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// setParams is a helper function for testSortFile
func setParams(path string, p *Params) error {
	configFile := filepath.Join(path, ".tforganize.yaml")
	if ok, _ := testAFS.Exists(configFile); ok {
		config, err := testAFS.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("could not read config file: %s", configFile)
		}
		if err := yaml.Unmarshal(config, p); err != nil {
			return fmt.Errorf("could not unmarshal config file: %s", configFile)
		}
	}
	return nil
}

// TestRunInvalidExcludePattern verifies that Sort returns a non-nil error
// containing "invalid exclude pattern" when an invalid glob is supplied.
func TestRunInvalidExcludePattern(t *testing.T) {
	dir := t.TempDir()
	err := Sort(dir, &Params{Excludes: []string{"[invalid"}})
	if err == nil {
		t.Fatal("expected error for invalid exclude pattern, got nil")
	}
	if !strings.Contains(err.Error(), "invalid exclude pattern") {
		t.Errorf("error message %q should contain 'invalid exclude pattern'", err.Error())
	}
}

// TestSortWithExcludesIntegration verifies the end-to-end exclude behaviour:
// - generated.tf is present but excluded → file is left unchanged
// - main.tf is present and not excluded → file is processed (sorted)
func TestSortWithExcludesIntegration(t *testing.T) {
	memFS := afero.NewMemMapFs()

	const target = "/excl_integration"
	_ = memFS.MkdirAll(target, 0755)

	mainContent := `resource "aws_instance" "b_server" {
  instance_type = "t2.micro"
  ami           = "ami-b"
}

resource "aws_instance" "a_server" {
  instance_type = "t2.micro"
  ami           = "ami-a"
}
`
	generatedContent := `resource "aws_s3_bucket" "my_bucket" {
  bucket = "my-bucket"
}
`

	_ = afero.WriteFile(memFS, filepath.Join(target, "main.tf"), []byte(mainContent), 0644)
	_ = afero.WriteFile(memFS, filepath.Join(target, "generated.tf"), []byte(generatedContent), 0644)

	outDir := "/excl_integration_out"
	_ = memFS.MkdirAll(outDir, 0755)

	s := NewSorter(&Params{
		Excludes:  []string{"generated.tf"},
		OutputDir: outDir,
	}, memFS)
	if err := s.run(target); err != nil {
		t.Fatalf("Sort returned unexpected error: %v", err)
	}

	// generated.tf should NOT appear in the output directory (was excluded).
	if _, err := memFS.Stat(filepath.Join(outDir, "generated.tf")); err == nil {
		t.Error("generated.tf should not be written to output dir (it was excluded)")
	}

	// main.tf SHOULD appear in the output directory and be sorted (a before b).
	mainBytes, err := afero.ReadFile(memFS, filepath.Join(outDir, "main.tf"))
	if err != nil {
		t.Fatalf("main.tf not found in output dir: %v", err)
	}
	out := string(mainBytes)
	aIdx := strings.Index(out, "a_server")
	bIdx := strings.Index(out, "b_server")
	if aIdx == -1 || bIdx == -1 {
		t.Fatalf("expected both a_server and b_server in main.tf output, got:\n%s", out)
	}
	if aIdx > bIdx {
		t.Errorf("a_server should come before b_server in sorted output")
	}
}

func TestGetLineSlice(t *testing.T) {

	/*********************************************************************/
	// Middle line (not start, not end) with RemoveComments=false
	// should be returned unchanged.
	/*********************************************************************/

	t.Run("middle line unchanged", func(t *testing.T) {
		s := NewSorter(&Params{}, afero.NewMemMapFs())
		line := `  some_attr = "value"`
		result := s.getLineSlice(line, 0, 5, 2, 3, 20)
		if result != line {
			t.Errorf("got %q, want %q", result, line)
		}
	})

	/*********************************************************************/
	// Starting line is truncated from startCol (1-indexed).
	/*********************************************************************/

	t.Run("start line truncated from startCol", func(t *testing.T) {
		s := NewSorter(&Params{}, afero.NewMemMapFs())
		line := `  some_attr = "value"`
		// startCol=3: remove the first 2 chars (the leading spaces)
		result := s.getLineSlice(line, 1, 1, 1, 3, 22)
		expected := `some_attr = "value"`
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	/*********************************************************************/
	// RemoveComments=true: a comment line is blanked out.
	/*********************************************************************/

	t.Run("remove_comments blanks comment line", func(t *testing.T) {
		s := NewSorter(&Params{RemoveComments: true}, afero.NewMemMapFs())
		line := "# this is a comment"
		result := s.getLineSlice(line, 5, 5, 5, 1, 20)
		if result != "" {
			t.Errorf("got %q, want %q", result, "")
		}
	})

	/*********************************************************************/
	// RemoveComments=true, single-line attribute (startLine == endLine):
	// truncate inline comment by slicing to endCol-startCol.
	//
	// Line:    `  foo = "bar" # inline`
	// startCol=3 (col of 'f'), endCol=14 (col after closing '"')
	// After startCol truncation: `foo = "bar" # inline`
	// Then [:endCol-startCol] = [:11] → `foo = "bar"`
	/*********************************************************************/

	t.Run("remove_comments single-line truncates inline comment", func(t *testing.T) {
		s := NewSorter(&Params{RemoveComments: true}, afero.NewMemMapFs())
		line := `  foo = "bar" # inline`
		result := s.getLineSlice(line, 1, 1, 1, 3, 14)
		expected := `foo = "bar"`
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	/*********************************************************************/
	// RemoveComments=true, multi-line attribute end line
	// (currentLine == endLine, startLine != endLine):
	// truncate at endCol-1 without startCol shift.
	//
	// Line:    `  "bar" # inline`
	// endCol=8 (col after closing '"')
	// [:endCol-1] = [:7] → `  "bar"`
	/*********************************************************************/

	t.Run("remove_comments multi-line end line truncates at endCol-1", func(t *testing.T) {
		s := NewSorter(&Params{RemoveComments: true}, afero.NewMemMapFs())
		line := `  "bar" # inline`
		result := s.getLineSlice(line, 1, 3, 3, 3, 8)
		expected := `  "bar"`
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})
}

func TestSort(t *testing.T) {

	/*********************************************************************/
	// Conflicting flags (Inline + GroupByType) should return early
	// without panicking.
	/*********************************************************************/

	t.Run("conflicting flags returns early", func(t *testing.T) {
		dir := t.TempDir()
		_ = Sort(dir, &Params{Inline: true, GroupByType: true})
	})

	/*********************************************************************/
	// OutputDir: sorted content is written to a separate directory.
	// Verifies writeFiles/writeFile code paths.
	/*********************************************************************/

	t.Run("output dir writes sorted file", func(t *testing.T) {
		dir := t.TempDir()
		outDir := t.TempDir()

		content := `resource "aws_instance" "b_server" {
  instance_type = "t2.micro"
  ami           = "ami-b"
}

resource "aws_instance" "a_server" {
  instance_type = "t2.micro"
  ami           = "ami-a"
}
`
		inputPath := filepath.Join(dir, "main.tf")
		if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Sort(inputPath, &Params{OutputDir: outDir}); err != nil {
			t.Fatalf("Sort returned unexpected error: %v", err)
		}

		outBytes, err := os.ReadFile(filepath.Join(outDir, "main.tf"))
		if err != nil {
			t.Fatalf("output file not created: %v", err)
		}
		out := string(outBytes)

		aIdx := strings.Index(out, "a_server")
		bIdx := strings.Index(out, "b_server")
		if aIdx == -1 || bIdx == -1 {
			t.Fatalf("expected both a_server and b_server in output, got:\n%s", out)
		}
		if aIdx > bIdx {
			t.Errorf("a_server should come before b_server in sorted output")
		}
	})

	/*********************************************************************/
	// Inline: sorted content overwrites the original file in place.
	// Verifies the getDirectory + writeFiles code paths.
	/*********************************************************************/

	t.Run("inline sorts file in place", func(t *testing.T) {
		dir := t.TempDir()

		content := `resource "aws_instance" "b_server" {
  instance_type = "t2.micro"
  ami           = "ami-b"
}

resource "aws_instance" "a_server" {
  instance_type = "t2.micro"
  ami           = "ami-a"
}
`
		inputPath := filepath.Join(dir, "main.tf")
		if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Sort(inputPath, &Params{Inline: true}); err != nil {
			t.Fatalf("Sort returned unexpected error: %v", err)
		}

		outBytes, err := os.ReadFile(inputPath)
		if err != nil {
			t.Fatalf("could not read file after inline sort: %v", err)
		}
		out := string(outBytes)

		aIdx := strings.Index(out, "a_server")
		bIdx := strings.Index(out, "b_server")
		if aIdx == -1 || bIdx == -1 {
			t.Fatalf("expected both a_server and b_server in output, got:\n%s", out)
		}
		if aIdx > bIdx {
			t.Errorf("a_server should come before b_server after inline sort")
		}
	})

	/*********************************************************************/
	// GroupByType: blocks are split across type-specific output files.
	// Verifies combineFiles + sortFiles code paths.
	/*********************************************************************/

	t.Run("group by type splits blocks into type files", func(t *testing.T) {
		dir := t.TempDir()
		outDir := t.TempDir()

		content := `resource "aws_s3_bucket" "my_bucket" {
  bucket = "my-bucket"
}

variable "env" {
  description = "environment"
  default     = "dev"
}

output "bucket_name" {
  value = aws_s3_bucket.my_bucket.bucket
}
`
		inputPath := filepath.Join(dir, "main.tf")
		if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Sort(inputPath, &Params{GroupByType: true, OutputDir: outDir}); err != nil {
			t.Fatalf("Sort returned unexpected error: %v", err)
		}

		varBytes, err := os.ReadFile(filepath.Join(outDir, "variables.tf"))
		if err != nil {
			t.Fatalf("variables.tf not created: %v", err)
		}
		if !strings.Contains(string(varBytes), "variable") {
			t.Errorf("variables.tf should contain variable block, got:\n%s", string(varBytes))
		}

		outBytes, err := os.ReadFile(filepath.Join(outDir, "outputs.tf"))
		if err != nil {
			t.Fatalf("outputs.tf not created: %v", err)
		}
		if !strings.Contains(string(outBytes), "output") {
			t.Errorf("outputs.tf should contain output block, got:\n%s", string(outBytes))
		}

		mainBytes, err := os.ReadFile(filepath.Join(outDir, "main.tf"))
		if err != nil {
			t.Fatalf("main.tf not created: %v", err)
		}
		if !strings.Contains(string(mainBytes), "resource") {
			t.Errorf("main.tf should contain resource block, got:\n%s", string(mainBytes))
		}
	})

	/*********************************************************************/
	// RemoveComments: block-level and inline comments are stripped.
	/*********************************************************************/

	t.Run("remove comments strips comments from output", func(t *testing.T) {
		dir := t.TempDir()
		outDir := t.TempDir()

		content := `# This comment should be removed
resource "aws_s3_bucket" "beta" {
  bucket = "my-beta-bucket"
}

resource "aws_s3_bucket" "alpha" {
  bucket = "my-alpha-bucket"
}
`
		inputPath := filepath.Join(dir, "main.tf")
		if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Sort(inputPath, &Params{OutputDir: outDir, RemoveComments: true}); err != nil {
			t.Fatalf("Sort returned unexpected error: %v", err)
		}

		outBytes, err := os.ReadFile(filepath.Join(outDir, "main.tf"))
		if err != nil {
			t.Fatalf("output file not created: %v", err)
		}
		out := string(outBytes)

		if strings.Contains(out, "#") {
			t.Errorf("output should not contain comments, got:\n%s", out)
		}

		aIdx := strings.Index(out, "alpha")
		bIdx := strings.Index(out, "beta")
		if aIdx == -1 || bIdx == -1 {
			t.Fatalf("expected both alpha and beta in output, got:\n%s", out)
		}
		if aIdx > bIdx {
			t.Errorf("alpha should come before beta in sorted output")
		}
	})
}

// TestSortBytesMultiLineHeader verifies that SortBytes correctly handles
// multi-line /** **/ headers with a partial header-pattern.
func TestSortBytesMultiLineHeader(t *testing.T) {

	/*********************************************************************/
	// Partial pattern ("Copyright") with keep-header: the full header
	// must be preserved at the top and the body must be sorted.
	/*********************************************************************/

	t.Run("partial pattern preserves full header", func(t *testing.T) {
		input := []byte(`/**
 * Copyright (c) 2025 Example Corp
 **/

resource "aws_s3_bucket" "beta" {
  bucket = "my-beta-bucket"
}

resource "aws_s3_bucket" "alpha" {
  bucket = "my-alpha-bucket"
}
`)
		result, err := SortBytes(input, "main.tf", &Params{
			HasHeader:     true,
			KeepHeader:    true,
			HeaderPattern: "Copyright",
		})
		if err != nil {
			t.Fatalf("SortBytes returned unexpected error: %v", err)
		}

		out := string(result)
		// Header must appear at the top, fully intact.
		if !strings.HasPrefix(out, "/**\n * Copyright (c) 2025 Example Corp\n **/") {
			t.Errorf("header not preserved at top of output:\n%s", out)
		}
		// alpha must come before beta.
		aIdx := strings.Index(out, "alpha")
		bIdx := strings.Index(out, "beta")
		if aIdx == -1 || bIdx == -1 {
			t.Fatalf("expected both alpha and beta in output:\n%s", out)
		}
		if aIdx > bIdx {
			t.Errorf("alpha should come before beta in sorted output")
		}
	})

	/*********************************************************************/
	// has-header without keep-header: the header should be stripped from
	// the output entirely.
	/*********************************************************************/

	t.Run("has-header without keep-header strips header", func(t *testing.T) {
		input := []byte(`/**
 * Copyright (c) 2025 Example Corp
 **/

resource "aws_s3_bucket" "only" {
  bucket = "my-bucket"
}
`)
		result, err := SortBytes(input, "main.tf", &Params{
			HasHeader:     true,
			HeaderPattern: "Copyright",
		})
		if err != nil {
			t.Fatalf("SortBytes returned unexpected error: %v", err)
		}

		out := string(result)
		if strings.Contains(out, "Copyright") {
			t.Errorf("header should be stripped when keep-header is false:\n%s", out)
		}
		if !strings.Contains(out, "aws_s3_bucket") {
			t.Errorf("resource block should still be present:\n%s", out)
		}
	})

	/*********************************************************************/
	// header-end-pattern: the header ends at the line matching the end
	// pattern; comments after that line belong to their blocks.
	/*********************************************************************/

	t.Run("header end pattern preserves block comment after header", func(t *testing.T) {
		input := []byte(`/**
 * Copyright (c) 2025
 **/
# This comment belongs to beta
resource "aws_s3_bucket" "beta" {
  bucket = "my-beta-bucket"
}

resource "aws_s3_bucket" "alpha" {
  bucket = "my-alpha-bucket"
}
`)
		result, err := SortBytes(input, "main.tf", &Params{
			HasHeader:        true,
			KeepHeader:       true,
			HeaderPattern:    "/**",
			HeaderEndPattern: "**/",
		})
		if err != nil {
			t.Fatalf("SortBytes returned unexpected error: %v", err)
		}

		out := string(result)
		// Header must be preserved.
		if !strings.HasPrefix(out, "/**\n * Copyright (c) 2025\n **/") {
			t.Errorf("header not preserved at top of output:\n%s", out)
		}
		// The block comment "# This comment belongs to beta" should still
		// be attached to the beta resource (after alpha due to sorting).
		if !strings.Contains(out, "# This comment belongs to beta") {
			t.Errorf("block comment should be preserved:\n%s", out)
		}
	})
}

// TestRecursive verifies that --recursive mode processes nested directories.
func TestRecursive(t *testing.T) {
	t.Run("inline recursive sorts all nested directories", func(t *testing.T) {
		// Copy the unsorted recursive testdata into a temp dir so inline writes don't mutate testdata.
		srcDir := filepath.Join(testDataDir, "recursive", unsortedDir)
		tmpDir := t.TempDir()
		copyDir(t, srcDir, tmpDir)

		s := NewSorter(&Params{Recursive: true, Inline: true}, afero.NewOsFs())
		if err := s.run(tmpDir); err != nil {
			t.Fatalf("recursive inline sort failed: %v", err)
		}

		// Compare each file against the expected sorted output.
		sortedDir := filepath.Join(testDataDir, "recursive", "sorted")
		compareDirs(t, sortedDir, tmpDir)
	})

	t.Run("output-dir recursive mirrors directory structure", func(t *testing.T) {
		srcDir := filepath.Join(testDataDir, "recursive", unsortedDir)
		outDir := t.TempDir()

		s := NewSorter(&Params{Recursive: true, OutputDir: outDir}, afero.NewOsFs())
		if err := s.run(srcDir); err != nil {
			t.Fatalf("recursive output-dir sort failed: %v", err)
		}

		// Compare each file against the expected sorted output.
		expectedDir := filepath.Join(testDataDir, "recursive", "sorted")
		compareDirs(t, expectedDir, outDir)
	})

	t.Run("check recursive detects unsorted files", func(t *testing.T) {
		srcDir := filepath.Join(testDataDir, "recursive", unsortedDir)
		s := NewSorter(&Params{Recursive: true, Check: true}, afero.NewOsFs())
		err := s.run(srcDir)
		if err == nil {
			t.Fatal("expected check error for unsorted recursive target, got nil")
		}
		if !errors.Is(err, ErrCheckFailed) {
			t.Fatalf("expected ErrCheckFailed, got: %v", err)
		}
	})

	t.Run("check recursive passes on sorted files", func(t *testing.T) {
		sortedRefDir := filepath.Join(testDataDir, "recursive", "sorted")
		s := NewSorter(&Params{Recursive: true, Check: true}, afero.NewOsFs())
		err := s.run(sortedRefDir)
		if err != nil {
			t.Fatalf("expected no error for already-sorted recursive target, got: %v", err)
		}
	})

	t.Run("recursive requires directory target", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "main.tf")
		if err := os.WriteFile(filePath, []byte("resource \"a\" \"b\" {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		s := NewSorter(&Params{Recursive: true}, afero.NewOsFs())
		err := s.run(filePath)
		if err == nil {
			t.Fatal("expected error for file target with --recursive, got nil")
		}
		if !strings.Contains(err.Error(), "recursive flag requires a directory target") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("recursive with exclude skips matching directories", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		_ = memFS.MkdirAll("/root/.terraform/modules", 0755)
		_ = memFS.MkdirAll("/root/live", 0755)
		_ = afero.WriteFile(memFS, "/root/.terraform/modules/main.tf", []byte("resource \"b\" \"z\" {}\nresource \"a\" \"y\" {}\n"), 0644)
		_ = afero.WriteFile(memFS, "/root/live/main.tf", []byte("resource \"b\" \"z\" {}\nresource \"a\" \"y\" {}\n"), 0644)

		outDir := "/out"
		_ = memFS.MkdirAll(outDir, 0755)

		s := NewSorter(&Params{
			Recursive: true,
			OutputDir: outDir,
			Excludes:  []string{".terraform/**"},
		}, memFS)
		if err := s.run("/root"); err != nil {
			t.Fatalf("recursive with exclude failed: %v", err)
		}

		// live/main.tf should exist in output.
		if _, err := memFS.Stat(filepath.Join(outDir, "live", "main.tf")); err != nil {
			t.Error("expected live/main.tf in output dir")
		}
	})
}

// copyDir copies all files from src into dst recursively.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}
}

// compareDirs compares all .tf files in expectedDir against the corresponding
// files in actualDir.
func compareDirs(t *testing.T, expectedDir, actualDir string) {
	t.Helper()
	err := filepath.Walk(expectedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".tf") {
			return nil
		}
		rel, _ := filepath.Rel(expectedDir, path)
		expectedBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		actualPath := filepath.Join(actualDir, rel)
		actualBytes, err := os.ReadFile(actualPath)
		if err != nil {
			t.Errorf("expected file %s not found in output: %v", rel, err)
			return nil
		}
		if string(expectedBytes) != string(actualBytes) {
			t.Errorf("file %s differs:\n--- expected ---\n%s\n--- actual ---\n%s", rel, string(expectedBytes), string(actualBytes))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("compareDirs walk failed: %v", err)
	}
}

// TestDiff verifies that --diff mode produces unified diff output.
func TestDiff(t *testing.T) {
	t.Run("diff shows changes for unsorted file", func(t *testing.T) {
		memFS := afero.NewMemMapFs()

		content := `resource "aws_instance" "web" {
  ami = "ami-12345"
}

resource "aws_instance" "app" {
  ami = "ami-67890"
}
`
		_ = memFS.MkdirAll("/test", 0755)
		_ = afero.WriteFile(memFS, "/test/main.tf", []byte(content), 0644)

		s := NewSorter(&Params{Diff: true}, memFS)
		err := s.run("/test")
		// Diff mode without --check should return nil even when files differ.
		if err != nil {
			t.Fatalf("diff mode returned unexpected error: %v", err)
		}
	})

	t.Run("diff with check returns error for unsorted file", func(t *testing.T) {
		memFS := afero.NewMemMapFs()

		content := `resource "aws_instance" "web" {
  ami = "ami-12345"
}

resource "aws_instance" "app" {
  ami = "ami-67890"
}
`
		_ = memFS.MkdirAll("/test", 0755)
		_ = afero.WriteFile(memFS, "/test/main.tf", []byte(content), 0644)

		s := NewSorter(&Params{Diff: true, Check: true}, memFS)
		err := s.run("/test")
		if err == nil {
			t.Fatal("expected error with --diff --check on unsorted file, got nil")
		}
		if !errors.Is(err, ErrCheckFailed) {
			t.Fatalf("expected ErrCheckFailed, got: %v", err)
		}
	})

	t.Run("diff returns nil for already-sorted file", func(t *testing.T) {
		memFS := afero.NewMemMapFs()

		content := `resource "aws_instance" "app" {
  ami = "ami-67890"
}

resource "aws_instance" "web" {
  ami = "ami-12345"
}
`
		_ = memFS.MkdirAll("/test", 0755)
		_ = afero.WriteFile(memFS, "/test/main.tf", []byte(content), 0644)

		s := NewSorter(&Params{Diff: true, Check: true}, memFS)
		err := s.run("/test")
		if err != nil {
			t.Fatalf("expected nil for already-sorted file, got: %v", err)
		}
	})

	t.Run("diff conflicts with output-dir", func(t *testing.T) {
		s := NewSorter(&Params{Diff: true, OutputDir: "/out"}, afero.NewMemMapFs())
		err := s.run("/test")
		if err == nil || !strings.Contains(err.Error(), "diff flag conflicts with the output-dir flag") {
			t.Fatalf("expected conflict error, got: %v", err)
		}
	})

	t.Run("diff conflicts with inline", func(t *testing.T) {
		s := NewSorter(&Params{Diff: true, Inline: true}, afero.NewMemMapFs())
		err := s.run("/test")
		if err == nil || !strings.Contains(err.Error(), "diff flag conflicts with the inline flag") {
			t.Fatalf("expected conflict error, got: %v", err)
		}
	})
}

// TestUnifiedDiff verifies the unifiedDiff function directly.
func TestUnifiedDiff(t *testing.T) {
	t.Run("identical strings produce empty diff", func(t *testing.T) {
		result := unifiedDiff("a.tf", "a.tf", "hello\n", "hello\n")
		if result != "" {
			t.Errorf("expected empty diff for identical strings, got:\n%s", result)
		}
	})

	t.Run("different strings produce diff with hunks", func(t *testing.T) {
		a := "line1\nline2\nline3\n"
		b := "line1\nchanged\nline3\n"
		result := unifiedDiff("a.tf", "b.tf", a, b)
		if !strings.Contains(result, "--- a.tf") {
			t.Error("expected --- header")
		}
		if !strings.Contains(result, "+++ b.tf") {
			t.Error("expected +++ header")
		}
		if !strings.Contains(result, "-line2") {
			t.Error("expected deletion of line2")
		}
		if !strings.Contains(result, "+changed") {
			t.Error("expected insertion of changed")
		}
	})

	t.Run("empty to non-empty shows all additions", func(t *testing.T) {
		result := unifiedDiff("a.tf", "b.tf", "", "line1\nline2\n")
		if !strings.Contains(result, "+line1") {
			t.Error("expected additions")
		}
	})
}
