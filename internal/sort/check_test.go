package sort

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// ─── Test fixtures ──────────────────────────────────────────────────────────

// sortedSingleBlock is a one-block file already in canonical sorted+formatted
// form. Because it is trivially sorted (one block) and the HCL is already
// hclwrite.Format-canonical, the byte comparison in check mode will pass.
var sortedSingleBlock = []byte("resource \"aws_instance\" \"alpha\" {\n  ami = \"ami-a\"\n}\n")

// unsortedTwoBlocks has two blocks in descending label order; the sorter
// will reorder them, so check mode must report a change.
var unsortedTwoBlocks = []byte("resource \"aws_instance\" \"beta\" {\n  ami = \"ami-b\"\n}\n\nresource \"aws_instance\" \"alpha\" {\n  ami = \"ami-a\"\n}\n")

// ─── Flag conflicts ─────────────────────────────────────────────────────────

func TestCheckFlagConflicts(t *testing.T) {
	t.Parallel()
	memFS := afero.NewMemMapFs()
	_ = memFS.MkdirAll("/test", 0755)
	_ = afero.WriteFile(memFS, "/test/main.tf", unsortedTwoBlocks, 0644)

	t.Run("check conflicts with output-dir", func(t *testing.T) {
		s := NewSorter(&Params{Check: true, OutputDir: "/out"}, memFS)
		err := s.run("/test")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		const want = "the check flag conflicts with the output-dir flag"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error containing %q, got: %v", want, err)
		}
	})

	// Per spec §2.7: check+inline is VALID — inline only affects write
	// destination, and check mode suppresses all writes. The unsorted file
	// on the memFS should cause ErrCheckFailed, not a flag-conflict error.
	t.Run("check with inline detects unsorted file", func(t *testing.T) {
		s := NewSorter(&Params{Check: true, Inline: true}, memFS)
		err := s.run("/test")
		if err == nil {
			t.Fatal("expected ErrCheckFailed, got nil")
		}
		if !errors.Is(err, ErrCheckFailed) {
			t.Fatalf("expected errors.Is(err, ErrCheckFailed), got: %v", err)
		}
	})
}

// ─── Check mode: already sorted ─────────────────────────────────────────────

func TestCheckModeAlreadySorted(t *testing.T) {
	t.Parallel()
	t.Run("single file", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		_ = memFS.MkdirAll("/sorted", 0755)
		_ = afero.WriteFile(memFS, "/sorted/main.tf", sortedSingleBlock, 0644)

		s := NewSorter(&Params{Check: true}, memFS)
		if err := s.run("/sorted/main.tf"); err != nil {
			t.Fatalf("expected nil for already-sorted single file, got: %v", err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		_ = memFS.MkdirAll("/sorted", 0755)
		_ = afero.WriteFile(memFS, "/sorted/alpha.tf", sortedSingleBlock, 0644)
		_ = afero.WriteFile(memFS, "/sorted/beta.tf",
			[]byte("resource \"aws_s3_bucket\" \"b\" {\n  bucket = \"my-bucket\"\n}\n"), 0644)

		s := NewSorter(&Params{Check: true}, memFS)
		if err := s.run("/sorted"); err != nil {
			t.Fatalf("expected nil for all-sorted directory, got: %v", err)
		}
	})
}

// ─── Check mode: unsorted file(s) ───────────────────────────────────────────

func TestCheckModeUnsortedFile(t *testing.T) {
	t.Parallel()
	t.Run("single file", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		_ = memFS.MkdirAll("/unsorted", 0755)
		_ = afero.WriteFile(memFS, "/unsorted/main.tf", unsortedTwoBlocks, 0644)

		s := NewSorter(&Params{Check: true}, memFS)
		err := s.run("/unsorted/main.tf")
		if err == nil {
			t.Fatal("expected ErrCheckFailed, got nil")
		}
		if !errors.Is(err, ErrCheckFailed) {
			t.Fatalf("expected errors.Is(err, ErrCheckFailed), got: %v", err)
		}
		if !strings.Contains(err.Error(), "main.tf") {
			t.Errorf("expected error to mention changed file, got: %v", err)
		}
	})

	t.Run("directory with one unsorted file", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		_ = memFS.MkdirAll("/mixed", 0755)
		_ = afero.WriteFile(memFS, "/mixed/sorted.tf", sortedSingleBlock, 0644)
		_ = afero.WriteFile(memFS, "/mixed/unsorted.tf", unsortedTwoBlocks, 0644)

		s := NewSorter(&Params{Check: true}, memFS)
		err := s.run("/mixed")
		if err == nil {
			t.Fatal("expected ErrCheckFailed, got nil")
		}
		if !errors.Is(err, ErrCheckFailed) {
			t.Fatalf("expected errors.Is(err, ErrCheckFailed), got: %v", err)
		}
		if !strings.Contains(err.Error(), "unsorted.tf") {
			t.Errorf("expected error to mention unsorted file, got: %v", err)
		}
	})
}

