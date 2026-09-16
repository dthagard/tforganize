package sort

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/afero"
)

// fsFailure injects errors at the filesystem boundary without changing the
// production sorter or the behavior of unrelated operations.
type fsFailure struct {
	afero.Fs
	open     func(string) (afero.File, error)
	openFile func(string, int, os.FileMode) (afero.File, error)
	stat     func(string) (os.FileInfo, error)
	mkdirAll func(string, os.FileMode) error
}

func (f *fsFailure) Open(name string) (afero.File, error) {
	if f.open != nil {
		return f.open(name)
	}
	return f.Fs.Open(name)
}

func (f *fsFailure) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if f.openFile != nil {
		return f.openFile(name, flag, perm)
	}
	return f.Fs.OpenFile(name, flag, perm)
}

func (f *fsFailure) Stat(name string) (os.FileInfo, error) {
	if f.stat != nil {
		return f.stat(name)
	}
	return f.Fs.Stat(name)
}

func (f *fsFailure) MkdirAll(name string, perm os.FileMode) error {
	if f.mkdirAll != nil {
		return f.mkdirAll(name, perm)
	}
	return f.Fs.MkdirAll(name, perm)
}

type fsReadFailure struct {
	afero.File
	err error
}

func (f *fsReadFailure) Read([]byte) (int, error) {
	return 0, f.err
}

type fsWriteFailure struct {
	afero.File
	err error
}

func (f *fsWriteFailure) Write([]byte) (int, error) {
	return 0, f.err
}

type fsCloseFailure struct {
	afero.File
	err error
}

func (f *fsCloseFailure) Close() error {
	return errors.Join(f.File.Close(), f.err)
}

func fsPut(t *testing.T, filesystem afero.Fs, path string, content []byte) {
	t.Helper()
	if err := filesystem.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(filesystem, path, content, 0644); err != nil {
		t.Fatal(err)
	}
}

