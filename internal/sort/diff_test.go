package sort

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitLines(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		got := splitLines("")
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("single line with trailing newline", func(t *testing.T) {
		got := splitLines("hello\n")
		want := []string{"hello\n"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("single line without trailing newline", func(t *testing.T) {
		got := splitLines("hello")
		want := []string{"hello"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("multiple lines", func(t *testing.T) {
		got := splitLines("a\nb\nc\n")
		want := []string{"a\n", "b\n", "c\n"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("multiple lines without trailing newline", func(t *testing.T) {
		got := splitLines("a\nb\nc")
		want := []string{"a\n", "b\n", "c"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestComputeEdits(t *testing.T) {
	t.Run("both empty", func(t *testing.T) {
		edits := computeEdits(nil, nil)
		if len(edits) != 0 {
			t.Errorf("expected no edits, got %d", len(edits))
		}
	})

	t.Run("a empty b non-empty", func(t *testing.T) {
		edits := computeEdits(nil, []string{"x\n", "y\n"})
		for _, e := range edits {
			if e.kind != editInsert {
				t.Errorf("expected all inserts, got kind %d", e.kind)
			}
		}
		if len(edits) != 2 {
			t.Errorf("expected 2 inserts, got %d", len(edits))
		}
	})

	t.Run("a non-empty b empty", func(t *testing.T) {
		edits := computeEdits([]string{"x\n", "y\n"}, nil)
		for _, e := range edits {
			if e.kind != editDelete {
				t.Errorf("expected all deletes, got kind %d", e.kind)
			}
		}
		if len(edits) != 2 {
			t.Errorf("expected 2 deletes, got %d", len(edits))
		}
	})

	t.Run("identical inputs", func(t *testing.T) {
		lines := []string{"a\n", "b\n", "c\n"}
		edits := computeEdits(lines, lines)
		for _, e := range edits {
			if e.kind != editEqual {
				t.Errorf("expected all equal, got kind %d", e.kind)
			}
		}
		if len(edits) != 3 {
			t.Errorf("expected 3 edits, got %d", len(edits))
		}
	})

	t.Run("single line changed", func(t *testing.T) {
		a := []string{"a\n", "b\n", "c\n"}
		b := []string{"a\n", "x\n", "c\n"}
		edits := computeEdits(a, b)

		var deletes, inserts, equals int
		for _, e := range edits {
			switch e.kind {
			case editDelete:
				deletes++
			case editInsert:
				inserts++
			case editEqual:
				equals++
			}
		}
		if equals != 2 {
			t.Errorf("expected 2 equals, got %d", equals)
		}
		if deletes != 1 {
			t.Errorf("expected 1 delete, got %d", deletes)
		}
		if inserts != 1 {
			t.Errorf("expected 1 insert, got %d", inserts)
		}
	})
}

func TestGroupEdits(t *testing.T) {
	t.Run("all equal returns nil", func(t *testing.T) {
		lines := []string{"a\n", "b\n"}
		edits := []edit{
			{kind: editEqual, aLine: 0, bLine: 0},
			{kind: editEqual, aLine: 1, bLine: 1},
		}
		hunks := groupEdits(edits, lines, lines, 3)
		if hunks != nil {
			t.Errorf("expected nil, got %v", hunks)
		}
	})

	t.Run("single change produces one hunk", func(t *testing.T) {
		a := []string{"a\n", "b\n", "c\n"}
		b := []string{"a\n", "x\n", "c\n"}
		edits := computeEdits(a, b)
		hunks := groupEdits(edits, a, b, 3)
		if len(hunks) != 1 {
			t.Fatalf("expected 1 hunk, got %d", len(hunks))
		}
		if !strings.Contains(hunks[0], "@@") {
			t.Error("expected hunk header with @@")
		}
		if !strings.Contains(hunks[0], "-b\n") {
			t.Error("expected deletion of b")
		}
		if !strings.Contains(hunks[0], "+x\n") {
			t.Error("expected insertion of x")
		}
	})

	t.Run("distant changes produce separate hunks", func(t *testing.T) {
		// 10 lines with changes at positions 0 and 9 — more than 2*context apart.
		a := make([]string, 10)
		b := make([]string, 10)
		for i := range a {
			a[i] = "same\n"
			b[i] = "same\n"
		}
		a[0] = "old-first\n"
		b[0] = "new-first\n"
		a[9] = "old-last\n"
		b[9] = "new-last\n"

		edits := computeEdits(a, b)
		hunks := groupEdits(edits, a, b, 1) // context=1 to keep hunks small
		if len(hunks) < 2 {
			t.Errorf("expected at least 2 separate hunks, got %d", len(hunks))
		}
	})

	t.Run("close changes merge into one hunk", func(t *testing.T) {
		a := []string{"a\n", "b\n", "c\n", "d\n", "e\n"}
		b := []string{"A\n", "b\n", "c\n", "d\n", "E\n"}
		edits := computeEdits(a, b)
		hunks := groupEdits(edits, a, b, 3) // context=3 merges everything
		if len(hunks) != 1 {
			t.Errorf("expected 1 merged hunk, got %d", len(hunks))
		}
	})
}

func TestEnsureNewline(t *testing.T) {
	t.Run("adds newline when missing", func(t *testing.T) {
		var b strings.Builder
		b.WriteString("hello")
		ensureNewline(&b)
		if !strings.HasSuffix(b.String(), "\n") {
			t.Errorf("expected trailing newline, got %q", b.String())
		}
	})

	t.Run("does not double newline", func(t *testing.T) {
		var b strings.Builder
		b.WriteString("hello\n")
		ensureNewline(&b)
		if strings.HasSuffix(b.String(), "\n\n") {
			t.Error("should not add extra newline")
		}
	})

	t.Run("empty builder is a no-op", func(t *testing.T) {
		var b strings.Builder
		ensureNewline(&b)
		if b.String() != "" {
			t.Errorf("expected empty, got %q", b.String())
		}
	})
}
