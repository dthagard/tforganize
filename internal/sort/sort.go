package sort

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
	hclwrite "github.com/hashicorp/hcl/v2/hclwrite"
	log "github.com/sirupsen/logrus"
)

const (
	blockTypeLabel   = "block.Type"
	blockLabelsLabel = "block.Labels"
)

// sortFiles sorts a list of files.
func sortFiles(files []string) (map[string][]byte, error) {
	log.WithField("files", files).Traceln("Starting sortFiles")

	// If we are grouping by type, we need to create a combined file
	// that contains all of the blocks from all of the files.
	filesToSort := files
	if params.GroupByType {
		log.Debugln("Creating combined file...")
		combinedFile, err := combineFiles(files)
		if err != nil {
			return nil, fmt.Errorf("could not combine files: %w", err)
		}
		// Reset filesToSort to only contain combinedFile.
		filesToSort = []string{combinedFile}
	}

	output := map[string][]byte{}
	for _, f := range filesToSort {
		sortedFileBytes, err := sortFile(f)
		if err != nil {
			return nil, fmt.Errorf("could not sort file: %w", err)
		}

		for k, v := range sortedFileBytes {
			output[k] = append(output[k], v...)
		}
	}

	return output, nil
}

// sortFile sorts a single file into one or more files.
func sortFile(path string) (map[string][]byte, error) {
	log.WithField("path", path).Traceln("Starting sortFile")

	body, err := parseHclFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not get body from file: %w", err)
	}

	log.Debugln("Sorting blocks...")
	sortedFileBytes, err := sortBlocks(body.Blocks)
	if err != nil {
		return nil, fmt.Errorf("could not sort blocks: %w", err)
	}

	output := map[string][]byte{}
	log.Debugln("Formatting blocks...")
	var buffer []byte
	for k, v := range sortedFileBytes {
		buffer = v
		// If we are keeping the header, add it back
		if params.KeepHeader {
			log.Debugln("Adding header...")
			buffer = addHeader(buffer)
		}
		output[k] = hclwrite.Format(buffer)
	}

	return output, nil
}

// sortBlocks sorts a list of blocks and returns the sorted blocks as a byte array organized by file.
func sortBlocks(blocks hclsyntax.Blocks) (map[string][]byte, error) {
	log.WithField("blocks", blocks).Traceln("Starting sortBlocks")

	// Initialize the output
	output := map[string][]byte{}

	sort.Sort(BlockListSorter(blocks))
	log.WithField("blocks", blocks).Debugln("Got back sorted blocks from BlockListSorter")

	// Iterate through each block and order its attributes and child blocks
	for _, block := range blocks {
		log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels, "block.Body": block.Body}).Debugln("Starting block iteration")

		// Sort the block
		blockBytes, err := getSortedBlockBytes(block)
		if err != nil {
			return nil, fmt.Errorf("could not sort block: %w", err)
		}

		outputKey := getFileNameFromPath(block.TypeRange.Filename)
		if params.GroupByType {
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

// getSortedBlockBytes recursively sorts a block based on its attributes and child blocks
func getSortedBlockBytes(block *hclsyntax.Block) ([]byte, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getSortedBlockBytes")

	// Sort the block keys
	keys := getSortedBlockKeys(block)

	// Write the block opening
	results, err := getBlockOpeningBytes(block)
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
			blockBytes, err := getBlockBodyBytes(block, keys[i])
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
			return true
		})
	}

	// Sort the post and non-meta blocks and attributes alphabetically
	for i := 1; i < 3; i++ {
		sort.Strings(keys[i])
	}

	log.WithField("keys", keys).Debugln("Returning sorted keys")
	return keys
}

// formatBlockKey returns a formatted string of a block's type and labels
func formatBlockKey(block *hclsyntax.Block) string {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting formatBlockKey")

	if len(block.Labels) > 0 {
		return block.Type + " " + strings.Join(block.Labels, " ")
	}
	return block.Type
}

