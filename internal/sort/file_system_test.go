package sort

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var (
	testFiles = []string{"foo.tf", "bar.tf", "baz.tf"}
)

func init() {
	// No-op
}

func fileSystemCleanup() {
	// No-op
}

func TestGetFilesFromTarget(t *testing.T) {
	t.Cleanup(fileSystemCleanup)

	/*********************************************************************/
	// Happy path test for getFilesFromTarget() with a directory with one file
	/*********************************************************************/

	t.Run("directory with one file", func(t *testing.T) {
		testDir := t.TempDir()
		AFS.MkdirAll(testDir, 0755)

		// Create a file in the directory
		foo, err := AFS.Create((filepath.Join(testDir, testFiles[0])))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		// Get files from target
		result, err := getFilesFromTarget(testDir)
		if err != nil {
			log.WithError(err).Errorln("could not get files from target")
		}

		if expected := []string{foo.Name()}; !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesFromTarget() returned %v, expected %v\n", result, expected)
		}
	})

	/*********************************************************************/
	// Happy path test for getFilesFromTarget() with a single file
	/*********************************************************************/

	t.Run("single file", func(t *testing.T) {
		testDir := t.TempDir()
		AFS.MkdirAll(testDir, 0755)

		// Create a file in the directory
		bar, err := AFS.Create(filepath.Join(testDir, testFiles[1]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer bar.Close()

		// Get files from target
		result, err := getFilesFromTarget(filepath.Join(testDir, testFiles[1]))
		if err != nil {
			log.WithError(err).Errorln("could not get file from target")
		}

		if expected := []string{bar.Name()}; !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesFromTarget() returned %v, expected %v\n", result, expected)
		}
	})

	/*********************************************************************/
	// Happy path test for getFilesFromTarget() with a directory with multiple files
	/*********************************************************************/

	t.Run("directory with multiple files", func(t *testing.T) {
		testDir := t.TempDir()
		AFS.MkdirAll(testDir, 0755)

		// Create files in the directory
		foo, err := AFS.Create((filepath.Join(testDir, testFiles[0])))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		bar, err := AFS.Create(filepath.Join(testDir, testFiles[1]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer bar.Close()

		baz, err := AFS.Create(filepath.Join(testDir, testFiles[2]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer baz.Close()

		// Get files from target
		result, err := getFilesFromTarget(testDir)
		if err != nil {
			log.WithError(err).Errorln("could not get files from target")
		}

		expected := []string{foo.Name(), bar.Name(), baz.Name()}

		// Sort the slices to ensure they are equal
		sort.Strings(result)
		sort.Strings(expected)

		// Compare the slices
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesFromTarget() returned %v, expected %v\n", result, expected)
		}
	})

	/*********************************************************************/
	// Sad path test for getFilesFromTarget() with a non-existent file
	/*********************************************************************/

	t.Run("non-existent file", func(t *testing.T) {
		testDir := t.TempDir()
		AFS.MkdirAll(testDir, 0755)

		// Get files from target
		if _, err := getFilesFromTarget(filepath.Join(testDir, "non-existent-file")); err == nil {
			t.Errorf("getFilesFromTarget() returned nil, expected error")
		}
	})
}

func TestGetFilesInFolder(t *testing.T) {
	t.Cleanup(fileSystemCleanup)

	/*********************************************************************/
	// Happy path test for getFilesInFolder() with a directory with no files
	/*********************************************************************/

	t.Run("directory with no files", func(t *testing.T) {
		testDir := t.TempDir()
		AFS.MkdirAll(testDir, 0755)

		// Get files from target
		result, err := getFilesInFolder(testDir)
		if err != nil {
			log.WithError(err).Errorln("could not get files from target")
		}
		if expected := []string{}; !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesInFolder(%s) returned %v, expected %v\n", testDir, result, expected)
		}
	})

	/*********************************************************************/
	// Happy path test for getFilesInFolder() with a directory with one file
	/*********************************************************************/

	t.Run("directory with one file", func(t *testing.T) {
		testDir := t.TempDir()
		AFS.MkdirAll(testDir, 0755)

		// Create a file in the directory
		foo, err := AFS.Create((filepath.Join(testDir, testFiles[0])))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		// Get files from target
		result, err := getFilesInFolder(testDir)
		if err != nil {
			log.WithError(err).Errorln("could not get files from target")
		}
		if expected := []string{foo.Name()}; !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesInFolder() returned %v, expected %v\n", result, expected)
		}
	})

	/*********************************************************************/
	// Happy path test for getFilesInFolder() with a directory with multiple files
	/*********************************************************************/

	t.Run("directory with multiple files", func(t *testing.T) {
		testDir := t.TempDir()
		AFS.MkdirAll(testDir, 0755)

		// Create files in the directory
		foo, err := AFS.Create(filepath.Join(testDir, testFiles[0]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		bar, err := AFS.Create(filepath.Join(testDir, testFiles[1]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer bar.Close()

		baz, err := AFS.Create(filepath.Join(testDir, testFiles[2]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer baz.Close()

		// Get files from target
		result, err := getFilesInFolder(testDir)
		if err != nil {
			log.WithError(err).Errorln("could not get files from target")
		}
		expected := []string{foo.Name(), bar.Name(), baz.Name()}

		// Sort the slices to ensure they are equal
		sort.Strings(result)
		sort.Strings(expected)

		// Compare the slices
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesInFolder() returned %v, expected %v", result, expected)
		}
	})

	/*********************************************************************/
	// Sad path test for getFilesInFolder() with a non-existent folder
	/*********************************************************************/

	t.Run("non-existent folder", func(t *testing.T) {
		// GetgetFilesInFolder() with a  files from target
		if _, err := getFilesInFolder("non-existent-folder"); err == nil {
			t.Errorf("getFilesInFolder() returned nil, expected Error")
		}
	})
}

// countingFs wraps an afero.Fs and counts the number of Open calls.
type countingFs struct {
	afero.Fs
	openCount int
}

func (c *countingFs) Open(name string) (afero.File, error) {
	c.openCount++
	return c.Fs.Open(name)
}

// TestGetLinesFromFileCache verifies that getLinesFromFile reads each file from
// the underlying filesystem exactly once per sort run and serves subsequent
// calls from the in-memory cache.
func TestGetLinesFromFileCache(t *testing.T) {
	memFS := afero.NewMemMapFs()
	counter := &countingFs{Fs: memFS}
	SetFileSystem(counter)
	defer SetFileSystem(afero.NewOsFs())

	// Ensure a clean cache for this test.
	clearLinesCache()

	const path = "/cache_test/main.tf"
	content := []byte("line1\nline2\nline3\n")
	if err := afero.WriteFile(memFS, path, content, 0644); err != nil {
		t.Fatalf("could not write test file: %v", err)
	}

	// First call — must read from the filesystem.
	lines1, err := getLinesFromFile(path)
	if err != nil {
		t.Fatalf("first getLinesFromFile returned error: %v", err)
	}

	// Second call — must be served from cache (no additional Open).
	lines2, err := getLinesFromFile(path)
	if err != nil {
		t.Fatalf("second getLinesFromFile returned error: %v", err)
	}

	if counter.openCount != 1 {
		t.Errorf("expected exactly 1 Open call, got %d", counter.openCount)
	}

	expected := []string{"line1", "line2", "line3"}
	if !reflect.DeepEqual(lines1, expected) {
		t.Errorf("getLinesFromFile returned %v, expected %v", lines1, expected)
	}
	if !reflect.DeepEqual(lines1, lines2) {
		t.Errorf("cache returned different result: first=%v second=%v", lines1, lines2)
	}
}
