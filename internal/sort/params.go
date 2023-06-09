package sort

// params holds the run-time parameters for the sort command.
var params = &Params{}

// initParams is a helper function to initialize the Params instance with default values.
func initParams() {
	params = &Params{
		GroupByType:    false,
		HasHeader:      false,
		HeaderPattern:  "",
		KeepHeader:     false,
		OutputDir:      "",
		RemoveComments: false,
	}
}
