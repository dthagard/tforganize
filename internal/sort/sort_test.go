package sort

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

const (
	sortedDir   = "sorted"
	testDataDir = "testdata"
	unsortedDir = "unsorted"
)

func init() {
	SetFileSystem(afero.NewOsFs())
}

func sortCleanup() {
	// No-op
}

func TestSortFile(t *testing.T) {
	t.Cleanup(sortCleanup)

	/*********************************************************************/
	// Happy path test for sortFile() with a single file
	/*********************************************************************/

	t.Run("single file", func(t *testing.T) {
		path := filepath.Join(testDataDir, "single_file")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Happy path test for sortFile() with a multiple files
	/*********************************************************************/

	t.Run("multiple files", func(t *testing.T) {
		path := filepath.Join(testDataDir, "multiple_files")
		testSortFile(path, t)
	})

	/*********************************************************************/
	// Happy path test for sortFile() with a multiple files with headers
	/*********************************************************************/

	t.Run("multiple files with headers", func(t *testing.T) {
		path := filepath.Join(testDataDir, "multiple_files_with_headers")
		testSortFile(path, t)
	})
}

// testSortFile is a helper function for TestSortFile
func testSortFile(path string, t *testing.T) {
	// Get the files in the unsorted directory
	unsortedFiles, err := AFS.ReadDir(filepath.Join(path, unsortedDir))
	if err != nil {
		log.WithError(err).Errorln("could not read unsorted directory")
	}

	// Set the params
	if err := setParams(path); err != nil {
		log.WithError(err).Errorln("could not set params")
	}
	params.OutputDir = filepath.Join("./scratch", path)

	// Check each file if sorted correctly
	for _, file := range unsortedFiles {
		fileName := file.Name()

		// Read the sorted file
		sortedFilePath := filepath.Join(path, sortedDir, fileName)
		sortedBytes, err := AFS.ReadFile(sortedFilePath)
		if err != nil {
			log.WithError(err).Errorln("could not read file")
		}

		// Sort the unsorted file
		unsortedFilePath := filepath.Join(path, unsortedDir, fileName)
		results, err := sortFile(unsortedFilePath)
		if err != nil {
			log.WithError(err).Errorln("could not sort file")
		}

		// Compare the sorted file to the expected sorted file
		if !reflect.DeepEqual(results[fileName], sortedBytes) {
			t.Errorf("sortFile(%s) did not match the expected sorted file", fileName)
		}
	}
}

// setParams is a helper function for testSortFile
func setParams(path string) error {
	configFile := filepath.Join(path, ".tforganize.yaml")
	if ok, _ := AFS.Exists(configFile); ok {
		config, err := AFS.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("could not read config file: %s", configFile)
		}
		if err := yaml.Unmarshal(config, params); err != nil {
			return fmt.Errorf("could not unmarshal config file: %s", configFile)
		}
	}
	return nil
}
