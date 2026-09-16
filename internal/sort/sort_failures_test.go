package sort

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// A source may disappear between parsing and the later comment/header read.
type hclVanishingFS struct {
	afero.Fs
	opens int
}

func (fs *hclVanishingFS) Open(name string) (afero.File, error) {
	fs.opens++
	if fs.opens > 1 {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}
	return fs.Fs.Open(name)
}

func TestSortFileHeaderReadFailure(t *testing.T) {
	mem := afero.NewMemMapFs()
	if err := afero.WriteFile(mem, "/main.tf", []byte("# header\nlocals {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	s := NewSorter(&Params{HasHeader: true}, &hclVanishingFS{Fs: mem})
	got, err := s.sortFile("/main.tf")
	if got != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("disappearing header source: output = %v, error = %v", got, err)
	}
}

func TestGroupedSortReadFailure(t *testing.T) {
	s := NewSorter(&Params{GroupByType: true}, afero.NewMemMapFs())
	got, err := s.sortFiles([]string{"/missing.tf"})
	if got != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing grouped input: output = %v, error = %v", got, err)
	}
}

func TestSortBodySourceReadFailures(t *testing.T) {
	for _, tt := range []struct {
		name           string
		content        string
		removeComments bool
	}{
		{"block comment source", "locals {}\n", false},
		{"attribute source", "locals {\n  value = 1\n}\n", true},
		{"nested attribute source", "resource \"test\" \"example\" {\n  nested {\n    value = 1\n  }\n}\n", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSorter(&Params{RemoveComments: tt.removeComments}, afero.NewMemMapFs())
			// Parsing itself does not retain the source lines. Reading them later
			// must report the missing file, including through nested blocks.
			body, err := s.parseHclBytes([]byte(tt.content), "/gone.tf")
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.sortBody(body, "/gone.tf")
			if got != nil || !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing source: output = %v, error = %v", got, err)
			}
		})
	}
}

func TestSortBytesRejectsInvalidPreservedHeader(t *testing.T) {
	got, err := SortBytes([]byte("locals {}\n"), "main.tf", &Params{
		KeepHeader:    true,
		HeaderPattern: "this is not an HCL comment",
	})
	if got != nil || err == nil || !strings.Contains(err.Error(), "main.tf") {
		t.Fatalf("invalid output header: output = %q, error = %v", got, err)
	}
}

func TestSortBodyRelativeSourceWithoutWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not permit removing the current directory")
	}
	s := NewSorter(&Params{RemoveComments: true}, afero.NewMemMapFs())
	body, err := s.parseHclBytes([]byte("locals {\n  value = 1\n}\n"), "relative.tf")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	got, err := s.sortBody(body, "relative.tf")
	if got != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unresolvable relative source: output = %v, error = %v", got, err)
	}
}

func TestLabeledPreMetaBlocksRetainStableOrder(t *testing.T) {
	// HCL permits multiple labeled blocks even where Terraform later imposes
	// schema restrictions. The sorter must retain equal-priority input order.
	input := "check \"example\" {\n  data \"test\" \"z\" {}\n  data \"test\" \"a\" {}\n}\n"
	want := "check \"example\" {\n  data \"test\" \"z\" {\n  }\n\n  data \"test\" \"a\" {\n  }\n}\n"
	got, err := SortBytes([]byte(input), "check.tf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("equal-priority blocks = %q, want %q", got, want)
	}
}
