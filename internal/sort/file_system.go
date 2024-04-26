package sort

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var (
	AFS *afero.Afero
	FS  afero.Fs
)

func initFileSystem() {
	log.Traceln("Starting initFileSystem")
	FS = afero.NewOsFs()
	AFS = &afero.Afero{Fs: FS}
}

// getFilesFromTarget returns a list of files to sort.
func getFilesFromTarget(target string) ([]string, error) {
	log.WithField("target", target).Traceln("Starting getFilesFromTarget")

	targetInfo, err := getPathInfo(target)
	if err != nil {
		return nil, err
	}

	files := []string{target}
	if targetInfo.IsDir() {
		log.Debugln("target is a directory")
		files, err = getFilesInFolder(target)
		if err != nil {
			return nil, fmt.Errorf("could not get files in folder: %w", err)
		}
	}

	return files, nil
}

// Get the filesystem info for a given target
func getPathInfo(path string) (fs.FileInfo, error) {
	log.WithField("path", path).Traceln("Starting getPathInfo")

	info, err := AFS.Fs.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("could not get target info: %w", err)
	}

	return info, nil
}

// getFilesInFolder returns a list of files in a folder.
func getFilesInFolder(path string) ([]string, error) {
	log.WithField("path", path).Traceln("Starting getFilesInFolder")

	files, err := AFS.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the directory: %w", err)
	}

	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		if isSortable(file) {
			filePath := filepath.Join(path, file.Name())
			fileNames = append(fileNames, filePath)
		}
	}

	return fileNames, nil
}

// getLinesFromFile returns a list of lines from a file.
func getLinesFromFile(filename string) ([]string, error) {
	log.WithField("filename", filename).Traceln("Starting getLinesFromFile")

	file, err := AFS.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// getFileNameFromPath returns the filename from a path.
func getFileNameFromPath(path string) string {
	log.WithField("path", path).Traceln("Starting getFileNameFromPath")

	return filepath.Base(path)
}

// combineFiles combines a list of files into a single file.
func combineFiles(inputFilePaths []string) (string, error) {
	log.WithField("inputFilePaths", inputFilePaths).Traceln("Starting combineFiles")

	// Create temporary file path
	tempDir := AFS.GetTempDir("tforganize/")

	// Create the output file
	outputFile, err := AFS.TempFile(tempDir, fmt.Sprintf("%v.tf", os.Getuid()))
	if err != nil {
		return "", fmt.Errorf("could not create temporary file: %w", err)
	}
	defer outputFile.Close()

	var buffer []byte
	// Iterate over the input file paths
	for _, inputPath := range inputFilePaths {
		inputFileBytes, err := AFS.ReadFile(inputPath)
		if err != nil {
			return "", fmt.Errorf("could not read file: %w", err)
		}
		buffer = append(buffer, inputFileBytes...)
	}

	if _, err = outputFile.Write(buffer); err != nil {
		return "", fmt.Errorf("could not write temporary file: %w", err)
	}

	log.Debugln("Files combined successfully.")
	return outputFile.Name(), nil
}

// writeFiles writes all of the processed files to the filesystem
func writeFiles(fileBytes map[string][]byte) error {
	log.WithField("fileBytes", fileBytes).Traceln("Starting writeFiles")

	log.WithField("OutputDir", params.OutputDir).Debugln("Creating output directory...")
	if err := AFS.MkdirAll(params.OutputDir, 0755); err != nil {
		return fmt.Errorf("could not create the output directory: %w", err)
	}

	// write bytes to the file
	for k, v := range fileBytes {
		fileName := filepath.Join(params.OutputDir, getFileNameFromPath(k))
		log.WithField("fileName", fileName).Debugln("Writing to file...")
		if err := writeFile(fileName, v); err != nil {
			return fmt.Errorf("could not write to the file: %w", err)
		}
	}

	return nil
}

// writeFile writes a byte array to a file
func writeFile(filename string, fileBytes []byte) error {
	log.WithFields(log.Fields{"filename": filename, "fileBytes": fileBytes}).Traceln("Starting writeFile")

	// create file
	log.WithField("filename", filename).Debugln("Creating file...")
	f, err := AFS.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create the file: %w", err)
	}
	// remember to close the file
	defer f.Close()

	log.Debugln("Writing to file...")
	if _, err = f.Write(fileBytes); err != nil {
		return fmt.Errorf("could not write to the file: %w", err)
	}
	log.Debugln("Done writing to file.")

	return nil
}
