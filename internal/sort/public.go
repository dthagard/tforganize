package sort

import (
	"github.com/spf13/afero"
)

// Params represents the parameters for the sort command.
type Params struct {
	// If the check flag is set, the sort command will not write any files.
	// Instead it reports which files would change and exits non-zero.
	// Conflicts with the output-dir flag.
	Check bool `yaml:"check"`
	// Excludes is a list of glob patterns. Files whose path matches any pattern
	// are skipped during sorting. Patterns support ** for cross-directory matching
	// (e.g. ".terraform/**", "*.generated.tf", "modules/*/"). Patterns are matched
	// against the path relative to the sort target directory. An invalid pattern
	// causes Sort to return an error immediately.
	Excludes []string `yaml:"exclude"`
	// If the group-by-type flag is set, the resources will be grouped by type in the output files.
	// Otherwise, the resources will be sorted alphabetically ascending by resource type and name in the existing files.
	// Conflicts with the inline flag.
	GroupByType bool `yaml:"group-by-type"`
	// If the has-header flag is set, the input files have a header.
	HasHeader bool `yaml:"has-header"`
	// If the header-pattern flag is set, the header pattern will be used to find the header in the input files.
	HeaderPattern string `yaml:"header-pattern"`
	// If the header-end-pattern flag is set, it marks the end of a multi-line header block.
	// When set together with header-pattern, everything from the first line matching
	// header-pattern to the first line matching header-end-pattern (inclusive) is treated
	// as the file header.
	HeaderEndPattern string `yaml:"header-end-pattern"`
	// If the inline flag is set, the resources will be sorted in place in the input files.
	// Conflicts with the group-by-type and output-dir flags.
	Inline bool `yaml:"inline"`
	// If the keep-header flag is set, the header matched in the header pattern will be persisted in the output files.
	KeepHeader bool `yaml:"keep-header"`
	// If the output directory is set, the sorted files will be written to the output directory.
	// Otherwise, the sorted files will be printed to stdout.
	// Conflicts with the inline flag.
	OutputDir string `yaml:"output-dir"`
	// If the recursive flag is set, nested directories are traversed.
	Recursive bool `yaml:"recursive"`
	// If the diff flag is set, a unified diff of changes is printed to stdout
	// instead of writing files.
	Diff bool `yaml:"diff"`
	// If NoSortByType is set, blocks are sorted alphabetically by type name
	// instead of using the logical type priority ordering (terraform → variable
	// → locals → data → resource → module → import → moved → removed → check
	// → output).
	NoSortByType bool `yaml:"no-sort-by-type"`
	// If the remove-comments flag is set, the comments will be removed from the files.
	// Otherwise, the comments will be preserved.
	RemoveComments bool `yaml:"remove-comments"`
}

// Sort sorts a Terraform file or folder.
//
// If the target is a folder, all files in the folder will be sorted.
// If the target is a file, only that file will be sorted.
// Sort returns an error when a fatal condition is encountered so callers can
// propagate it and exit non-zero.
func Sort(target string, settings *Params) error {
	s := NewSorter(settings, afero.NewOsFs())
	return s.run(target)
}

// SortBytes sorts raw HCL content and returns the sorted bytes.
// The filename parameter is used for error messages and HCL diagnostics.
func SortBytes(content []byte, filename string, settings *Params) ([]byte, error) {
	s := NewSorter(settings, afero.NewMemMapFs())
	results, err := s.sortFileBytes(content, filename)
	if err != nil {
		return nil, err
	}

	// Combine all output files into a single byte slice for stdout output.
	var output []byte
	for _, v := range results {
		if len(output) > 0 {
			output = append(output, '\n')
		}
		output = append(output, v...)
	}
	return output, nil
}