func fsWantContents(t *testing.T, filesystem afero.Fs, path string, want []byte) {
	t.Helper()
	got, err := afero.ReadFile(filesystem, path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

func TestFilesystemDiscoveryFailures(t *testing.T) {
	base := afero.NewMemMapFs()
	fsPut(t, base, "/src/main.tf", sortedSingleBlock)
	denied := errors.New("directory access denied")

	t.Run("directory cannot be opened", func(t *testing.T) {
		filesystem := &fsFailure{Fs: base, open: func(string) (afero.File, error) {
			return nil, denied
		}}
		got, err := NewSorter(nil, filesystem).getFilesFromTarget("/src")
		if !errors.Is(err, denied) || got != nil {
			t.Fatalf("files = %v, error = %v; want no files and directory error", got, err)
		}
		fsWantContents(t, base, "/src/main.tf", sortedSingleBlock)
	})

	t.Run("invalid exclude on single file", func(t *testing.T) {
		s := NewSorter(&Params{Excludes: []string{"["}}, base)
		got, err := s.getFilesFromTarget("/src/main.tf")
		if err == nil || got != nil {
			t.Fatalf("files = %v, error = %v; want no files and invalid-pattern error", got, err)
		}
	})

	t.Run("missing output source directory", func(t *testing.T) {
		got, err := NewSorter(nil, base).getDirectory("/missing/main.tf")
		if !os.IsNotExist(errors.Unwrap(err)) || got != "" {
			t.Fatalf("directory = %q, error = %v; want missing-path error", got, err)
		}
	})

	t.Run("mixed absolute and relative exclude paths", func(t *testing.T) {
		s := NewSorter(&Params{Excludes: []string{"main.tf"}}, base)
		got, err := s.isExcluded("/src", "nested/main.tf")
		if err != nil || !got {
			t.Fatalf("excluded = %v, error = %v; want basename match", got, err)
		}
	})
}

func TestFilesystemReadFailuresDoNotCachePartialResults(t *testing.T) {
	for _, operation := range []string{"open", "read", "line too long"} {
		t.Run(operation, func(t *testing.T) {
			base := afero.NewMemMapFs()
			path := "/src/main.tf"
			content := []byte("first\nsecond\n")
			fsPut(t, base, path, content)
			failure := errors.New("input unavailable")
			filesystem := &fsFailure{Fs: base}
			switch operation {
			case "open":
				filesystem.open = func(string) (afero.File, error) { return nil, failure }
			case "read":
				filesystem.open = func(name string) (afero.File, error) {
					file, err := base.Open(name)
					if err != nil {
						return nil, err
					}
					return &fsReadFailure{File: file, err: failure}, nil
				}
			case "line too long":
				failure = bufio.ErrTooLong
				fsPut(t, base, path, make([]byte, 1<<20))
			}
			s := NewSorter(nil, filesystem)
			lines, err := s.getLinesFromFile(path)
			if !errors.Is(err, failure) || lines != nil {
				t.Fatalf("lines = %v, error = %v; want no partial result and %v", lines, err, failure)
			}
			// A failed read must not poison subsequent reads on this sorter.
			filesystem.open = nil
			fsPut(t, base, path, content)
			lines, err = s.getLinesFromFile(path)
			if err != nil || !reflect.DeepEqual(lines, []string{"first", "second"}) {
				t.Fatalf("retry lines = %v, error = %v", lines, err)
			}
		})
	}
}

func TestCombineFilesReadFailureReturnsNoPartialOutput(t *testing.T) {
	base := afero.NewMemMapFs()
	fsPut(t, base, "/src/first.tf", []byte("first\n"))
	fsPut(t, base, "/src/second.tf", []byte("second\n"))
	failure := errors.New("second file unreadable")
	filesystem := &fsFailure{Fs: base, open: func(name string) (afero.File, error) {
		if name == "/src/second.tf" {
			return nil, failure
		}
		return base.Open(name)
	}}
	s := NewSorter(nil, filesystem)
	got, err := s.combineFiles([]string{"/src/first.tf", "/src/second.tf"})
	if !errors.Is(err, failure) || got != nil {
		t.Fatalf("combined = %q, error = %v; want no partial output and read error", got, err)
	}
	filesystem.open = nil
	got, err = s.combineFiles([]string{"/src/first.tf", "/src/second.tf"})
	if err != nil || string(got) != "first\nsecond\n" {
		t.Fatalf("combined = %q, error = %v", got, err)
	}
	fsWantContents(t, base, "/src/first.tf", []byte("first\n"))
	fsWantContents(t, base, "/src/second.tf", []byte("second\n"))
}

func TestWriteFilesFailuresPreserveInputs(t *testing.T) {
	for _, operation := range []string{"mkdir", "create", "write"} {
		t.Run(operation, func(t *testing.T) {
			base := afero.NewMemMapFs()
			fsPut(t, base, "/src/main.tf", unsortedTwoBlocks)
			failure := errors.New("output unavailable")
			filesystem := &fsFailure{Fs: base}
			switch operation {
			case "mkdir":
				filesystem.mkdirAll = func(string, os.FileMode) error { return failure }
			case "create":
				filesystem.openFile = func(string, int, os.FileMode) (afero.File, error) {
					return nil, failure
				}
			case "write":
				filesystem.openFile = func(name string, flags int, mode os.FileMode) (afero.File, error) {
					file, err := base.OpenFile(name, flags, mode)
					if err != nil {
						return nil, err
					}
					return &fsWriteFailure{File: file, err: failure}, nil
				}
			}
			s := NewSorter(&Params{OutputDir: "/out"}, filesystem)
			err := s.writeFiles(map[string][]byte{"main.tf": sortedSingleBlock})
			if !errors.Is(err, failure) {
				t.Fatalf("writeFiles error = %v, want %v", err, failure)
			}
			fsWantContents(t, base, "/src/main.tf", unsortedTwoBlocks)
			if operation == "write" {
				fsWantContents(t, base, "/out/main.tf", []byte{})
			} else if _, err := base.Stat("/out/main.tf"); !os.IsNotExist(err) {
				t.Fatalf("unexpected output file: %v", err)
			}
		})
	}
}

func TestWriteFileReportsCloseFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	fsPut(t, base, "/out/main.tf", []byte("old content\n"))
	failure := errors.New("output flush failed")
	filesystem := &fsFailure{Fs: base, openFile: func(name string, flags int, mode os.FileMode) (afero.File, error) {
		file, err := base.OpenFile(name, flags, mode)
		if err != nil {
			return nil, err
		}
		return &fsCloseFailure{File: file, err: failure}, nil
	}}
	err := NewSorter(nil, filesystem).writeFile("/out/main.tf", sortedSingleBlock)
	if !errors.Is(err, failure) {
		t.Fatalf("writeFile error = %v, want flush failure", err)
	}
	fsWantContents(t, base, "/out/main.tf", sortedSingleBlock)
}

func TestWriteFilePreservesWriteErrorWhenCloseFails(t *testing.T) {
	base := afero.NewMemMapFs()
	fsPut(t, base, "/out/main.tf", []byte("old content\n"))
	writeFailure := errors.New("output write failed")
	closeFailure := errors.New("output flush failed")
	filesystem := &fsFailure{Fs: base, openFile: func(name string, flags int, mode os.FileMode) (afero.File, error) {
		file, err := base.OpenFile(name, flags, mode)
		if err != nil {
			return nil, err
		}
		return &fsWriteFailure{
			File: &fsCloseFailure{File: file, err: closeFailure},
			err:  writeFailure,
		}, nil
	}}
	err := NewSorter(nil, filesystem).writeFile("/out/main.tf", sortedSingleBlock)
	if !errors.Is(err, writeFailure) || errors.Is(err, closeFailure) {
		t.Fatalf("writeFile error = %v, want primary write failure", err)
	}
	fsWantContents(t, base, "/out/main.tf", []byte{})
}
