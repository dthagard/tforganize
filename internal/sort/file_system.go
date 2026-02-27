package sort

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// getFilesFromTarget returns a list of files to sort.
func (s *Sorter) getFilesFromTarget(target string) ([]string, error) {
	log.WithField("target", target).Traceln("Starting getFilesFromTarget")

	targetInfo, err := s.getPathInfo(target)
	if err != nil {
		return nil, err
	}

	if targetInfo.IsDir() {
		log.Debugln("target is a directory")
		files, err := s.getFilesInFolder(target)
		if err != nil {
			return nil, fmt.Errorf("could not get files in folder: %w", err)
		}
		return files, nil
	}

	// Single-file target: check excludes before returning.
	excluded, err := s.isExcluded(filepath.Dir(target), target)
	if err != nil {
		return nil, err
	}
	if excluded {
		// Return empty list — caller (run) handles gracefully (nothing to sort).
		return []string{}, nil
	}
	return []string{target}, nil
}

// getPathInfo returns the filesystem info for a given path.
// When running on a real filesystem, it uses Lstat to detect symlinks and
// refuses them to prevent overwriting symlink targets in --inline mode.
func (s *Sorter) getPathInfo(path string) (fs.FileInfo, error) {
	log.WithField("path", path).Traceln("Starting getPathInfo")

	// Use Lstat when available to detect symlinks.
	if lstater, ok := s.fs.(afero.Lstater); ok {
		info, lstatCalled, err := lstater.LstatIfPossible(path)
		if err != nil {
			return nil, fmt.Errorf("could not get target info: %w", err)
		}
		if lstatCalled && info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing to process symlink: %s", path)
		}
		return info, nil
	}

	info, err := s.fs.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("could not get target info: %w", err)
	}

	return info, nil
}

// getDirectory returns the directory for a given path.
func (s *Sorter) getDirectory(path string) (string, error) {
	log.WithField("path", path).Traceln("Starting getDirectory")

	info, err := s.getPathInfo(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return path, nil
	}

	return filepath.Dir(path), nil
}

// getFilesInFolder returns a list of files in a folder.
func (s *Sorter) getFilesInFolder(path string) ([]string, error) {
	log.WithField("path", path).Traceln("Starting getFilesInFolder")

	files, err := s.afs.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the directory: %w", err)
	}

	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		if !isSortable(file) {
			continue
		}
		filePath := filepath.Join(path, file.Name())
		excluded, err := s.isExcluded(path, filePath)
		if err != nil {
			return nil, err
		}
		if excluded {
			continue
		}
		fileNames = append(fileNames, filePath)
	}

	return fileNames, nil
}

// isExcluded reports whether the given absolute filePath should be excluded
// from processing, based on the exclude glob patterns in s.params.Excludes.
//
// Matching is performed against the path of filePath relative to targetDir,
// using forward-slash separators. This mirrors .gitignore semantics.
//
// Returns (false, nil) when no patterns are configured.
// Returns (false, error) when a pattern is syntactically invalid — callers
// must propagate this error; pattern validity should already be checked in
// run() before any file iteration begins.
func (s *Sorter) isExcluded(targetDir string, filePath string) (bool, error) {
	if len(s.params.Excludes) == 0 {
		return false, nil
	}

	rel, err := filepath.Rel(targetDir, filePath)
	if err != nil {
		// Unreachable in practice: targetDir and filePath are always both
		// absolute. Fall back to the basename.
		rel = filepath.Base(filePath)
	}
	// Normalise to forward slashes for cross-platform pattern matching.
	rel = filepath.ToSlash(rel)

	for _, pattern := range s.params.Excludes {
		matched, err := doublestar.Match(pattern, rel)
		if err != nil {
			return false, fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
		if matched {
			log.WithFields(log.Fields{
				"file":    filePath,
				"pattern": pattern,
			}).Debugln("file excluded by pattern")
			return true, nil
		}
	}
	return false, nil
}

// getLinesFromFile returns a list of lines from a file.
// Results are cached in s.linesCache for the duration of the current sort run.
func (s *Sorter) getLinesFromFile(filename string) ([]string, error) {
	log.WithField("filename", filename).Traceln("Starting getLinesFromFile")

	if lines, ok := s.linesCache[filename]; ok {
		log.WithField("filename", filename).Traceln("getLinesFromFile cache hit")
		return lines, nil
	}

	file, err := s.afs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), 1<<20) // 1 MB max line length

	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	s.linesCache[filename] = lines
	return lines, nil
}

// cacheLinesFromBytes splits raw content into lines and stores them in the
// lines cache under the given filename.
func (s *Sorter) cacheLinesFromBytes(content []byte, filename string) {
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	s.linesCache[filename] = lines
	if abs, err := filepath.Abs(filename); err == nil && abs != filename {
		s.linesCache[abs] = lines
	}
}

// getFileNameFromPath returns the filename from a path.
func getFileNameFromPath(path string) string {
	log.WithField("path", path).Traceln("Starting getFileNameFromPath")

	return filepath.Base(path)
}

// combineFiles combines a list of files into a single in-memory byte buffer.
func (s *Sorter) combineFiles(inputFilePaths []string) ([]byte, error) {
	log.WithField("inputFilePaths", inputFilePaths).Traceln("Starting combineFiles")

	var buffer []byte
	// Iterate over the input file paths
	for _, inputPath := range inputFilePaths {
		inputFileBytes, err := s.afs.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("could not read file: %w", err)
		}
		buffer = append(buffer, inputFileBytes...)
	}

	log.Debugln("Files combined successfully.")
	return buffer, nil
}

// writeFiles writes all of the processed files to the filesystem.
func (s *Sorter) writeFiles(fileBytes map[string][]byte) error {
	log.WithField("fileBytes", fileBytes).Traceln("Starting writeFiles")

	log.WithField("OutputDir", s.params.OutputDir).Debugln("Creating output directory...")
	if err := s.afs.MkdirAll(s.params.OutputDir, 0755); err != nil {
		return fmt.Errorf("could not create the output directory: %w", err)
	}

	// write bytes to the file
	for k, v := range fileBytes {
		fileName := filepath.Join(s.params.OutputDir, getFileNameFromPath(k))
		log.WithField("fileName", fileName).Debugln("Writing to file...")
		if err := s.writeFile(fileName, v); err != nil {
			return fmt.Errorf("could not write to the file: %w", err)
		}
	}

	return nil
}

// writeFile writes a byte array to a file, preserving the original file's
// permissions when it exists. Falls back to 0644 for new files.
func (s *Sorter) writeFile(filename string, fileBytes []byte) error {
	log.WithFields(log.Fields{"filename": filename, "fileBytes": fileBytes}).Traceln("Starting writeFile")

	// Preserve original file permissions when overwriting.
	perm := fs.FileMode(0644)
	if info, err := s.fs.Stat(filename); err == nil {
		perm = info.Mode().Perm()
	}

	// create file
	log.WithField("filename", filename).Debugln("Creating file...")
	f, err := s.fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
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
