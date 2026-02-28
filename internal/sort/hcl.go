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

// parseHclFile reads an HCL file from the given path and returns the body of the file.
func (s *Sorter) parseHclFile(path string) (*hclsyntax.Body, error) {
	log.WithField("path", path).Traceln("Starting parseHclFile")

	// Read the HCL file content
	hclContent, err := s.afs.ReadFile(path)
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

// parseHclBytes parses raw HCL content and returns the body.
func (s *Sorter) parseHclBytes(content []byte, filename string) (*hclsyntax.Body, error) {
	log.WithField("filename", filename).Traceln("Starting parseHclBytes")

	// Parse the HCL content
	file, diag := hclParseFn(content, filename)
	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diag.Error())
	}

	body := file.Body.(*hclsyntax.Body)
	return body, nil
}

// BlockListSorter implements the sort.Interface for []*hclsyntax.Block.
// When sortByType is true, blocks are ordered by logical type priority
// (see blockTypePriority); otherwise they are ordered alphabetically by type.
type BlockListSorter struct {
	blocks     []*hclsyntax.Block
	sortByType bool
}

// Len returns the length of the array.
func (bs BlockListSorter) Len() int {
	return len(bs.blocks)
}

// Less compares two blocks based on their types and labels.
func (bs BlockListSorter) Less(i, j int) bool {
	block1 := bs.blocks[i]
	block2 := bs.blocks[j]

	// First, compare the Type fields
	if block1.Type != block2.Type {
		if bs.sortByType {
			return getBlockTypePriority(block1.Type) < getBlockTypePriority(block2.Type)
		}
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

// Swap swaps two blocks in the array.
func (bs BlockListSorter) Swap(i, j int) {
	bs.blocks[i], bs.blocks[j] = bs.blocks[j], bs.blocks[i]
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

// getNodeComment returns the comment lines for the node starting at the given line.
// filename is used to look up the pre-detected header for correct removal.
func (s *Sorter) getNodeComment(lines []string, startLine int, filename string) []string {
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
		if s.params.HasHeader {
			comment = s.removeHeader(comment, filename)
		}

		// Remove the trailing empty lines
		comment = removeTrailingEmptyLines(comment)
	}

	return comment
}

// detectFileHeader scans the raw lines of a file for a leading comment header
// and stores the detected text in s.detectedHeaders keyed by filename.
//
// The detection logic:
//  1. If HeaderEndPattern is set, everything from the first line matching
//     HeaderPattern (or the first comment line) through the first line
//     matching HeaderEndPattern is treated as the header.
//  2. Otherwise, the leading block comment (/* … */) or consecutive line
//     comments (# / //) are captured as the header.
//  3. The captured text must contain HeaderPattern (when non-empty) to be
//     accepted as a header.
func (s *Sorter) detectFileHeader(filename string) error {
	lines, err := s.getLinesFromFile(filename)
	if err != nil {
		return err
	}

	header := s.findHeaderInLines(lines)
	if header != "" {
		s.detectedHeadersMu.Lock()
		s.detectedHeaders[filename] = header
		s.detectedHeadersMu.Unlock()
	}
	return nil
}

// findHeaderInLines extracts a header comment block from the top of a file's
// lines. It returns the joined header text or "" if no header is found.
func (s *Sorter) findHeaderInLines(lines []string) string {
	var headerLines []string
	insideBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip leading empty lines before the header starts
		if len(headerLines) == 0 && trimmed == "" {
			continue
		}

		if insideBlockComment {
			headerLines = append(headerLines, line)
			// Check for end of block comment
			if s.params.HeaderEndPattern != "" {
				if strings.Contains(trimmed, s.params.HeaderEndPattern) {
					break
				}
			} else if strings.HasSuffix(trimmed, "*/") {
				break
			}
			continue
		}

		// Check for start of block comment
		if strings.HasPrefix(trimmed, "/*") {
			insideBlockComment = true
			headerLines = append(headerLines, line)
			// Single-line block comment (e.g. /* header */)
			if s.params.HeaderEndPattern != "" {
				if strings.Contains(trimmed, s.params.HeaderEndPattern) {
					break
				}
			} else if strings.HasSuffix(trimmed, "*/") {
				break
			}
			continue
		}

		// Check for line comments (# or //)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			headerLines = append(headerLines, line)
			if s.params.HeaderEndPattern != "" && strings.Contains(trimmed, s.params.HeaderEndPattern) {
				break
			}
			continue
		}

		// Non-comment, non-blank line — header region ends
		break
	}

	if len(headerLines) == 0 {
		return ""
	}

	header := strings.Join(headerLines, "\n")

	// If HeaderPattern is set, verify the header contains it.
	// Trim trailing newlines from the pattern to handle YAML literal blocks.
	pattern := strings.TrimRight(s.params.HeaderPattern, "\n")
	if pattern != "" && !strings.Contains(header, pattern) {
		return "" // Pattern not found in detected header
	}

	return header
}

