package sort

import (
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func init() {
	// No-op
}

func hclCleanup() {
	// No-op
}

func TestIsSortable(t *testing.T) {
	t.Cleanup(hclCleanup)

	/*********************************************************************/
	// Happy path test for isSortable() with a sortable file
	/*********************************************************************/

	t.Run("sortable file", func(t *testing.T) {
		testDir := t.TempDir()

		// Create a sortable file
		sortable, err := AFS.Create(filepath.Join(testDir, "foo.tf"))
		if err != nil {
			log.WithError(err).Errorln("could not create sortable file")
		}
		defer sortable.Close()

		// Get sortable file info
		sortableStat, err := sortable.Stat()
		if err != nil {
			log.WithError(err).Errorln("could not get sortable file")
		}

		// Test sortable file
		expected := true
		if result := isSortable(sortableStat); result != expected {
			t.Errorf("isSortable() returned %t, expected %t", result, expected)
		}
	})

	/*********************************************************************/
	// Happy path test for isSortable() with a non-sortable file
	/*********************************************************************/

	t.Run("non-sortable file", func(t *testing.T) {
		testDir := t.TempDir()

		// Create a non-sortable file
		unsortable, err := AFS.Create(filepath.Join(testDir, "main.txt"))
		if err != nil {
			log.WithError(err).Errorln("could not create sortable file")
		}
		defer unsortable.Close()

		// Get sortable file info
		nonSortableStat, err := unsortable.Stat()
		if err != nil {
			log.WithError(err).Errorln("could not get sortable file")
		}

		// Test sortable file
		expected := false
		if result := isSortable(nonSortableStat); result != expected {
			t.Errorf("isSortable() returned %t, expected %t", result, expected)
		}
	})
}

func TestIsStartOfComment(t *testing.T) {

	/*********************************************************************/
	// Happy path test for isStartOfComment() with a comment string
	/*********************************************************************/

	t.Run("comment string", func(t *testing.T) {
		testData := []string{
			"/***************************************************************",
			"/**",
			"/*",
			"#",
			"//",
			"#################################################################",
			"  #",
		}

		for _, s := range testData {
			result := isStartOfComment(s)
			expected := true
			if result != expected {
				t.Errorf("isStartOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})

	/*********************************************************************/
	// Sad path test for isStartOfComment() with non-comment string
	/*********************************************************************/

	t.Run("not comment string", func(t *testing.T) {
		testData := []string{
			"***************************************************************",
			"/",
			"   ",
			"",
			"this is not a comment",
			"also not a comment ## foo",
			"  foobar",
		}

		for _, s := range testData {
			result := isStartOfComment(s)
			expected := false
			if result != expected {
				t.Errorf("isStartOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})
}

func TestIsEndOfComment(t *testing.T) {

	/*********************************************************************/
	// Happy path test for isEndOfComment() with a closing comment string
	/*********************************************************************/

	t.Run("comment string", func(t *testing.T) {
		testData := []string{
			"***************************************************************/",
			"**/",
			"*/",
			"  */",
		}

		for _, s := range testData {
			result := isEndOfComment(s)
			expected := true
			if result != expected {
				t.Errorf("isEndOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})

	/*********************************************************************/
	// Sad path test for isEndOfComment() with non-comment string
	/*********************************************************************/

	t.Run("not comment string", func(t *testing.T) {
		testData := []string{
			"***************************************************************",
			"/",
			"   ",
			"",
			"#",
			"this is not a comment",
			"  foobar",
		}

		for _, s := range testData {
			result := isEndOfComment(s)
			expected := false
			if result != expected {
				t.Errorf("isEndOfComment(%s) returned %v, expected %v\n", s, result, expected)
			}
		}
	})
}

func TestRemoveLeadingEmptyLines(t *testing.T) {

	/*********************************************************************/
	// Happy path test for removeLeadingEmptyLines() with a leading empty lines
	/*********************************************************************/

	t.Run("with leading lines", func(t *testing.T) {
		testData := []string{
			"",
			"",
			"/******************************************",
			"  Mandatory firewall rules",
			" *****************************************/",
		}

		expected := []string{
			"/******************************************",
			"  Mandatory firewall rules",
			" *****************************************/",
		}

		result := removeLeadingEmptyLines(testData)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("removeLeadingEmptyLines returned %v, expected %v\n", result, expected)
		}
	})
}
