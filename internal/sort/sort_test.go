package sort

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

const (
	sortedDir   = "sorted"
	testDataDir = "testdata"
	unsortedDir = "unsorted"
)

func init() {
	SetFileSystem(afero.NewOsFs())
}

func sortCleanup() {
	// No-op
}

func TestSortFile(t *testing.T) {
	t.Cleanup(sortCleanup)

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
	// Regression test for issue #5: multiple nested blocks of the same
	// type (no labels) must all be preserved in the output. Previously,
	// only the first block was emitted and all subsequent blocks were
	// silently replaced with a copy of the first.
	/*********************************************************************/

	t.Run("multiple nested blocks", func(t *testing.T) {
		path := filepath.Join(testDataDir, "multiple_nested_blocks")
		testSortFile(path, t)
	})
}

// testSortFile is a helper function for TestSortFile
func testSortFile(path string, t *testing.T) {
	// Reset params to defaults so state from a previous test case doesn't bleed in.
	initParams()

	// Get the files in the unsorted directory
	unsortedFiles, err := AFS.ReadDir(filepath.Join(path, unsortedDir))
	if err != nil {
		log.WithError(err).Errorln("could not read unsorted directory")
	}

	// Set the params
	if err := setParams(path); err != nil {
		log.WithError(err).Errorln("could not set params")
	}
	params.OutputDir = filepath.Join("./scratch", path)

	// Check each file if sorted correctly
	for _, file := range unsortedFiles {
		fileName := file.Name()

		// Read the sorted file
		sortedFilePath := filepath.Join(path, sortedDir, fileName)
		sortedBytes, err := AFS.ReadFile(sortedFilePath)
		if err != nil {
			log.WithError(err).Errorln("could not read file")
		}

		// Sort the unsorted file
		unsortedFilePath := filepath.Join(path, unsortedDir, fileName)
		results, err := sortFile(unsortedFilePath)
		if err != nil {
			log.WithError(err).Errorln("could not sort file")
		}

		// Compare the sorted file to the expected sorted file
		if !reflect.DeepEqual(results[fileName], sortedBytes) {
			t.Errorf("sortFile(%s) did not match the expected sorted file", fileName)
		}
	}
}

// setParams is a helper function for testSortFile
func setParams(path string) error {
	configFile := filepath.Join(path, ".tforganize.yaml")
	if ok, _ := AFS.Exists(configFile); ok {
		config, err := AFS.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("could not read config file: %s", configFile)
		}
		if err := yaml.Unmarshal(config, params); err != nil {
			return fmt.Errorf("could not unmarshal config file: %s", configFile)
		}
	}
	return nil
}

func TestGetLineSlice(t *testing.T) {

	/*********************************************************************/
	// Middle line (not start, not end) with RemoveComments=false
	// should be returned unchanged.
	/*********************************************************************/

	t.Run("middle line unchanged", func(t *testing.T) {
		initParams()
		line := `  some_attr = "value"`
		result := getLineSlice(line, 0, 5, 2, 3, 20)
		if result != line {
			t.Errorf("got %q, want %q", result, line)
		}
	})

	/*********************************************************************/
	// Starting line is truncated from startCol (1-indexed).
	/*********************************************************************/

	t.Run("start line truncated from startCol", func(t *testing.T) {
		initParams()
		line := `  some_attr = "value"`
		// startCol=3: remove the first 2 chars (the leading spaces)
		result := getLineSlice(line, 1, 1, 1, 3, 22)
		expected := `some_attr = "value"`
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	/*********************************************************************/
	// RemoveComments=true: a comment line is blanked out.
	/*********************************************************************/

	t.Run("remove_comments blanks comment line", func(t *testing.T) {
		initParams()
		params.RemoveComments = true
		line := "# this is a comment"
		result := getLineSlice(line, 5, 5, 5, 1, 20)
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
		initParams()
		params.RemoveComments = true
		line := `  foo = "bar" # inline`
		result := getLineSlice(line, 1, 1, 1, 3, 14)
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
		initParams()
		params.RemoveComments = true
		line := `  "bar" # inline`
		result := getLineSlice(line, 1, 3, 3, 3, 8)
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
		initParams()
		dir := t.TempDir()
		Sort(dir, &Params{Inline: true, GroupByType: true})
	})

	/*********************************************************************/
	// OutputDir: sorted content is written to a separate directory.
	// Verifies writeFiles/writeFile code paths.
	/*********************************************************************/

	t.Run("output dir writes sorted file", func(t *testing.T) {
		initParams()
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

		Sort(inputPath, &Params{OutputDir: outDir})

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
		initParams()
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

		Sort(inputPath, &Params{Inline: true})

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
		initParams()
		dir := t.TempDir()
		outDir := t.TempDir()

		// Ensure the tforganize temp dir exists (used by combineFiles).
		if err := os.MkdirAll(filepath.Join(os.TempDir(), "tforganize"), 0755); err != nil {
			t.Fatalf("could not create tforganize temp dir: %v", err)
		}

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

		Sort(inputPath, &Params{GroupByType: true, OutputDir: outDir})

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
		initParams()
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

		Sort(inputPath, &Params{OutputDir: outDir, RemoveComments: true})

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