// removeHeader removes the header from the comment lines.
//
// When a pre-detected header is available for the file, the function performs a
// line-by-line prefix match and strips exactly those lines. This correctly
// handles partial header-pattern values (e.g. just "Copyright" or "/**").
//
// When no pre-detected header exists (e.g. SortBytes with an exact pattern),
// it falls back to the legacy strings.Replace behaviour.
func (s *Sorter) removeHeader(lines []string, filename string) []string {
	log.WithField("lines", lines).Traceln("Starting removeHeader")

	// Look up the pre-detected header for this file.
	s.detectedHeadersMu.Lock()
	detectedHeader := s.detectedHeaders[filename]
	s.detectedHeadersMu.Unlock()

	if detectedHeader != "" {
		headerLines := strings.Split(detectedHeader, "\n")

		// Check if the comment lines start with the detected header.
		if len(lines) >= len(headerLines) {
			match := true
			for i, hl := range headerLines {
				if lines[i] != hl {
					match = false
					break
				}
			}
			if match {
				remaining := lines[len(headerLines):]
				stripped := removeLeadingEmptyLines(remaining)
				if !s.params.KeepHeader && len(stripped) > 0 && len(remaining) > len(stripped) {
					stripped = append([]string{""}, stripped...)
				}
				return stripped
			}
		}
	}

	// Fallback: legacy exact-substring removal (handles SortBytes and
	// cases where the full header text is provided as the pattern).
	comment := strings.Join(lines, "\n")
	comment = strings.Replace(comment, s.params.HeaderPattern, "", 1)
	result := strings.Split(comment, "\n")

	// Remove the leading empty lines
	stripped := removeLeadingEmptyLines(result)
	if !s.params.KeepHeader && len(stripped) > 0 && len(result)-len(stripped) >= 2 {
		stripped = append([]string{""}, stripped...)
	}

	return stripped
}

// addHeader prefixes a comment header to a byte array.
//
// When a pre-detected header is available for the file, it is used instead of
// the raw HeaderPattern. This ensures the complete header is re-added even when
// HeaderPattern is only a partial match string.
func (s *Sorter) addHeader(buffer []byte, filename string) []byte {
	log.Traceln("Starting addHeader")

	// Use the pre-detected header if available; fall back to HeaderPattern.
	s.detectedHeadersMu.Lock()
	header := s.detectedHeaders[filename]
	s.detectedHeadersMu.Unlock()

	if header != "" {
		headerBytes := []byte(header)
		newlineBytes := []byte("\n\n")

		result := make([]byte, 0, len(headerBytes)+len(buffer)+len(newlineBytes))
		result = append(result, headerBytes...)
		result = append(result, newlineBytes...)
		result = append(result, buffer...)
		return result
	}

	// Fallback: use HeaderPattern directly (legacy behaviour).
	headerBytes := []byte(s.params.HeaderPattern)
	newlineBytes := []byte("\n")

	result := make([]byte, 0, len(headerBytes)+len(buffer)+len(newlineBytes))
	result = append(result, headerBytes...)
	result = append(result, newlineBytes...)
	result = append(result, buffer...)

	return result
}

// isStartOfComment checks if a line is the start of a comment.
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

// isEndOfComment checks if a line is the end of a comment.
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

// isEmptyLine checks if a line is empty.
func isEmptyLine(line string) bool {
	log.WithField("line", line).Traceln("Starting isEmptyLine")

	s := strings.TrimSpace(line)
	log.WithField("s", s).Debugln("Trimmed line")

	return len(s) == 0
}

// removeTrailingEmptyLines truncates extra empty lines from the end of a slice.
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

// removeLeadingEmptyLines removes empty lines from the beginning of a slice.
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
