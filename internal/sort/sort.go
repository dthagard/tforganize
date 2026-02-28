package sort

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
	hclwrite "github.com/hashicorp/hcl/v2/hclwrite"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	blockTypeLabel   = "block.Type"
	blockLabelsLabel = "block.Labels"
)

// sortFiles sorts a list of files.
func (s *Sorter) sortFiles(files []string) (map[string][]byte, error) {
	log.WithField("files", files).Traceln("Starting sortFiles")

	if s.params.GroupByType {
		log.Debugln("Creating combined file...")
		combinedBytes, err := s.combineFiles(files)
		if err != nil {
			return nil, fmt.Errorf("could not combine files: %w", err)
		}
		return s.sortFileBytes(combinedBytes, "combined.tf")
	}

	// Process files in parallel when there are multiple files.
	type fileResult struct {
		sorted map[string][]byte
	}
	results := make([]fileResult, len(files))

	g := new(errgroup.Group)
	for i, f := range files {
		g.Go(func() error {
			sortedFileBytes, err := s.sortFile(f)
			if err != nil {
				return fmt.Errorf("could not sort file %s: %w", f, err)
			}
			results[i] = fileResult{sorted: sortedFileBytes}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Merge results sequentially to preserve deterministic output order.
	output := map[string][]byte{}
	for _, r := range results {
		for k, v := range r.sorted {
			output[k] = append(output[k], v...)
		}
	}

	return output, nil
}

// sortFile sorts a single file into one or more files.
func (s *Sorter) sortFile(path string) (map[string][]byte, error) {
	log.WithField("path", path).Traceln("Starting sortFile")

	body, err := s.parseHclFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not get body from file: %w", err)
	}

	// Detect the file header before sorting so removeHeader/addHeader can
	// operate on the complete header text regardless of HeaderPattern value.
	if s.params.HasHeader {
		if err := s.detectFileHeader(path); err != nil {
			return nil, fmt.Errorf("could not detect file header: %w", err)
		}
	}

	return s.sortBody(body, path)
}

// sortFileBytes sorts in-memory HCL content into one or more output files.
func (s *Sorter) sortFileBytes(content []byte, filename string) (map[string][]byte, error) {
	log.WithField("filename", filename).Traceln("Starting sortFileBytes")

	s.cacheLinesFromBytes(content, filename)

	// Detect the file header before sorting.
	if s.params.HasHeader {
		if err := s.detectFileHeader(filename); err != nil {
			return nil, fmt.Errorf("could not detect file header: %w", err)
		}
	}

	body, err := s.parseHclBytes(content, filename)
	if err != nil {
		return nil, fmt.Errorf("could not get body from bytes: %w", err)
	}

	return s.sortBody(body, filename)
}

// sortBody sorts an HCL body's blocks and returns the formatted output.
// inputFilename is the original file path, used to look up the pre-detected
// header for correct re-addition when --keep-header is set.
func (s *Sorter) sortBody(body *hclsyntax.Body, inputFilename string) (map[string][]byte, error) {
	log.Debugln("Sorting blocks...")
	sortedFileBytes, err := s.sortBlocks(body.Blocks)
	if err != nil {
		return nil, fmt.Errorf("could not sort blocks: %w", err)
	}

	output := map[string][]byte{}
	for k, v := range sortedFileBytes {
		buffer := v
		if s.params.KeepHeader {
			log.Debugln("Adding header...")
			buffer = s.addHeader(buffer, inputFilename)
		}
		formatted := hclwrite.Format(buffer)

		// Validate the formatted output is still valid HCL.
		if _, diag := hclParseFn(formatted, k); diag.HasErrors() {
			return nil, fmt.Errorf("sorted output for %s is not valid HCL: %s", k, diag.Error())
		}

		output[k] = formatted
	}

	return output, nil
}

// sortBlocks sorts a list of blocks and returns the sorted blocks as a byte array organized by file.
func (s *Sorter) sortBlocks(blocks hclsyntax.Blocks) (map[string][]byte, error) {
	log.WithField("blocks", blocks).Traceln("Starting sortBlocks")

	// Initialize the output
	output := map[string][]byte{}

	sort.Stable(BlockListSorter{
		blocks:     blocks,
		sortByType: !s.params.NoSortByType,
	})
	log.WithField("blocks", blocks).Debugln("Got back sorted blocks from BlockListSorter")

	// Iterate through each block and order its attributes and child blocks
	for _, block := range blocks {
		log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels, "block.Body": block.Body}).Debugln("Starting block iteration")

		// Sort the block
		blockBytes, err := s.getSortedBlockBytes(block)
		if err != nil {
			return nil, fmt.Errorf("could not sort block: %w", err)
		}

		outputKey := getFileNameFromPath(block.TypeRange.Filename)
		if s.params.GroupByType {
			outputKey = defaultFileGroup
			if v, ok := fileGroups[block.Type]; ok {
				outputKey = v
			}
		}

		output[outputKey] = addNewLineIfBufferExists(output[outputKey])
		output[outputKey] = append(output[outputKey], blockBytes...)
	}

	return output, nil
}

