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
	GroupByType bool `yaml:"group-by-type"`
	// If the has-header flag is set, the input files have a header.
	HasHeader bool `yaml:"has-header"`
	// If the header-pattern flag is set, the header pattern will be used to find the header in the input files.
	HeaderPattern string `yaml:"header-pattern"`
	// If the keep-header flag is set, the header matched in the header pattern will be persisted in the output files.
	KeepHeader bool `yaml:"keep-header"`
	// If the output directory is set, the sorted files will be written to the output directory.
	// Otherwise, the sorted files will be printed to stdout.
	OutputDir string `yaml:"output-dir"`
	// If the remove-comments flag is set, the comments will be removed from the files.
	// Otherwise, the comments will be preserved.
	RemoveComments bool `yaml:"remove-comments"`
}

// Sort sorts a Terraform file or folder.
//
// If the target is a folder, all files in the folder will be sorted.
// If the target is a file, only that file will be sorted.
func Sort(target string, settings *Params) {
	log.WithField("target", target).Traceln("Starting sort")

	if settings != nil {
		log.WithField("settings", settings).Debugln("Found settings")
		params = settings
	}
	log.WithField("params", params).Debugln("Using params for SortFiles")

	// Get files from target
	files, err := getFilesFromTarget(target)
	if err != nil {
		log.WithError(err).Errorln("could not get files from target")
	}

	// Sort the files
	sortedFiles, err := sortFiles(files)
	if err != nil {
		log.WithError(err).Errorln("could not sort files")
	}

	// Output the sorted files
	if params.OutputDir != "" {
		err := writeFiles(sortedFiles)
		if err != nil {
			log.WithError(err).Errorln("could not write files")
		}
	} else {
		for _, body := range sortedFiles {
			fmt.Println(string(body))
		}
	}
}

// SetFileSystem sets the filesystem to use for the Sort command
func SetFileSystem(fs afero.Fs) {
	log.WithField("fs", fs).Traceln("Starting SetFileSystem")

	FS = fs
	AFS = &afero.Afero{Fs: FS}
}
