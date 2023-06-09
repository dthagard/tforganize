package sort

import (
	log "github.com/sirupsen/logrus"
)

// stringExists checks if a string exists in a slice of strings.
func stringExists(arr []string, target string) bool {
	log.WithFields(log.Fields{"arr": arr, "target": target}).Traceln("Starting stringExists")

	for _, str := range arr {
		if str == target {
			return true
		}
	}
	return false
}

// reverseStringArray reverses a string array.
func reverseStringArray(arr []string) []string {
	log.WithField("arr", arr).Traceln("Starting reverseStringArray")

	length := len(arr)
	for i := 0; i < length/2; i++ {
		// Swap elements from both ends
		arr[i], arr[length-1-i] = arr[length-1-i], arr[i]
	}
	return arr
}