// getSortedBlockBytes recursively sorts a block based on its attributes and child blocks.
func (s *Sorter) getSortedBlockBytes(block *hclsyntax.Block) ([]byte, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getSortedBlockBytes")

	// Sort the block keys
	keys := getSortedBlockKeys(block)

	// Write the block opening
	results, err := s.getBlockOpeningBytes(block)
	if err != nil {
		return nil, fmt.Errorf("could not get block opening bytes: %w", err)
	}

	// Create buffer for the block body
	var buffer []byte

	// Write the block attributes and child blocks
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Debugln("Looping keys for getBlockBodyBytes")
	for i := 0; i < 3; i++ {
		log.WithFields(log.Fields{"i": i, "keys[i]": keys[i]}).Debugln("Found keys")
		if len(keys[i]) > 0 {
			log.WithFields(log.Fields{"i": i, "keys[i]": keys[i]}).Debugln("Using keys")
			buffer = addNewLineIfBufferExists(buffer)
			blockBytes, err := s.getBlockBodyBytes(block, keys[i])
			if err != nil {
				return nil, fmt.Errorf("could not append label to output: %w", err)
			}
			buffer = append(buffer, blockBytes...)
		}
	}

	results = append(results, buffer...)

	// Write the block closing
	results = append(results, getBlockClosingBytes(block)...)

	return results, nil
}

// getSortedBlockKeys returns a sorted map of the block attributes and child blocks
// grouped into three separate categories:
// 0. Pre-Meta Arguments
// 1. Arguments
// 2. Post-Meta Arguments
// This is done to ensure that the arguments are sorted in the correct order.
// See https://www.terraform.io/docs/configuration/syntax.html
func getSortedBlockKeys(block *hclsyntax.Block) map[int][]string {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getSortedBlockKeys")

	// Initialize the return map
	keys := make(map[int][]string)
	for i := 0; i < 3; i++ {
		keys[i] = []string{}
	}

	// Get the meta arguments for the block type
	metaArgs := getMetaArguments(block)

	// Categorize the body attributes
	for k := range block.Body.Attributes {
		if stringExists(metaArgs[0], k) {
			keys[0] = append(keys[0], k)
			log.WithField("k", k).Debugln("Found pre-meta attribute")
		} else if stringExists(metaArgs[1], k) {
			keys[2] = append(keys[2], k)
			log.WithField("k", k).Debugln("Found post-meta attribute")
		} else {
			keys[1] = append(keys[1], k)
			log.WithField("k", k).Debugln("Found normal attribute")
		}
	}

	// Categories the body blocks
	for _, b := range block.Body.Blocks {
		if stringExists(metaArgs[0], b.Type) {
			keys[0] = append(keys[0], formatBlockKey(b))
			log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Debugln("Found pre-meta block")
		} else if stringExists(metaArgs[1], b.Type) {
			keys[2] = append(keys[2], formatBlockKey(b))
			log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Debugln("Found post-meta block")
		} else {
			keys[1] = append(keys[1], formatBlockKey(b))
			log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Debugln("Found normal block")
		}
	}

	// Sort the pre-meta blocks and attributes by the order of the meta arguments
	if len(keys[0]) > 0 {
		sort.SliceStable(keys[0], func(i, j int) bool {
			key1 := keys[0][i]
			key2 := keys[0][j]

			for _, arg := range metaArgs[0] {
				if arg == key1 {
					return true
				} else if arg == key2 {
					return false
				}
			}
			return false
		})
	}

	// Sort the post and non-meta blocks and attributes alphabetically
	for i := 1; i < 3; i++ {
		sort.Strings(keys[i])
	}

	log.WithField("keys", keys).Debugln("Returning sorted keys")
	return keys
}

