package sort

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// These tests are intentionally nonparallel: the existing output API writes
// directly to process stdout/stderr, and path fallback tests change cwd.
func fsCaptureOutput(t *testing.T, stream **os.File, action func() error) (string, error) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	original := *stream
	*stream = file
	defer func() { *stream = original }()
	actionErr := action()
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(output), actionErr
}

func TestRunFilesystemFailures(t *testing.T) {
	t.Run("recursive target unavailable", func(t *testing.T) {
		err := NewSorter(&Params{Recursive: true}, afero.NewMemMapFs()).run("/missing")
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("run error = %v, want missing target", err)
		}
	})

	t.Run("inline target disappears after reading", func(t *testing.T) {
		base := afero.NewMemMapFs()
		fsPut(t, base, "/src/main.tf", sortedSingleBlock)
		failure := errors.New("target metadata unavailable")
		filesystem := &fsFailure{Fs: base}
		readStarted := false
		filesystem.open = func(name string) (afero.File, error) {
			readStarted = true
			return base.Open(name)
		}
		filesystem.stat = func(name string) (os.FileInfo, error) {
			if readStarted {
				return nil, failure
			}
			return base.Stat(name)
		}
		err := NewSorter(&Params{Inline: true}, filesystem).run("/src/main.tf")
		if !errors.Is(err, failure) {
			t.Fatalf("run error = %v, want metadata error", err)
		}
		fsWantContents(t, base, "/src/main.tf", sortedSingleBlock)
	})

	t.Run("output directory unavailable", func(t *testing.T) {
		base := afero.NewMemMapFs()
		fsPut(t, base, "/src/main.tf", unsortedTwoBlocks)
		failure := errors.New("cannot create output directory")
		filesystem := &fsFailure{Fs: base, mkdirAll: func(string, os.FileMode) error { return failure }}
		err := NewSorter(&Params{OutputDir: "/out"}, filesystem).run("/src/main.tf")
		if !errors.Is(err, failure) {
			t.Fatalf("run error = %v, want output error", err)
		}
		fsWantContents(t, base, "/src/main.tf", unsortedTwoBlocks)
		if _, err := base.Stat("/out"); !os.IsNotExist(err) {
			t.Fatalf("output directory unexpectedly exists: %v", err)
		}
	})
}

func TestRecursiveFilesystemFailures(t *testing.T) {
	for _, operation := range []string{"walk stat", "list directory", "read source", "write output"} {
		t.Run(operation, func(t *testing.T) {
			base := afero.NewMemMapFs()
			fsPut(t, base, "/src/nested/main.tf", sortedSingleBlock)
			failure := errors.New("recursive operation denied")
			filesystem := &fsFailure{Fs: base}
			switch operation {
			case "walk stat":
				filesystem.stat = func(name string) (os.FileInfo, error) {
					if name == "/src/nested" {
						return nil, failure
					}
					return base.Stat(name)
				}
			case "list directory":
				filesystem.open = func(name string) (afero.File, error) {
					if name == "/src/nested" {
						return nil, failure
					}
					return base.Open(name)
				}
			case "read source":
				filesystem.open = func(name string) (afero.File, error) {
					if name == "/src/nested/main.tf" {
						return nil, failure
					}
					return base.Open(name)
				}
			case "write output":
				filesystem.mkdirAll = func(string, os.FileMode) error { return failure }
			}
			err := NewSorter(&Params{Recursive: true, OutputDir: "/out"}, filesystem).run("/src")
			if !errors.Is(err, failure) {
				t.Fatalf("run error = %v, want %v", err, failure)
			}
			fsWantContents(t, base, "/src/nested/main.tf", sortedSingleBlock)
			if _, err := base.Stat("/out/nested/main.tf"); !os.IsNotExist(err) {
				t.Fatalf("output file unexpectedly exists: %v", err)
			}
		})
	}
}

func TestRecursivePrintsEveryDirectoryWithoutWriting(t *testing.T) {
	base := afero.NewMemMapFs()
	first := []byte("variable \"first\" {\n  default = 1\n}\n")
	second := []byte("variable \"second\" {\n  default = 2\n}\n")
	fsPut(t, base, "/src/main.tf", first)
	fsPut(t, base, "/src/nested/main.tf", second)
	s := NewSorter(&Params{Recursive: true}, base)
	got, err := fsCaptureOutput(t, &os.Stdout, func() error { return s.run("/src") })
	if err != nil || got != string(first)+string(second) {
		t.Fatalf("recursive output = %q, error = %v", got, err)
	}
	fsWantContents(t, base, "/src/main.tf", first)
	fsWantContents(t, base, "/src/nested/main.tf", second)
}

