package sort

import (
	"path/filepath"
	"reflect"
	"testing"

	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
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

func TestBlockListSorterLess(t *testing.T) {

	/*********************************************************************/
	// Different types: the lexicographically smaller type comes first.
	/*********************************************************************/

	t.Run("different types ordered alphabetically", func(t *testing.T) {
		bs := BlockListSorter{
			&hclsyntax.Block{Type: "resource"},
			&hclsyntax.Block{Type: "data"},
		}
		// data < resource, so Less(1,0) should be true
		if !bs.Less(1, 0) {
			t.Error("data should come before resource")
		}
		if bs.Less(0, 1) {
			t.Error("resource should not come before data")
		}
	})

	/*********************************************************************/
	// Same type, no labels on either block: len(0) < len(0) is false.
	/*********************************************************************/

	t.Run("same type no labels returns false", func(t *testing.T) {
		bs := BlockListSorter{
			&hclsyntax.Block{Type: "terraform"},
			&hclsyntax.Block{Type: "terraform"},
		}
		if bs.Less(0, 1) {
			t.Error("equal no-label blocks should not be Less")
		}
	})

	/*********************************************************************/
	// Same type, first label differs: label ordering determines result.
	/*********************************************************************/

	t.Run("same type first label differs", func(t *testing.T) {
		bs := BlockListSorter{
			&hclsyntax.Block{Type: "resource", Labels: []string{"aws_s3_bucket", "beta"}},
			&hclsyntax.Block{Type: "resource", Labels: []string{"aws_s3_bucket", "alpha"}},
		}
		// "alpha" < "beta", so block[1] should come before block[0]
		if !bs.Less(1, 0) {
			t.Error("alpha label should come before beta label")
		}
		if bs.Less(0, 1) {
			t.Error("beta label should not come before alpha label")
		}
	})

	/*********************************************************************/
	// Same type, same first label, block with fewer labels comes first.
	// Exercises the len(block1.Labels) < len(block2.Labels) return path.
	/*********************************************************************/

	t.Run("same type fewer labels comes first", func(t *testing.T) {
		bs := BlockListSorter{
			&hclsyntax.Block{Type: "resource", Labels: []string{"aws_s3_bucket"}},
			&hclsyntax.Block{Type: "resource", Labels: []string{"aws_s3_bucket", "extra"}},
		}
		if !bs.Less(0, 1) {
			t.Error("block with fewer labels should come first")
		}
		if bs.Less(1, 0) {
			t.Error("block with more labels should not come first")
		}
	})

	/*********************************************************************/
	// Same type, identical labels: not Less in either direction.
	/*********************************************************************/

	t.Run("same type identical labels returns false", func(t *testing.T) {
		bs := BlockListSorter{
			&hclsyntax.Block{Type: "resource", Labels: []string{"aws_s3_bucket", "my_bucket"}},
			&hclsyntax.Block{Type: "resource", Labels: []string{"aws_s3_bucket", "my_bucket"}},
		}
		if bs.Less(0, 1) || bs.Less(1, 0) {
			t.Error("identical blocks should not be Less in either direction")
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
