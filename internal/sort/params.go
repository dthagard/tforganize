package sort

// Deprecated: params is the package-level run-time parameters for the sort command.
// New callers should use NewSorter(params, fs) instead of relying on this variable.
// This variable will be removed in a future release once all callers have migrated.
var params = &Params{}

// Deprecated: initParams initialises the package-level params variable with default values.
// New callers should use NewSorter(params, fs) instead.
// This function will be removed in a future release.
func initParams() {
	params = &Params{
		GroupByType:    false,
		HasHeader:      false,
		HeaderPattern:  "",
		Inline:         false,
		KeepHeader:     false,
		OutputDir:      "",
		RemoveComments: false,
	}
	clearLinesCache()
}
