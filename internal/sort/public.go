package sort

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// Params represents the parameters for the sort command.
type Params struct {
	// If the group-by-type flag is set, the resources will be grouped by type in the output files.
	// Otherwise, the resources will be sorted alphabetically ascending by resource type and name in the existing files.
	// Conflicts with the inline flag.
	GroupByType bool `yaml:"group-by-type"`
	// If the has-header flag is set, the input files have a header.
	HasHeader bool `yaml:"has-header"`
	// If the header-pattern flag is set, the header pattern will be used to find the header in the input files.
	HeaderPattern string `yaml:"header-pattern"`
	// If the inline flag is set, the resources will be sorted in place in the input files.
	// Conflicts with the group-by-type and output-dir flags.
	Inline bool `yaml:"inline"`
	// If the keep-header flag is set, the header matched in the header pattern will be persisted in the output files.
	KeepHeader bool `yaml:"keep-header"`
	// If the output directory is set, the sorted files will be written to the output directory.
	// Otherwise, the sorted files will be printed to stdout.
	// Conflicts with the inline flag.
	OutputDir string `yaml:"output-dir"`
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
	// Clear the file-lines cache at the start of every run so that files
	// read in a previous Sort call are not reused across runs.
	clearLinesCache()

	// Copy settings into the package-level params before any dereference so
	// that (a) nil settings is safe and (b) we never mutate the caller's struct.
	if settings != nil {
		log.WithField("settings", settings).Debugln("Found settings")
		*params = *settings
	}

	// Check the parameters for inconsistencies
	if params.Inline && (params.GroupByType || params.OutputDir != "") {
		return fmt.Errorf("the inline flag conflicts with the group-by-type and output-dir flags")
	}

	log.WithField("target", target).Traceln("Starting sort")
	log.WithField("params", params).Debugln("Using params for SortFiles")

	// Get files from target
	files, err := getFilesFromTarget(target)
	if err != nil {
		return fmt.Errorf("could not get files from target: %w", err)
	}

	// Sort the files
	sortedFiles, err := sortFiles(files)
	if err != nil {
		return fmt.Errorf("could not sort files: %w", err)
	}

	// Write the sorted files to the target directory if the inline flag is set
	if params.Inline {
		params.OutputDir, err = getDirectory(target)
		if err != nil {
			return fmt.Errorf("could not get directory for the target: %w", err)
		}
	}

	// Output the sorted files
	if params.OutputDir != "" {
		if err := writeFiles(sortedFiles); err != nil {
			return fmt.Errorf("could not write files: %w", err)
		}
	} else {
		for _, body := range sortedFiles {
			fmt.Println(string(body))
		}
	}

	return nil
}

// SetFileSystem sets the filesystem to use for the Sort command
func SetFileSystem(fs afero.Fs) {
	log.WithField("fs", fs).Traceln("Starting SetFileSystem")

	FS = fs
	AFS = &afero.Afero{Fs: FS}
}
