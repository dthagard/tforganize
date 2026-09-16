package sort

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestHCLReadAndParseFailures(t *testing.T) {
	s := NewSorter(nil, afero.NewMemMapFs())
	body, err := s.parseHclFile("/missing.tf")
	if body != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing input: body = %v, error = %v", body, err)
	}
	if err := s.detectFileHeader("/missing.tf"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing header input: error = %v", err)
	}

	output, err := SortBytes([]byte("resource \"unfinished\" {"), "malformed.tf", nil)
	if output != nil || err == nil || !strings.Contains(err.Error(), "malformed.tf") {
		t.Fatalf("malformed input: output = %q, error = %v", output, err)
	}
}

func TestHeaderTerminatesBeforeFollowingComment(t *testing.T) {
	for _, tt := range []struct {
		name   string
		header string
		end    string
	}{
		{"single line block", "/* Copyright Example */", ""},
		{"explicit block terminator", "/* Copyright Example END */", "END"},
		{"explicit line terminator", "// Copyright Example END", "END"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSorter(&Params{HeaderPattern: "Copyright", HeaderEndPattern: tt.end}, afero.NewMemMapFs())
			got := s.findHeaderInLines([]string{tt.header, "# belongs to the resource", "resource \"x\" \"y\" {}"})
			if got != tt.header {
				t.Fatalf("header = %q, want %q", got, tt.header)
			}
		})
	}
}

func TestDetectedHeaderDoesNotConsumeOtherBlockComment(t *testing.T) {
	s := NewSorter(&Params{HasHeader: true, HeaderPattern: "Copyright"}, afero.NewMemMapFs())
	s.detectedHeaders["main.tf"] = "# Copyright Example"
	lines := []string{"# Resource documentation", "# Keep this explanation"}
	if got := s.removeHeader(lines, "main.tf"); !reflect.DeepEqual(got, lines) {
		t.Fatalf("unrelated comment = %#v, want %#v", got, lines)
	}
}

func TestTrailingEmptyLinesPreserveOneSeparator(t *testing.T) {
	lines := []string{"# explanation", "", " ", "\t"}
	want := []string{"# explanation", ""}
	if got := removeTrailingEmptyLines(lines); !reflect.DeepEqual(got, want) {
		t.Fatalf("trimmed comment = %#v, want %#v", got, want)
	}
}

func TestAttributeCommentsRemainAttachedWhenSorted(t *testing.T) {
	input := "locals {\n  z = 2\n  # explains a\n  a = 1\n}\n"
	want := "locals {\n  # explains a\n  a = 1\n  z = 2\n}\n"
	got, err := SortBytes([]byte(input), "comments.tf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("sorted output = %q, want %q", got, want)
	}
}
