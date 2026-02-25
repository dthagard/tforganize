package sort

import "errors"

// ErrCheckFailed is returned by Sort (and Sorter.run) when --check mode is
// enabled and one or more files would be changed by sorting.
// Callers can test for this specific condition with errors.Is:
//
//	if errors.Is(err, ErrCheckFailed) { ... }
var ErrCheckFailed = errors.New("tforganize: one or more files would be changed by sort")
