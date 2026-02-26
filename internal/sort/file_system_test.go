package sort

import (
	"os"
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		// Create a file in the directory
		foo, err := os.Create(filepath.Join(testDir, testFiles[0]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		// Get files from target
		result, err := s.getFilesFromTarget(testDir)
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		// Create a file in the directory
		bar, err := os.Create(filepath.Join(testDir, testFiles[1]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer bar.Close()

		// Get files from target
		result, err := s.getFilesFromTarget(filepath.Join(testDir, testFiles[1]))
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		// Create files in the directory
		foo, err := os.Create(filepath.Join(testDir, testFiles[0]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		bar, err := os.Create(filepath.Join(testDir, testFiles[1]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer bar.Close()

		baz, err := os.Create(filepath.Join(testDir, testFiles[2]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer baz.Close()

		// Get files from target
		result, err := s.getFilesFromTarget(testDir)
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		// Get files from target
		if _, err := s.getFilesFromTarget(filepath.Join(testDir, "non-existent-file")); err == nil {
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		// Get files from target
		result, err := s.getFilesInFolder(testDir)
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		// Create a file in the directory
		foo, err := os.Create(filepath.Join(testDir, testFiles[0]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		// Get files from target
		result, err := s.getFilesInFolder(testDir)
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		// Create files in the directory
		foo, err := os.Create(filepath.Join(testDir, testFiles[0]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer foo.Close()

		bar, err := os.Create(filepath.Join(testDir, testFiles[1]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer bar.Close()

		baz, err := os.Create(filepath.Join(testDir, testFiles[2]))
		if err != nil {
			log.WithError(err).Errorln("could not create file")
		}
		defer baz.Close()

		// Get files from target
		result, err := s.getFilesInFolder(testDir)
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
		s := NewSorter(&Params{}, afero.NewOsFs())

		if _, err := s.getFilesInFolder("non-existent-folder"); err == nil {
			t.Errorf("getFilesInFolder() returned nil, expected Error")
		}
	})
}

// TestIsExcluded verifies the isExcluded helper against a table of cases.
func TestIsExcluded(t *testing.T) {
	cases := []struct {
		name             string
		targetDir        string
		filePath         string
		patterns         []string
		expectedExcluded bool
		expectedErr      bool
	}{
		{
			name:             "no patterns",
			targetDir:        "/a",
			filePath:         "/a/main.tf",
			patterns:         []string{},
			expectedExcluded: false,
			expectedErr:      false,
		},
		{
			name:             "exact basename match",
			targetDir:        "/a",
			filePath:         "/a/main.tf",
			patterns:         []string{"main.tf"},
			expectedExcluded: true,
			expectedErr:      false,
		},
		{
			name:             "wildcard match no suffix",
			targetDir:        "/a",
			filePath:         "/a/generated.tf",
			patterns:         []string{"*.generated.tf"},
			expectedExcluded: false,
			expectedErr:      false,
		},
		{
			name:             "wildcard match suffix",
			targetDir:        "/a",
			filePath:         "/a/foo.generated.tf",
			patterns:         []string{"*.generated.tf"},
			expectedExcluded: true,
			expectedErr:      false,
		},
		{
			name:             "double-star dir match",
			targetDir:        "/a",
			filePath:         "/a/.terraform/lock.hcl",
			patterns:         []string{".terraform/**"},
			expectedExcluded: true,
			expectedErr:      false,
		},
		{
			name:             "double-star no match",
			targetDir:        "/a",
			filePath:         "/a/variables.tf",
			patterns:         []string{".terraform/**"},
			expectedExcluded: false,
			expectedErr:      false,
		},
		{
			name:             "multiple patterns first matches",
			targetDir:        "/a",
			filePath:         "/a/main.tf",
			patterns:         []string{"main.tf", "*.tf"},
			expectedExcluded: true,
			expectedErr:      false,
		},
		{
			name:             "multiple patterns second matches",
			targetDir:        "/a",
			filePath:         "/a/main.tf",
			patterns:         []string{"variables.tf", "main.tf"},
			expectedExcluded: true,
			expectedErr:      false,
		},
		{
			name:             "multiple patterns none match",
			targetDir:        "/a",
			filePath:         "/a/main.tf",
			patterns:         []string{"variables.tf", "outputs.tf"},
			expectedExcluded: false,
			expectedErr:      false,
		},
		{
			name:             "invalid pattern",
			targetDir:        "/a",
			filePath:         "/a/main.tf",
			patterns:         []string{"[bad"},
			expectedExcluded: false,
			expectedErr:      true,
		},
		{
			name:             "cross-platform path sep",
			targetDir:        "/a",
			filePath:         "/a/sub/main.tf",
			patterns:         []string{"sub/main.tf"},
			expectedExcluded: true,
			expectedErr:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSorter(&Params{Excludes: tc.patterns}, afero.NewMemMapFs())
			excluded, err := s.isExcluded(tc.targetDir, tc.filePath)
			if tc.expectedErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if excluded != tc.expectedExcluded {
				t.Errorf("isExcluded() = %v, want %v", excluded, tc.expectedExcluded)
			}
		})
	}
}

// TestGetFilesInFolderWithExcludes verifies that getFilesInFolder honours the
// Excludes patterns by skipping matched files.
func TestGetFilesInFolderWithExcludes(t *testing.T) {
	t.Run("single exact exclude", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		const dir = "/excl"
		_ = memFS.MkdirAll(dir, 0755)
		for _, name := range []string{"foo.tf", "bar.tf", "generated.tf"} {
			_ = afero.WriteFile(memFS, filepath.Join(dir, name), []byte(""), 0644)
		}

		s := NewSorter(&Params{Excludes: []string{"generated.tf"}}, memFS)
		result, err := s.getFilesInFolder(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sort.Strings(result)
		expected := []string{filepath.Join(dir, "bar.tf"), filepath.Join(dir, "foo.tf")}
		sort.Strings(expected)

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesInFolder() = %v, want %v", result, expected)
		}
	})

	t.Run("exclude with wildcard", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		const dir = "/excl2"
		_ = memFS.MkdirAll(dir, 0755)
		for _, name := range []string{"foo.tf", "bar.generated.tf", "baz.tf"} {
			_ = afero.WriteFile(memFS, filepath.Join(dir, name), []byte(""), 0644)
		}

		s := NewSorter(&Params{Excludes: []string{"*.generated.tf"}}, memFS)
		result, err := s.getFilesInFolder(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sort.Strings(result)
		expected := []string{filepath.Join(dir, "baz.tf"), filepath.Join(dir, "foo.tf")}
		sort.Strings(expected)

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesInFolder() = %v, want %v", result, expected)
		}
	})

	t.Run("exclude all", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		const dir = "/excl3"
		_ = memFS.MkdirAll(dir, 0755)
		for _, name := range []string{"foo.tf", "bar.tf"} {
			_ = afero.WriteFile(memFS, filepath.Join(dir, name), []byte(""), 0644)
		}

		s := NewSorter(&Params{Excludes: []string{"*.tf"}}, memFS)
		result, err := s.getFilesInFolder(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("getFilesInFolder() = %v, want empty slice", result)
		}
	})

	t.Run("invalid pattern returns error", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		const dir = "/excl4"
		_ = memFS.MkdirAll(dir, 0755)
		_ = afero.WriteFile(memFS, filepath.Join(dir, "foo.tf"), []byte(""), 0644)

		s := NewSorter(&Params{Excludes: []string{"[bad"}}, memFS)
		_, err := s.getFilesInFolder(dir)
		if err == nil {
			t.Errorf("expected error for invalid pattern, got nil")
		}
	})
}

// TestGetFilesFromTargetSingleFileExcluded verifies that getFilesFromTarget
// returns an empty slice (no error) when the single-file target is excluded.
func TestGetFilesFromTargetSingleFileExcluded(t *testing.T) {
	t.Run("file matches exclude pattern", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		const dir = "/tmp/testdir"
		_ = memFS.MkdirAll(dir, 0755)
		target := filepath.Join(dir, "main.tf")
		_ = afero.WriteFile(memFS, target, []byte(""), 0644)

		s := NewSorter(&Params{Excludes: []string{"main.tf"}}, memFS)
		result, err := s.getFilesFromTarget(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("getFilesFromTarget() = %v, want empty slice", result)
		}
	})

	t.Run("file does not match exclude pattern", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		const dir = "/tmp/testdir2"
		_ = memFS.MkdirAll(dir, 0755)
		target := filepath.Join(dir, "main.tf")
		_ = afero.WriteFile(memFS, target, []byte(""), 0644)

		s := NewSorter(&Params{Excludes: []string{"variables.tf"}}, memFS)
		result, err := s.getFilesFromTarget(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := []string{target}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getFilesFromTarget() = %v, want %v", result, expected)
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

	const path = "/cache_test/main.tf"
	content := []byte("line1\nline2\nline3\n")
	if err := afero.WriteFile(memFS, path, content, 0644); err != nil {
		t.Fatalf("could not write test file: %v", err)
	}

	s := NewSorter(&Params{}, counter)

	// First call — must read from the filesystem.
	lines1, err := s.getLinesFromFile(path)
	if err != nil {
		t.Fatalf("first getLinesFromFile returned error: %v", err)
	}

	// Second call — must be served from cache (no additional Open).
	lines2, err := s.getLinesFromFile(path)
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
