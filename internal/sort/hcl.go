package sort

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
	log "github.com/sirupsen/logrus"
)

// hclParseFn is the function used to parse raw HCL bytes into an hcl.File.
// It is a package-level variable so tests can replace it with a stub that
// returns a non-hclsyntax body, exercising the type-assertion error path.
var hclParseFn = func(content []byte, filename string) (*hcl.File, hcl.Diagnostics) {
	return hclparse.NewParser().ParseHCL(content, filename)
}

// parseHclFile reads an HCL file from the given path and returns the body of the file
func parseHclFile(path string) (*hclsyntax.Body, error) {
	log.WithField("path", path).Traceln("Starting parseHclFile")

	// Read the HCL file content
	hclContent, err := AFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read HCL file: %w", err)
	}

	// Parse the HCL content
	file, diag := hclParseFn(hclContent, path)
	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diag.Error())
	}
	log.WithField("file", file).Debugln("Got back file from parser.ParseHCL")

	// Get the body of the HCL file
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected HCL body type %T: expected *hclsyntax.Body", file.Body)
	}

	return body, nil
}

// BlockListSorter implements the sort.Interface for []*hclsyntax.Block
type BlockListSorter []*hclsyntax.Block

// Len returns the length of the array
func (bs BlockListSorter) Len() int {
	return len(bs)
}

// Less compares two blocks based on their types and labels
func (bs BlockListSorter) Less(i, j int) bool {
	block1 := bs[i]
	block2 := bs[j]

	// First, compare the Type fields
	if block1.Type != block2.Type {
		return block1.Type < block2.Type
	}

	// If the Type is the same, compare the Labels array
	// based on the order of the strings
	minLen := len(block1.Labels)
	if len(block2.Labels) < minLen {
		minLen = len(block2.Labels)
	}

	for k := 0; k < minLen; k++ {
		if block1.Labels[k] != block2.Labels[k] {
			return block1.Labels[k] < block2.Labels[k]
		}
	}

	// If the common Labels are the same, the one with fewer Labels should come first
	return len(block1.Labels) < len(block2.Labels)
}

// Swap swaps two blocks in the array
func (bs BlockListSorter) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}

// isSortable returns true if the file is sortable.
func isSortable(file fs.FileInfo) bool {
	if filepath.Ext(file.Name()) != ".tf" {
		log.WithField("file.Name()", file.Name()).Debugln("File is not sortable")
		return false
	}
	log.WithField("file.Name()", file.Name()).Debugln("File is sortable")
	return true
}

// getNodeComment returns the comment lines for the node starting at the given line
func getNodeComment(lines []string, startLine int) []string {
	log.WithFields(log.Fields{"lines": lines, "startLine": startLine}).Traceln("Starting getNodeComment")

	// Initialize the return value
	var comment []string

	var buffer []string
	insertNewLine := false
	insideComment := false
	for i := startLine - 1; i >= 0; i-- {
		log.WithField("i", i).Debugln("Checking line for comment")

		// Check if the previous line is a comment
		if isStartOfComment(lines[i]) {
			// Preserve a single empty line between comments
			if insertNewLine {
				buffer = append(buffer, "")
				insertNewLine = false
			}
			buffer = append(buffer, lines[i])
			insideComment = false
		} else if isEndOfComment(lines[i]) {
			if insertNewLine {
				buffer = append(buffer, "")
				insertNewLine = false
			}
			buffer = append(buffer, lines[i])
			insideComment = true
		} else if isEmptyLine(lines[i]) {
			insertNewLine = true
		} else {
			if !insideComment {
				break
			}
			buffer = append(buffer, lines[i])
		}
	}

	if len(buffer) > 0 {
		// Reverse the comment lines since we captured them in the reverse order
		comment = reverseStringArray(buffer)

		// Remove the header if present
		if params.HasHeader {
			comment = removeHeader(comment)
		}

		// Remove the trailing empty lines
		comment = removeTrailingEmptyLines(comment)
	}

	return comment
}

// removeHeader removes the header from the comment lines
func removeHeader(lines []string) []string {
	log.WithField("lines", lines).Traceln("Starting removeHeader")

	comment := strings.Join(lines, "\n")
	comment = strings.Replace(comment, params.HeaderPattern, "", 1)
	result := strings.Split(comment, "\n")

	// Remove the leading empty lines
	result = removeLeadingEmptyLines(result)

	return result
}

// addHeader prefixes a comment header to a byte array
func addHeader(buffer []byte) []byte {
	log.Traceln("Starting addHeader")

	headerBytes := []byte(params.HeaderPattern)
	newlineBytes := []byte("\n")

	result := make([]byte, 0, len(headerBytes)+len(buffer)+len(newlineBytes))
	result = append(result, headerBytes...)
	result = append(result, newlineBytes...)
	result = append(result, buffer...)

	return result
}

// isStartOfComment checks if a line is the start of a comment
func isStartOfComment(line string) bool {
	log.WithField("line", line).Traceln("Starting isStartOfComment")

	if isEmptyLine(line) {
		return false
	}

	s := strings.TrimSpace(line)
	log.WithField("s", s).Debugln("Trimmed line")

	if s[:1] == "#" {
		log.Debugln("Line is start of a comment")
		return true
	}

	if len(s) >= 2 && (s[:2] == "//" || s[:2] == "/*") {
		log.Debugln("Line is start of a comment")
		return true
	}

	return false
}

// isEndOfComment checks if a line is the end of a comment
func isEndOfComment(line string) bool {
	log.WithField("line", line).Traceln("Starting isEndOfComment")

	if isEmptyLine(line) {
		return false
	}

	s := strings.TrimSpace(line)
	log.WithField("s", s).Debugln("Trimmed line")

	if len(s) >= 2 && s[len(s)-2:] == "*/" {
		log.Debugln("Line is end of a comment")
		return true
	}

	return false
}

// isEmptyLine checks if a line is empty
func isEmptyLine(line string) bool {
	log.WithField("line", line).Traceln("Starting isEmptyLine")

	s := strings.TrimSpace(line)
	log.WithField("s", s).Debugln("Trimmed line")

	return len(s) == 0
}

// removeLeadingEmptyLines truncates extra empty lines from the end of the comment
func removeTrailingEmptyLines(lines []string) []string {
	log.WithField("lines", lines).Traceln("Starting removeTrailingEmptyLines")

	result := lines
	for i := len(lines) - 1; i >= 0; i-- {
		if i > 0 && isEmptyLine(lines[i-1]) && isEmptyLine(lines[i]) {
			result = lines[:i]
		} else {
			break
		}
	}

	return result
}

// removeLeadingEmptyLines removes empty lines from the beginning of the comment
func removeLeadingEmptyLines(lines []string) []string {
	log.WithField("lines", lines).Traceln("Starting removeLeadingEmptyLines")

	result := lines
	for i := 0; i < len(lines); i++ {
		if isEmptyLine(lines[i]) {
			result = lines[i+1:]
		} else {
			break
		}
	}

	return result
}
