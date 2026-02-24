package sort

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// TestSortNilSettings verifies that Sort does not panic when settings is nil.
func TestSortNilSettings(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Sort panicked with nil settings: %v", r)
		}
	}()
	initParams()
	// A nonexistent target causes a logged error; the important thing is no panic.
	Sort("nonexistent-path-that-does-not-exist", nil)
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
	Sort(target, settings)

	if settings.OutputDir != "" {
		t.Fatalf("Sort mutated caller's OutputDir: got %q, want empty string", settings.OutputDir)
	}
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