// formatBlockKey returns a formatted string of a block's type and labels.
func formatBlockKey(block *hclsyntax.Block) string {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting formatBlockKey")

	if len(block.Labels) > 0 {
		return block.Type + " " + strings.Join(block.Labels, " ")
	}
	return block.Type
}

// getBlockOpeningBytes returns the opening byte array of a block.
func (s *Sorter) getBlockOpeningBytes(block *hclsyntax.Block) ([]byte, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getBlockOpeningBytes")

	// Initialize the output
	var output []byte

	if !s.params.RemoveComments {
		log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Debugln("Getting node comment")
		lines, err := s.getLinesFromFile(block.TypeRange.Filename)
		if err != nil {
			return nil, fmt.Errorf("could not get lines from file: %w", err)
		}

		startLine := block.TypeRange.Start.Line - 1 // Subtract 1 to account for 0-indexing of string arrays vs HCL line numbers
		nodeComment := s.getNodeComment(lines, startLine, block.TypeRange.Filename)

		if len(nodeComment) > 0 {
			output = append(output, []byte(fmt.Sprintf("%s\n", strings.Join(nodeComment, "\n")))...)
		}
	}

	// Append the block type to the output
	output = append(output, []byte(block.Type)...)

	// Append the block labels
	for _, label := range block.Labels {
		output = append(output, []byte(fmt.Sprintf("  \"%s\"", label))...)
	}

	// Append the block opening
	output = append(output, []byte(" {\n")...)

	return output, nil
}

// getBlockBodyBytes returns the byte array of all the attributes and child blocks of a block.
func (s *Sorter) getBlockBodyBytes(block *hclsyntax.Block, keys []string) ([]byte, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getBlockBodyBytes")

	var output []byte
	var buffer []byte

	// Get the path of the file
	path, err := getPathFromBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not get path from block: %w", err)
	}

	// Pre-group child blocks by key to correctly handle multiple blocks of the
	// same type (e.g. multiple "statement" or "ingress" blocks). A plain linear
	// search always matches the first block, causing duplicates and data loss.
	childBlocksByKey := make(map[string][]*hclsyntax.Block)
	for _, childBlock := range block.Body.Blocks {
		childBlockKey := formatBlockKey(childBlock)
		childBlocksByKey[childBlockKey] = append(childBlocksByKey[childBlockKey], childBlock)
	}
	childBlockIndex := make(map[string]int)

	// Loop over the keys
	for i, key := range keys {
		// Write the block attributes
		if attribute, ok := block.Body.Attributes[key]; ok {
			log.WithField("attribute.Name", attribute.Name).Debugln("Found key in block attributes")
			b, err := s.getAttributeBytes(attribute, path)
			if err != nil {
				return nil, fmt.Errorf("could not write attribute: %w", err)
			}
			buffer = append(buffer, b...)
			continue
		}

		// Write the block child blocks — use the pre-grouped map so each
		// occurrence of a duplicate key maps to the next unused block.
		if childBlocks, ok := childBlocksByKey[key]; ok {
			idx := childBlockIndex[key]
			if idx < len(childBlocks) {
				childBlock := childBlocks[idx]
				childBlockIndex[key]++

				log.WithField("childBlock", childBlock).Debugln("Found child block in blocks")
				buffer = addNewLineIfBufferExists(buffer)
				b, err := s.getSortedBlockBytes(childBlock)
				if err != nil {
					return nil, fmt.Errorf("could not sort block: %w", err)
				}
				buffer = append(buffer, b...)

				// If there are more keys to parse, then we add a newline after the block
				if i < len(keys)-1 {
					buffer = append(buffer, []byte("\n")...)
				}

				// Append the buffer to the output and clear the buffer
				output = append(output, buffer...)
				buffer = []byte{}
			}
		}
	}

	// Flush the remaining buffer to the output
	if len(buffer) > 0 {
		output = append(output, buffer...)
	}

	return output, nil
}

