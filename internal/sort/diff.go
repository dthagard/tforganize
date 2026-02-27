package sort

import (
	"fmt"
	"strings"
)

// unifiedDiff generates a unified diff between two strings, similar to
// `diff -u`. Returns an empty string when a and b are identical.
func unifiedDiff(aName, bName, a, b string) string {
	if a == b {
		return ""
	}

	aLines := splitLines(a)
	bLines := splitLines(b)

	// Simple Myers-like diff using the longest common subsequence.
	edits := computeEdits(aLines, bLines)

	var buf strings.Builder
	fmt.Fprintf(&buf, "--- %s\n", aName)
	fmt.Fprintf(&buf, "+++ %s\n", bName)

	// Generate hunks from the edit script.
	hunks := groupEdits(edits, aLines, bLines, 3)
	for _, h := range hunks {
		buf.WriteString(h)
	}

	return buf.String()
}

type editKind int

const (
	editEqual  editKind = iota
	editDelete          // line only in a
	editInsert          // line only in b
)

type edit struct {
	kind  editKind
	aLine int // 0-indexed line in a (-1 for inserts)
	bLine int // 0-indexed line in b (-1 for deletes)
}

// splitLines splits s into lines, preserving trailing newlines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	// SplitAfter may leave an empty trailing element.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// computeEdits produces an edit script using a simple O(NM) LCS approach.
// Good enough for typical Terraform files.
func computeEdits(a, b []string) []edit {
	n := len(a)
	m := len(b)

	// Build LCS table.
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	// Backtrack to produce edits.
	var edits []edit
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			edits = append(edits, edit{kind: editEqual, aLine: i, bLine: j})
			i++
			j++
		} else if lcs[i+1][j] >= lcs[i][j+1] {
			edits = append(edits, edit{kind: editDelete, aLine: i, bLine: -1})
			i++
		} else {
			edits = append(edits, edit{kind: editInsert, aLine: -1, bLine: j})
			j++
		}
	}
	for ; i < n; i++ {
		edits = append(edits, edit{kind: editDelete, aLine: i, bLine: -1})
	}
	for ; j < m; j++ {
		edits = append(edits, edit{kind: editInsert, aLine: -1, bLine: j})
	}

	return edits
}

// groupEdits groups edits into unified-diff hunks with `context` lines of
// surrounding equal lines.
func groupEdits(edits []edit, aLines, bLines []string, context int) []string {
	var hunks []string

	// Find ranges of non-equal edits.
	type hunkRange struct{ start, end int } // indices into edits
	var ranges []hunkRange
	i := 0
	for i < len(edits) {
		if edits[i].kind == editEqual {
			i++
			continue
		}
		start := i
		for i < len(edits) && edits[i].kind != editEqual {
			i++
		}
		ranges = append(ranges, hunkRange{start, i})
	}

	if len(ranges) == 0 {
		return nil
	}

	// Merge nearby ranges and emit hunks.
	for ri := 0; ri < len(ranges); {
		// Determine context-expanded range.
		hStart := ranges[ri].start - context
		if hStart < 0 {
			hStart = 0
		}
		hEnd := ranges[ri].end + context
		if hEnd > len(edits) {
			hEnd = len(edits)
		}

		// Merge overlapping ranges.
		ri++
		for ri < len(ranges) && ranges[ri].start-context <= hEnd {
			hEnd = ranges[ri].end + context
			if hEnd > len(edits) {
				hEnd = len(edits)
			}
			ri++
		}

		// Compute line numbers.
		aStart := 0
		bStart := 0
		if hStart < len(edits) {
			for k := 0; k < hStart; k++ {
				if edits[k].kind != editInsert {
					aStart++
				}
				if edits[k].kind != editDelete {
					bStart++
				}
			}
		}

		aCount := 0
		bCount := 0
		var body strings.Builder
		for k := hStart; k < hEnd; k++ {
			e := edits[k]
			switch e.kind {
			case editEqual:
				body.WriteString(" ")
				body.WriteString(aLines[e.aLine])
				ensureNewline(&body)
				aCount++
				bCount++
			case editDelete:
				body.WriteString("-")
				body.WriteString(aLines[e.aLine])
				ensureNewline(&body)
				aCount++
			case editInsert:
				body.WriteString("+")
				body.WriteString(bLines[e.bLine])
				ensureNewline(&body)
				bCount++
			}
		}

		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", aStart+1, aCount, bStart+1, bCount)
		hunks = append(hunks, header+body.String())
	}

	return hunks
}

// ensureNewline ensures the builder ends with a newline.
func ensureNewline(b *strings.Builder) {
	s := b.String()
	if len(s) > 0 && s[len(s)-1] != '\n' {
		b.WriteByte('\n')
	}
}
