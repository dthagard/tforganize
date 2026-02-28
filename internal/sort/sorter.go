package sort

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	gosort "sort"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/spf13/afero"
)

// Sorter holds all per-run state for a single Sort execution.
// It is safe to use concurrently; each Sort call creates its own Sorter.
type Sorter struct {
	params     *Params
	fs         afero.Fs
	afs        *afero.Afero
	mu         sync.Mutex
	linesCache map[string][]string
	// detectedHeaders maps input file paths to their detected header text.
	// Populated by detectFileHeader before sorting; used by removeHeader
	// and addHeader to handle the complete header regardless of whether
	// HeaderPattern is a full match or a partial substring.
	detectedHeaders   map[string]string
	detectedHeadersMu sync.Mutex
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
		params:          &paramsCopy,
		fs:              fs,
		afs:             &afero.Afero{Fs: fs},
		linesCache:      make(map[string][]string),
		detectedHeaders: make(map[string]string),
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
	if s.params.Diff && s.params.OutputDir != "" {
		return fmt.Errorf("the diff flag conflicts with the output-dir flag")
	}
	if s.params.Diff && s.params.Inline {
		return fmt.Errorf("the diff flag conflicts with the inline flag")
	}

	// 1a. Validate exclude glob patterns.
	for _, p := range s.params.Excludes {
		if !doublestar.ValidatePattern(p) {
			return fmt.Errorf("invalid exclude pattern %q", p)
		}
	}

	// 2. Handle recursive mode: process each directory independently.
	if s.params.Recursive {
		info, err := s.getPathInfo(target)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("the recursive flag requires a directory target")
		}
		return s.runRecursive(target)
	}

	return s.runSingle(target)
}

// runSingle processes a single target (file or directory).
func (s *Sorter) runSingle(target string) error {
	// Resolve target files
	files, err := s.getFilesFromTarget(target)
	if err != nil {
		return fmt.Errorf("could not get files from target: %w", err)
	}

	// Sort
	sortedFiles, err := s.sortFiles(files)
	if err != nil {
		return fmt.Errorf("could not sort files: %w", err)
	}

	// Diff mode — show unified diff of changes
	if s.params.Diff {
		return s.runDiffMode(target, files, sortedFiles)
	}

	// Check mode — compare and return without writing
	if s.params.Check {
		return s.runCheckMode(target, files, sortedFiles)
	}

	// Resolve output dir for inline mode
	if s.params.Inline {
		s.params.OutputDir, err = s.getDirectory(target)
		if err != nil {
			return fmt.Errorf("could not get directory for the target: %w", err)
		}
	}

	// Write or print
	if s.params.OutputDir != "" {
		if err := s.writeFiles(sortedFiles); err != nil {
			return fmt.Errorf("could not write files: %w", err)
		}
	} else {
		for _, body := range sortedFiles {
			fmt.Print(string(body))
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

// runRecursive walks the target directory recursively, processing each
// sub-directory that contains .tf files independently.
func (s *Sorter) runRecursive(target string) error {
	var firstCheckErr error

	err := afero.Walk(s.fs, target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		// Check if this directory has .tf files.
		files, err := s.getFilesInFolder(path)
		if err != nil {
			return fmt.Errorf("could not get files in %s: %w", path, err)
		}
		if len(files) == 0 {
			return nil
		}

		// Create a per-directory sorter with fresh cache and appropriate OutputDir.
		dirParams := *s.params
		dirParams.Recursive = false // prevent infinite recursion
		if dirParams.Inline {
			dirParams.OutputDir = path
		} else if dirParams.OutputDir != "" {
			// Mirror directory structure in the output dir.
			rel, relErr := filepath.Rel(target, path)
			if relErr == nil {
				dirParams.OutputDir = filepath.Join(s.params.OutputDir, rel)
			}
		}

		dirSorter := NewSorter(&dirParams, s.fs)
		sortedFiles, sortErr := dirSorter.sortFiles(files)
		if sortErr != nil {
			return fmt.Errorf("could not sort files in %s: %w", path, sortErr)
		}

		if dirParams.Check {
			if checkErr := dirSorter.runCheckMode(path, files, sortedFiles); checkErr != nil {
				if firstCheckErr == nil {
					firstCheckErr = checkErr
				}
			}
			return nil
		}

		if dirParams.OutputDir != "" {
			if writeErr := dirSorter.writeFiles(sortedFiles); writeErr != nil {
				return fmt.Errorf("could not write files in %s: %w", path, writeErr)
			}
		} else {
			for _, body := range sortedFiles {
				fmt.Print(string(body))
			}
		}

		return nil
	})

	if err != nil {
		return err
	}
	return firstCheckErr
}

// runDiffMode prints a unified diff for each file that would change and
// optionally returns ErrCheckFailed when combined with --check.
func (s *Sorter) runDiffMode(target string, inputFiles []string, sortedFiles map[string][]byte) error {
	var changed []string

	// Collect sorted output keys deterministically.
	keys := make([]string, 0, len(sortedFiles))
	for k := range sortedFiles {
		keys = append(keys, k)
	}
	gosort.Strings(keys)

	for _, outputKey := range keys {
		sortedBytes := sortedFiles[outputKey]

		originalPath, err := s.resolveOriginalPath(target, inputFiles, outputKey)
		if err != nil {
			// New file would be created — show entire content as additions.
			absPath, absErr := filepath.Abs(outputKey)
			if absErr != nil {
				absPath = outputKey
			}
			diff := unifiedDiff(absPath, absPath, "", string(sortedBytes))
			if diff != "" {
				fmt.Print(diff)
				changed = append(changed, absPath)
			}
			continue
		}

		originalBytes, err := s.afs.ReadFile(originalPath)
		if err != nil {
			if os.IsNotExist(err) {
				diff := unifiedDiff(originalPath, originalPath, "", string(sortedBytes))
				if diff != "" {
					fmt.Print(diff)
					changed = append(changed, originalPath)
				}
				continue
			}
			return fmt.Errorf("diff: could not read original file %s: %w", originalPath, err)
		}

		diff := unifiedDiff(originalPath, originalPath, string(originalBytes), string(sortedBytes))
		if diff != "" {
			fmt.Print(diff)
			changed = append(changed, originalPath)
		}
	}

	if s.params.Check && len(changed) > 0 {
		return fmt.Errorf("%w: %s", ErrCheckFailed, strings.Join(changed, ", "))
	}

	return nil
}
