package sort

import (
	"path/filepath"
	"sort"
	"testing"

	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/afero"
)

func BenchmarkSortSingleFile(b *testing.B) {
	unsortedPath := filepath.Join("testdata", "single_file", "unsorted", "main.tf")

	osFs := afero.NewOsFs()
	afs := &afero.Afero{Fs: osFs}

	if _, err := afs.Stat(unsortedPath); err != nil {
		b.Fatalf("testdata file not found: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewSorter(&Params{}, osFs)
		if _, err := s.sortFile(unsortedPath); err != nil {
			b.Fatalf("sortFile returned error: %v", err)
		}
	}
}

func BenchmarkBlockListSorter(b *testing.B) {
	types := []string{"resource", "data", "variable", "output", "module", "locals"}
	labels := []string{"zulu", "yankee", "x_ray", "whiskey", "victor", "uniform", "tango", "sierra"}

	var blocks []*hclsyntax.Block
	for _, typ := range types {
		for _, lbl := range labels {
			blocks = append(blocks, &hclsyntax.Block{
				Type:   typ,
				Labels: []string{typ + "_" + lbl, lbl},
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp := make([]*hclsyntax.Block, len(blocks))
		copy(cp, blocks)
		sort.Stable(BlockListSorter(cp))
	}
}