// ─── Check mode: stderr output ──────────────────────────────────────────────

func TestCheckModeStderrOutput(t *testing.T) {
	memFS := afero.NewMemMapFs()
	_ = memFS.MkdirAll("/check", 0755)
	_ = afero.WriteFile(memFS, "/check/main.tf", unsortedTwoBlocks, 0644)

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stderr = w

	s := NewSorter(&Params{Check: true}, memFS)
	_ = s.run("/check/main.tf")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = oldStderr

	stderr := buf.String()
	if !strings.Contains(stderr, "would be changed") {
		t.Errorf("expected stderr to contain 'would be changed', got: %q", stderr)
	}
	if !strings.Contains(stderr, "main.tf") {
		t.Errorf("expected stderr to mention changed file, got: %q", stderr)
	}
}

// ─── Check mode: does not write ─────────────────────────────────────────────

func TestCheckModeDoesNotWrite(t *testing.T) {
	t.Parallel()
	memFS := afero.NewMemMapFs()
	_ = memFS.MkdirAll("/nowrite", 0755)
	_ = afero.WriteFile(memFS, "/nowrite/main.tf", unsortedTwoBlocks, 0644)

	originalBytes, _ := afero.ReadFile(memFS, "/nowrite/main.tf")

	s := NewSorter(&Params{Check: true}, memFS)
	_ = s.run("/nowrite/main.tf")

	afterBytes, err := afero.ReadFile(memFS, "/nowrite/main.tf")
	if err != nil {
		t.Fatalf("could not read file after check: %v", err)
	}
	if !bytes.Equal(originalBytes, afterBytes) {
		t.Error("check mode modified the file; expected no writes")
	}
}

// ─── Check mode: --group-by-type ────────────────────────────────────────────

func TestCheckModeGroupByType(t *testing.T) {
	t.Parallel()
	t.Run("already grouped and sorted", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		_ = memFS.MkdirAll("/grouped", 0755)
		_ = memFS.MkdirAll(filepath.Join(os.TempDir(), "tforganize"), 0755)

		_ = afero.WriteFile(memFS, "/grouped/main.tf",
			[]byte("resource \"aws_instance\" \"a\" {\n  ami = \"ami-a\"\n}\n"), 0644)
		_ = afero.WriteFile(memFS, "/grouped/variables.tf",
			[]byte("variable \"env\" {\n  default = \"dev\"\n}\n"), 0644)

		s := NewSorter(&Params{Check: true, GroupByType: true}, memFS)
		if err := s.run("/grouped"); err != nil {
			t.Fatalf("expected nil for already-grouped-and-sorted, got: %v", err)
		}
	})

	t.Run("unsorted within group file", func(t *testing.T) {
		memFS := afero.NewMemMapFs()
		_ = memFS.MkdirAll("/grouped-unsorted", 0755)
		_ = memFS.MkdirAll(filepath.Join(os.TempDir(), "tforganize"), 0755)

		_ = afero.WriteFile(memFS, "/grouped-unsorted/variables.tf", []byte(
			"variable \"z_var\" {\n  default = \"z\"\n}\n\nvariable \"a_var\" {\n  default = \"a\"\n}\n",
		), 0644)

		s := NewSorter(&Params{Check: true, GroupByType: true}, memFS)
		err := s.run("/grouped-unsorted")
		if err == nil {
			t.Fatal("expected ErrCheckFailed, got nil")
		}
		if !errors.Is(err, ErrCheckFailed) {
			t.Fatalf("expected errors.Is(err, ErrCheckFailed), got: %v", err)
		}
	})
}

// ─── resolveOriginalPath ────────────────────────────────────────────────────

func TestResolveOriginalPathNoMatch(t *testing.T) {
	t.Parallel()
	s := NewSorter(&Params{}, afero.NewMemMapFs())
	inputFiles := []string{"/dir/main.tf", "/dir/variables.tf"}
	_, err := s.resolveOriginalPath("/dir", inputFiles, "nonexistent.tf")
	if err == nil {
		t.Fatal("expected error for non-matching output key, got nil")
	}
	if !strings.Contains(err.Error(), "no original file found") {
		t.Errorf("expected 'no original file found' error, got: %v", err)
	}
}

