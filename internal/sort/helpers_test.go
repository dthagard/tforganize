package sort

import (
	"testing"
)

func TestStringExists(t *testing.T) {
	arr := []string{
		"foo",
		"bar",
		"baz",
	}

	/*********************************************************************/
	// Happy path test for stringExists() with string in array
	/*********************************************************************/

	t.Run("string in array", func(t *testing.T) {
		if res := stringExists(arr, "foo"); !res {
			t.Errorf("stringExists() returned %v, expected %v", res, true)
		}
	})

	/*********************************************************************/
	// Sad path test for stringExists() with string missing from array
	/*********************************************************************/

	t.Run("string missing from array", func(t *testing.T) {
		if res := stringExists(arr, "buzz"); res {
			t.Errorf("stringExists() returned %v, expected %v", res, false)
		}
	})

	/*********************************************************************/
	// Sad path test for stringExists() with untrimmed string
	/*********************************************************************/

	t.Run("untrimmed string", func(t *testing.T) {
		if res := stringExists(arr, "foo "); res {
			t.Errorf("stringExists() returned %v, expected %v", res, false)
		}
	})
}

func TestReverseStringArray(t *testing.T) {
	/*********************************************************************/
	// Happy path test for reverseStringArray() with correctly reversed strings
	/*********************************************************************/

	t.Run("correctly reversed strings", func(t *testing.T) {
		arr := map[int][]string{}
		arr[0] = []string{
			"foo",
			"bar",
		}
		arr[1] = []string{
			"foo",
			"bar",
			"baz",
		}
		arr[2] = []string{
			"foo",
			"bar",
			"baz",
			"buzz",
		}

		arrRev := map[int][]string{}
		arrRev[0] = []string{
			"bar",
			"foo",
		}
		arrRev[1] = []string{
			"baz",
			"bar",
			"foo",
		}
		arrRev[2] = []string{
			"buzz",
			"baz",
			"bar",
			"foo",
		}

		for i, a := range arr {
			res := reverseStringArray(a)
			if len(res) != len(arrRev[i]) {
				t.Errorf("reverseStringArray() returned %v, expected %v\n", res, arrRev[i])
			}

			for j, str := range res {
				if str != arrRev[i][j] {
					t.Errorf("reverseStringArray() returned %s, expected %s\n", str, arrRev[i][j])
				}
			}
		}
	})
}
