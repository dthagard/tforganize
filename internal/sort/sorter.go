package sort

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	gosort "sort"
	"strings"

	"github.com/spf13/afero"
)

// Sorter holds all per-run state for a single Sort execution.
// It is safe to use concurrently; each Sort call creates its own Sorter.
type Sorter struct {
	params     *Params
	fs         afero.Fs
	afs        *afero.Afero
	linesCache map[string][]string
}

// NewSorter constructs a Sorter for a single sort run.
//
// params must not be nil; pass &Params{} for zero-value defaults.
// fs must not be nil; pass afero.NewOsFs() for production use.
//
// NewSorter makes a shallow copy of *params so the caller's struct is
// never mutated.
func NewSorter(params *Params, fs afero.Fs) *Sorter {
	if params == nil {
		params = &Params{}
	}
	paramsCopy := *params // copy, not pointer share
	return &Sorter{
		params:     &paramsCopy,
		fs:         fs,
		afs:        &afero.Afero{Fs: fs},
		linesCache: make(map[string][]string),
	}
}

// run is the internal entry point for a sort execution.
func (s *Sorter) run(target string) error {
	// 1. Validate flag combinations
	if s.params.Inline && (s.params.GroupByType || s.params.OutputDir != "") {
		return fmt.Errorf("the inline flag conflicts with the group-by-type and output-dir flags")
	}
	if s.params.KeepHeader && (!s.params.HasHeader || s.params.HeaderPattern == "") {
		return fmt.Errorf("keep-header requires has-header=true and a non-empty header-pattern")
	}
	if s.params.Check && s.params.OutputDir != "" {
		return fmt.Errorf("the check flag conflicts with the output-dir flag")
	}

	// 2. Resolve target files
	files, err := s.getFilesFromTarget(target)
	if err != nil {
		return fmt.Errorf("could not get files from target: %w", err)
	}

	// 3. Sort
	sortedFiles, err := s.sortFiles(files)
	if err != nil {
		return fmt.Errorf("could not sort files: %w", err)
	}

	// 4. Check mode — compare and return without writing
	if s.params.Check {
		return s.runCheckMode(target, files, sortedFiles)
	}

	// 5. Resolve output dir for inline mode
	if s.params.Inline {
		s.params.OutputDir, err = s.getDirectory(target)
		if err != nil {
			return fmt.Errorf("could not get directory for the target: %w", err)
		}
	}

	// 6. Write or print
	if s.params.OutputDir != "" {
		if err := s.writeFiles(sortedFiles); err != nil {
			return fmt.Errorf("could not write files: %w", err)
		}
	} else {
		for _, body := range sortedFiles {
			fmt.Println(string(body))
		}
	}

	return nil
}

// runCheckMode compares the sorted output against the current file contents and
// returns ErrCheckFailed (wrapped with the list of differing files) when any
// file would change.
//
// target is the original sort target (file or directory path).
// inputFiles is the list of resolved input file paths passed to sortFiles.
// sortedFiles is the map[basename][]byte returned by sortFiles.
func (s *Sorter) runCheckMode(target string, inputFiles []string, sortedFiles map[string][]byte) error {
	var changed []string

	for outputKey, sortedBytes := range sortedFiles {
		// Resolve the path of the file we're comparing against.
		originalPath, err := s.resolveOriginalPath(target, inputFiles, outputKey)
		if err != nil {
			// File does not exist in its expected location — it would be created.
			// Ensure the path is absolute per spec §2.8.
			absPath, absErr := filepath.Abs(outputKey)
			if absErr != nil {
				absPath = outputKey
			}
			changed = append(changed, absPath)
			continue
		}

		originalBytes, err := s.afs.ReadFile(originalPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Original file doesn't exist yet — it would be created by sorting.
				changed = append(changed, originalPath)
				continue
			}
			return fmt.Errorf("check: could not read original file %s: %w", originalPath, err)
		}

		if !bytes.Equal(originalBytes, sortedBytes) {
			// Ensure the path is absolute per spec §2.8.
			absPath, absErr := filepath.Abs(originalPath)
			if absErr != nil {
				absPath = originalPath
			}
			changed = append(changed, absPath)
		}
	}

	if len(changed) == 0 {
		return nil
	}

	gosort.Strings(changed) // deterministic output order
	fmt.Fprintln(os.Stderr, "The following files would be changed by tforganize sort:")
	for _, f := range changed {
		fmt.Fprintf(os.Stderr, "  - %s\n", f)
	}
	fmt.Fprintln(os.Stderr, "\nRun 'tforganize sort <target>' to sort these files.")

	return fmt.Errorf("%w: %s", ErrCheckFailed, strings.Join(changed, ", "))
}

// resolveOriginalPath maps an outputKey (basename) from sortedFiles back to
// the full path of the original source file.
//
// When --group-by-type is set, sorted output keys are canonical group filenames
// (e.g. "variables.tf"). The original file is resolved as filepath.Join(target, outputKey).
//
// Otherwise, the original file is found by matching outputKey against the
// basename of each path in inputFiles.
//
// Returns an error if no matching original file can be found.
func (s *Sorter) resolveOriginalPath(target string, inputFiles []string, outputKey string) (string, error) {
	if s.params.GroupByType {
		// In group-by-type mode the canonical output filename lives directly
		// in the target directory.
		return filepath.Join(target, outputKey), nil
	}

	// Non-grouped: find the input file whose base name matches the output key.
	for _, f := range inputFiles {
		if filepath.Base(f) == outputKey {
			return f, nil
		}
	}

	return "", fmt.Errorf("check: no original file found for output key %q", outputKey)
}