// addNewLineIfBufferExists adds a newline to the buffer if the buffer is not empty.
func addNewLineIfBufferExists(buffer []byte) []byte {
	log.WithField("buffer", buffer).Traceln("Starting addNewLineIfBufferExists")

	if len(buffer) > 0 {
		buffer = append(buffer, []byte("\n")...)
	}
	return buffer
}

// getAttributeBytes returns the byte array of an attribute.
func (s *Sorter) getAttributeBytes(attribute *hclsyntax.Attribute, path string) ([]byte, error) {
	log.WithField("attribute.Name", attribute.Name).Traceln("Starting getAttributeBytes")

	content, err := s.readNodeFromFile(
		path,
		attribute.Range().Start.Line,
		attribute.Range().Start.Column,
		attribute.Range().End.Line,
		attribute.Range().End.Column,
	)
	if err != nil {
		return nil, fmt.Errorf("could not read file contents: %w", err)
	}

	str := strings.Join(content, "\n")
	b := []byte(fmt.Sprintf("%s\n", str))

	return b, nil
}

// getBlockClosingBytes returns the closing byte array of a block.
func getBlockClosingBytes(block *hclsyntax.Block) []byte {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getBlockClosingBytes")

	return []byte("}\n")
}

// getPathFromBlock returns the file path of the block.
func getPathFromBlock(block *hclsyntax.Block) (string, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getPathFromBlock")

	path, err := filepath.Abs(block.TypeRange.Filename)
	if err != nil {
		return "", fmt.Errorf("could not get absolute path from block: %w", err)
	}
	return path, nil
}

// readNodeFromFile reads a node from a file and returns the contents as a slice of strings.
//
// The startLine and startCol are inclusive.
// The endLine and endCol are exclusive.
func (s *Sorter) readNodeFromFile(filename string, startLine, startCol, endLine, endCol int) ([]string, error) {
	log.WithFields(log.Fields{"filename": filename, "startLine": startLine, "startCol": startCol, "endLine": endLine, "endCol": endCol}).Traceln("Starting readNodeFromFile")

	// HCL lines are 1-indexed, so we need to subtract 1 from the start and end lines
	startLine--
	endLine--

	lines, err := s.getLinesFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not get lines from file: %w", err)
	}

	var output []string

	if !s.params.RemoveComments {
		nodeComment := s.getNodeComment(lines, startLine, filename)

		if len(nodeComment) > 0 {
			output = append(output, nodeComment...)
		}
	}

	for i := startLine; i <= endLine; i++ {
		// Grab the current line
		line := lines[i]
		log.WithField("line", line).Debugln("Current line")

		// Grab the correct slice of the line
		lineSlice := s.getLineSlice(line, startLine, endLine, i, startCol, endCol)

		// Append the line slice to the buffer
		if len(lineSlice) > 0 {
			output = append(output, lineSlice)
		}
	}

	log.WithField("output", output).Debugln("Returning output")
	return output, nil
}

// getLineSlice returns the correct slice of the line based on the start and end columns.
func (s *Sorter) getLineSlice(line string, startLine, endLine, currentLine, startCol, endCol int) string {
	log.WithFields(log.Fields{"line": line, "startLine": startLine, "endLine": endLine, "currentLine": currentLine, "startCol": startCol, "endCol": endCol}).Traceln("Starting getLineSlice")

	// Check if it is the starting line
	if currentLine == startLine {
		log.Debugln("Current line is starting line.")
		// Truncate the line to the starting column
		line = line[startCol-1:]
	}

	if s.params.RemoveComments {
		if isStartOfComment(line) {
			log.Debugln("Removing comment line.")
			line = ""
		} else if currentLine == endLine { // Using the node endCol will truncate comments
			log.Debugln("Truncating line to end column.")
			if startLine == endLine {
				// Truncate the line from the ending column
				line = line[:endCol-startCol]
			} else {
				// Truncate the line from the ending column
				line = line[:endCol-1]
			}
		}
	}

	return line
}
