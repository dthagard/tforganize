package sort

import (
	"fmt"
	"path/filepath"
	"reflect"
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
	initParams()
	err := Sort("nonexistent-path-that-does-not-exist", nil)
	if err == nil {
		t.Fatal("expected an error for nonexistent target, got nil")
	}
}

// TestSortInlineDoesNotMutateSettings verifies that Sort with Inline=true does
// not overwrite the caller's Params.OutputDir.
func TestSortInlineDoesNotMutateSettings(t *testing.T) {
	// Use an in-memory filesystem so Sort can actually process a file.
	memFS := afero.NewMemMapFs()
	SetFileSystem(memFS)
	defer SetFileSystem(afero.NewOsFs())

	const target = "/testinline"
	_ = memFS.MkdirAll(target, 0755)
	_ = afero.WriteFile(memFS, filepath.Join(target, "main.tf"), []byte("resource \"aws_s3_bucket\" \"b\" {}\n"), 0644)

	initParams()
	settings := &Params{Inline: true}
	if err := Sort(target, settings); err != nil {
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
		initParams()
		err := Sort("nonexistent-path-xyz-does-not-exist", nil)
		if err == nil {
			t.Fatal("expected error for nonexistent target, got nil")
		}
	})

	t.Run("inline conflicts with group-by-type", func(t *testing.T) {
		initParams()
		err := Sort(".", &Params{Inline: true, GroupByType: true})
		if err == nil {
			t.Fatal("expected error when inline conflicts with group-by-type, got nil")
		}
	})

	t.Run("inline conflicts with output-dir", func(t *testing.T) {
		initParams()
		err := Sort(".", &Params{Inline: true, OutputDir: "/some/dir"})
		if err == nil {
			t.Fatal("expected error when inline conflicts with output-dir, got nil")
		}
	})

	t.Run("sortFiles failure via stub", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		SetFileSystem(memFS)
		defer SetFileSystem(afero.NewOsFs())

		const target = "/bad-hcl"
		_ = memFS.MkdirAll(target, 0755)
		_ = afero.WriteFile(memFS, filepath.Join(target, "bad.tf"), []byte("this is not { valid HCL\n"), 0644)

		initParams()
		err := Sort(target, nil)
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
}

// testSortFile is a helper function for TestSortFile
func testSortFile(path string, t *testing.T) {
	t.Helper()

	// Reset params to defaults so state from a previous test case doesn't bleed in.
	initParams()

	// Get the files in the unsorted directory
	unsortedFiles, err := AFS.ReadDir(filepath.Join(path, unsortedDir))
	if err != nil {
		t.Fatalf("could not read unsorted directory: %v", err)
	}

	// Set the params
	if err := setParams(path); err != nil {
		t.Fatalf("could not set params: %v", err)
	}
	params.OutputDir = filepath.Join("./scratch", path)

	// Check each file if sorted correctly
	for _, file := range unsortedFiles {
		fileName := file.Name()

		// Read the sorted file
		sortedFilePath := filepath.Join(path, sortedDir, fileName)
		sortedBytes, err := AFS.ReadFile(sortedFilePath)
		if err != nil {
			t.Fatalf("could not read expected sorted file %s: %v", sortedFilePath, err)
		}

		// Sort the unsorted file
		unsortedFilePath := filepath.Join(path, unsortedDir, fileName)
		results, err := sortFile(unsortedFilePath)
		if err != nil {
			t.Fatalf("sortFile(%s) returned unexpected error: %v", unsortedFilePath, err)
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
