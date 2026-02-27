package sort

import (
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
	log "github.com/sirupsen/logrus"
)

const defaultFileGroup = "main.tf"

// metaArguments is a map of block types to meta arguments.
// The "pre" arguments are the ones that should be first inside of a block.
// The "post" arguments are the ones that should be last inside of a block.
// If a block type doesn't have meta arguments, the "default" ones are used.
// Pre meta arguments should be sorted in the order they should appear in the block.
//
// Sources:
//   - https://developer.hashicorp.com/terraform/language/resources/syntax
//   - https://developer.hashicorp.com/terraform/language/modules/develop/refactoring
//   - https://developer.hashicorp.com/terraform/language/import
var (
	// fileGroups maps block types to the canonical output file name used when
	// --group-by-type is enabled. Block types not listed here fall back to
	// defaultFileGroup ("main.tf").
	//
	// Default file name mapping:
	//   check     → checks.tf
	//   data      → data.tf
	//   import    → imports.tf
	//   locals    → locals.tf
	//   moved     → main.tf  (refactoring blocks live alongside resources)
	//   output    → outputs.tf
	//   removed   → main.tf  (refactoring blocks live alongside resources)
	//   terraform → versions.tf
	//   variable  → variables.tf
	fileGroups = map[string]string{
		"check":     "checks.tf",
		"data":      "data.tf",
		"import":    "imports.tf",
		"locals":    "locals.tf",
		"moved":     "main.tf",
		"output":    "outputs.tf",
		"removed":   "main.tf",
		"terraform": "versions.tf",
		"variable":  "variables.tf",
	}

	metaArguments = map[string]map[string][]string{
		"check": {
			"pre":  []string{"data"},
			"post": []string{},
		},
		"data": {
			"pre":  []string{"count", "for_each", "provider"},
			"post": []string{"provisioner", "depends_on"},
		},
		"dynamic": {
			"pre":  []string{"for_each"},
			"post": []string{},
		},
		// import block: https://developer.hashicorp.com/terraform/language/import
		// Required: to, id. Optional: provider.
		"import": {
			"pre":  []string{"to", "id", "provider"},
			"post": []string{},
		},
		"local": {
			"pre":  []string{},
			"post": []string{},
		},
		"module": {
			"pre":  []string{"source", "version", "providers", "count", "for_each"},
			"post": []string{"depends_on"},
		},
		// output block: https://developer.hashicorp.com/terraform/language/values/outputs
		// description and value first, sensitive/precondition/depends_on last.
		"output": {
			"pre":  []string{"description", "value"},
			"post": []string{"sensitive", "precondition", "depends_on"},
		},
		// moved block: https://developer.hashicorp.com/terraform/language/modules/develop/refactoring
		// Required: from, to.
		"moved": {
			"pre":  []string{"from", "to"},
			"post": []string{},
		},
		// removed block: https://developer.hashicorp.com/terraform/language/resources/syntax#removing-resources
		// Required: from (attribute), lifecycle (nested block with destroy bool).
		"removed": {
			"pre":  []string{"from"},
			"post": []string{"lifecycle"},
		},
		"resource": {
			"pre":  []string{"count", "for_each", "provider"},
			"post": []string{"provisioner", "lifecycle", "depends_on", "triggers_replace"},
		},
		"terraform": {
			"pre":  []string{"required_version", "required_providers"},
			"post": []string{},
		},
		"variable": {
			"pre": []string{"description", "type", "default", "nullable", "sensitive"},
			"post": []string{"validation"},
		},
		"default": {
			"pre":  []string{},
			"post": []string{},
		},
	}
)

// getMetaArguments returns the meta arguments that should be first and last inside of a block
func getMetaArguments(block *hclsyntax.Block) [][]string {
	log.WithFields(log.Fields{blockTypeLabel: block.Type, blockLabelsLabel: block.Labels}).Traceln("Starting getMetaArguments")

	// Initialize the return value
	metaArgs := make([][]string, 2)

	// Check if the block type has meta arguments
	if blockType, ok := metaArguments[block.Type]; ok {
		if args, ok := blockType["pre"]; ok {
			metaArgs[0] = args
		}
		if args, ok := blockType["post"]; ok {
			metaArgs[1] = args
		}
	}

	// If the block type doesn't have meta arguments, use the default ones
	if len(metaArgs[0]) == 0 {
		metaArgs[0] = metaArguments["default"]["pre"]
	}
	if len(metaArgs[1]) == 0 {
		metaArgs[1] = metaArguments["default"]["post"]
	}

	return metaArgs
}