func TestCheckAndDiffReadFailures(t *testing.T) {
	for _, mode := range []string{"check", "diff"} {
		t.Run(mode, func(t *testing.T) {
			base := afero.NewMemMapFs()
			fsPut(t, base, "/src/main.tf", unsortedTwoBlocks)
			failure := errors.New("original source unreadable")
			filesystem := &fsFailure{Fs: base, open: func(string) (afero.File, error) {
				return nil, failure
			}}
			s := NewSorter(&Params{Check: true}, filesystem)
			action := func() error {
				return s.runCheckMode("/src", []string{"/src/main.tf"}, map[string][]byte{"main.tf": sortedSingleBlock})
			}
			stream := &os.Stderr
			if mode == "diff" {
				stream = &os.Stdout
				action = func() error {
					return s.runDiffMode("/src", []string{"/src/main.tf"}, map[string][]byte{"main.tf": sortedSingleBlock})
				}
			}
			got, err := fsCaptureOutput(t, stream, action)
			if !errors.Is(err, failure) || errors.Is(err, ErrCheckFailed) || got != "" {
				t.Fatalf("output = %q, error = %v; want I/O failure, not changes", got, err)
			}
			fsWantContents(t, base, "/src/main.tf", unsortedTwoBlocks)
		})
	}
}

func TestCheckAndDiffReportUnresolvedOutput(t *testing.T) {
	for _, mode := range []string{"check", "diff"} {
		t.Run(mode, func(t *testing.T) {
			base := afero.NewMemMapFs()
			s := NewSorter(&Params{Check: true}, base)
			const key = "new.tf"
			abs, err := filepath.Abs(key)
			if err != nil {
				t.Fatal(err)
			}
			outputs := map[string][]byte{key: []byte("new content\n")}
			var got string
			if mode == "check" {
				got, err = fsCaptureOutput(t, &os.Stderr, func() error {
					return s.runCheckMode("/src", nil, outputs)
				})
				if !strings.Contains(got, abs) {
					t.Errorf("check output omitted new path: %q", got)
				}
			} else {
				got, err = fsCaptureOutput(t, &os.Stdout, func() error {
					return s.runDiffMode("/src", nil, outputs)
				})
				want := "--- " + abs + "\n+++ " + abs + "\n@@ -1,0 +1,1 @@\n+new content\n"
				if got != want {
					t.Errorf("diff = %q, want %q", got, want)
				}
			}
			if !errors.Is(err, ErrCheckFailed) || !strings.Contains(err.Error(), abs) {
				t.Fatalf("error = %v; want new-file change at %s", err, abs)
			}
			if _, err := base.Stat(key); !os.IsNotExist(err) {
				t.Fatalf("report mode created file: %v", err)
			}
		})
	}
}

func TestCheckAndDiffFallbackWhenWorkingDirectoryRemoved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not allow removal of the current directory")
	}
	for _, mode := range []string{"check unresolved", "check changed", "diff unresolved"} {
		t.Run(mode, func(t *testing.T) {
			removed := t.TempDir()
			t.Chdir(removed)
			if err := os.Remove(removed); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Getwd(); err == nil {
				t.Fatal("removed current directory unexpectedly resolves")
			}
			base := afero.NewMemMapFs()
			s := NewSorter(&Params{Check: true}, base)
			const key = "main.tf"
			outputs := map[string][]byte{key: []byte("new\n")}
			var got string
			var err error
			switch mode {
			case "check unresolved":
				got, err = fsCaptureOutput(t, &os.Stderr, func() error {
					return s.runCheckMode(".", nil, outputs)
				})
			case "check changed":
				fsPut(t, base, key, []byte("old\n"))
				got, err = fsCaptureOutput(t, &os.Stderr, func() error {
					return s.runCheckMode(".", []string{key}, outputs)
				})
				fsWantContents(t, base, key, []byte("old\n"))
			case "diff unresolved":
				got, err = fsCaptureOutput(t, &os.Stdout, func() error {
					return s.runDiffMode(".", nil, outputs)
				})
				want := "--- main.tf\n+++ main.tf\n@@ -1,0 +1,1 @@\n+new\n"
				if got != want {
					t.Errorf("diff = %q, want %q", got, want)
				}
			}
			if !errors.Is(err, ErrCheckFailed) || !strings.Contains(err.Error(), key) {
				t.Fatalf("error = %v; want relative-path change report", err)
			}
			if !strings.Contains(got, key) {
				t.Errorf("output omitted fallback path: %q", got)
			}
		})
	}
}