// getBlockOpeningBytes returns the opening byte array of a block.
func getBlockOpeningBytes(block *hclsyntax.Block) ([]byte, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getBlockOpeningBytes")

	// Initialize the output
	var output []byte

	if !params.RemoveComments {
		log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Debugln("Getting node comment")
		lines, err := getLinesFromFile(block.TypeRange.Filename)
		if err != nil {
			return nil, fmt.Errorf("could not get lines from file: %w", err)
		}

		startLine := block.TypeRange.Start.Line - 1 // Subtract 1 to account for 0-indexing of string arrays vs HCL line numbers
		nodeComment := getNodeComment(lines, startLine)

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
func getBlockBodyBytes(block *hclsyntax.Block, keys []string) ([]byte, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getBlockBodyBytes")

	var output []byte
	var buffer []byte

	// Get the path of the file
	path, err := getPathFromBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not get path from block: %w", err)
	}

	// Loop over the keys
	for i, key := range keys {
		// Write the block attributes
		if attribute, ok := block.Body.Attributes[key]; ok {
			log.WithField("attribute.Name", attribute.Name).Debugln("Found key in block attributes")
			b, err := getAttributeBytes(attribute, path)
			if err != nil {
				return nil, fmt.Errorf("could not write attribute: %w", err)
			}
			buffer = append(buffer, b...)
			continue
		}

		// Write the block child blocks
		for _, childBlock := range block.Body.Blocks {
			childBlockKey := formatBlockKey(childBlock)
			log.WithFields(log.Fields{"key": key, "childBlockKey": childBlockKey}).Traceln("Checking key against child block")
			if childBlockKey == key {
				log.WithField("childBlock", childBlock).Debugln("Found child block in blocks")
				buffer = addNewLineIfBufferExists(buffer)
				b, err := getSortedBlockBytes(childBlock)
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
				break
			}
		}
	}

	// Flush the remaining buffer to the output
	if len(buffer) > 0 {
		output = append(output, buffer...)
	}

	return output, nil
}

// addNewlineIfBufferExists adds a newline to the buffer if the buffer is not empty.
func addNewLineIfBufferExists(buffer []byte) []byte {
	log.WithField("buffer", buffer).Traceln("Starting addNewLineIfBufferExists")

	if len(buffer) > 0 {
		buffer = append(buffer, []byte("\n")...)
	}
	return buffer
}

// getAttributeBytes returns the byte array of an attribute.
func getAttributeBytes(attribute *hclsyntax.Attribute, path string) ([]byte, error) {
	log.WithField("attribute.Name", attribute.Name).Traceln("Starting getAttributeBytes")

	content, err := readNodeFromFile(
		path,
		attribute.Range().Start.Line,
		attribute.Range().Start.Column,
		attribute.Range().End.Line,
		attribute.Range().End.Column,
	)
	if err != nil {
		return nil, fmt.Errorf("could not read file contents: %w", err)
	}

	s := strings.Join(content, "\n")
	b := []byte(fmt.Sprintf("%s\n", s))

	return b, nil
}

// getBlockClosingBytes returns the closing byte array of a block
func getBlockClosingBytes(block *hclsyntax.Block) []byte {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getBlockClosingBytes")

	return []byte("}\n")
}

// Returns the file path of the block
func getPathFromBlock(block *hclsyntax.Block) (string, error) {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getPathFromBlock")

	path, err := filepath.Abs(block.TypeRange.Filename)
	if err != nil {
		return "", fmt.Errorf("could not get absolute path from block: %w", err)
	}
	return path, nil
}

// readNodeFromFile reads a node from a file and returns the contents as a slice of strings
//
// The startLine and startCol are inclusive
// The endLine and endCol are exclusive
func readNodeFromFile(filename string, startLine, startCol, endLine, endCol int) ([]string, error) {
	log.WithFields(log.Fields{"filename": filename, "startLine": startLine, "startCol": startCol, "endLine": endLine, "endCol": endCol}).Traceln("Starting readNodeFromFile")

	// HCL lines are 1-indexed, so we need to subtract 1 from the start and end lines
	startLine--
	endLine--

	lines, err := getLinesFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not get lines from file: %w", err)
	}

	var output []string

	if !params.RemoveComments {
		nodeComment := getNodeComment(lines, startLine)

		if len(nodeComment) > 0 {
			output = append(output, nodeComment...)
		}
	}

	for i := startLine; i <= endLine; i++ {
		// Grab the current line
		line := lines[i]
		log.WithField("line", line).Debugln("Current line")

		// Grab the correct slice of the line
		lineSlice := getLineSlice(line, startLine, endLine, i, startCol, endCol)

		// Append the line slice to the buffer
		if len(lineSlice) > 0 {
			output = append(output, lineSlice)
		}
	}

	log.WithField("output", output).Debugln("Returning output")
	return output, nil
}

// getLineSlice returns the correct slice of the line based on the start and end columns
func getLineSlice(line string, startLine, endLine, currentLine, startCol, endCol int) string {
	log.WithFields(log.Fields{"line": line, "startLine": startLine, "endLine": endLine, "currentLine": currentLine, "startCol": startCol, "endCol": endCol}).Traceln("Starting getLineSlice")

	// Check if it is the starting line
	if currentLine == startLine {
		log.Debugln("Current line is starting line.")
		// Truncate the line to the starting column
		line = line[startCol-1:]
	}

	if params.RemoveComments {
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