func TestResolveOriginalPathGroupByType(t *testing.T) {
	t.Parallel()
	s := NewSorter(&Params{GroupByType: true}, afero.NewMemMapFs())
	got, err := s.resolveOriginalPath("/dir", nil, "variables.tf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/dir", "variables.tf")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ─── Check mode: --remove-comments changes content ─────────────────────────

func TestCheckMode_WithRemoveComments(t *testing.T) {
	t.Parallel()
	memFS := afero.NewMemMapFs()
	_ = memFS.MkdirAll("/comments", 0755)

	// A single-block file with an inline comment. With RemoveComments: true
	// the sorted output will strip the comment, causing a byte difference.
	fileWithComment := []byte("resource \"aws_instance\" \"a\" {\n  ami = \"ami-a\" # inline comment\n}\n")
	_ = afero.WriteFile(memFS, "/comments/main.tf", fileWithComment, 0644)

	s := NewSorter(&Params{Check: true, RemoveComments: true}, memFS)
	err := s.run("/comments/main.tf")
	if err == nil {
		t.Fatal("expected ErrCheckFailed, got nil")
	}
	if !errors.Is(err, ErrCheckFailed) {
		t.Fatalf("expected errors.Is(err, ErrCheckFailed), got: %v", err)
	}
}

// ─── ErrCheckFailed sentinel semantics ─────────────────────────────────────

func TestErrCheckFailed_Sentinel(t *testing.T) {
	t.Parallel()
	memFS := afero.NewMemMapFs()
	_ = memFS.MkdirAll("/sentinel", 0755)
	_ = afero.WriteFile(memFS, "/sentinel/main.tf", unsortedTwoBlocks, 0644)

	s := NewSorter(&Params{Check: true}, memFS)
	err := s.run("/sentinel/main.tf")
	if err == nil {
		t.Fatal("expected ErrCheckFailed, got nil")
	}
	if !errors.Is(err, ErrCheckFailed) {
		t.Fatal("expected errors.Is(err, ErrCheckFailed) to be true")
	}
	if errors.Is(err, io.EOF) {
		t.Fatal("expected errors.Is(err, io.EOF) to be false")
	}
}

// ─── Check mode: --group-by-type with new file ─────────────────────────────

func TestCheckMode_GroupByType_NewFile(t *testing.T) {
	t.Parallel()
	memFS := afero.NewMemMapFs()
	_ = memFS.MkdirAll("/newfile", 0755)
	_ = memFS.MkdirAll(filepath.Join(os.TempDir(), "tforganize"), 0755)

	// Input has a variable block but target dir has no variables.tf.
	// GroupByType will produce a variables.tf output key; since the file
	// does not exist on disk it should be reported as "would change".
	_ = afero.WriteFile(memFS, "/newfile/main.tf", []byte(
		"variable \"env\" {\n  default = \"dev\"\n}\n",
	), 0644)

	s := NewSorter(&Params{Check: true, GroupByType: true}, memFS)
	err := s.run("/newfile")
	if err == nil {
		t.Fatal("expected ErrCheckFailed, got nil")
	}
	if !errors.Is(err, ErrCheckFailed) {
		t.Fatalf("expected errors.Is(err, ErrCheckFailed), got: %v", err)
	}
	if !strings.Contains(err.Error(), "variables.tf") {
		t.Errorf("expected error to mention variables.tf, got: %v", err)
	}
}

// ─── Check mode: multiple files all clean ──────────────────────────────────

func TestCheckMode_MultipleFiles_AllClean(t *testing.T) {
	t.Parallel()
	memFS := afero.NewMemMapFs()
	_ = memFS.MkdirAll("/allclean", 0755)

	// Three already-sorted single-block files — check should return nil.
	_ = afero.WriteFile(memFS, "/allclean/alpha.tf", sortedSingleBlock, 0644)
	_ = afero.WriteFile(memFS, "/allclean/beta.tf",
		[]byte("resource \"aws_s3_bucket\" \"b\" {\n  bucket = \"my-bucket\"\n}\n"), 0644)
	_ = afero.WriteFile(memFS, "/allclean/gamma.tf",
		[]byte("resource \"aws_iam_role\" \"r\" {\n  name = \"my-role\"\n}\n"), 0644)

	s := NewSorter(&Params{Check: true}, memFS)
	if err := s.run("/allclean"); err != nil {
		t.Fatalf("expected nil for all-clean directory, got: %v", err)
	}
}
