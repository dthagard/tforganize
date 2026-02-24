package sort

import (
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
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

func TestGetNodeComment(t *testing.T) {

	/*********************************************************************/
	// Block comment directly adjacent to the resource (no blank line).
	// The closing */ must not inject a spurious blank line.
	/*********************************************************************/

	t.Run("block comment adjacent to block", func(t *testing.T) {
		lines := []string{
			"/*",
			" * This is a block comment",
			" */",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		// startLine is the 0-indexed position of the resource line.
		expected := []string{
			"/*",
			" * This is a block comment",
			" */",
		}
		result := getNodeComment(lines, 3)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getNodeComment() = %v, want %v", result, expected)
		}
	})

	/*********************************************************************/
	// Block comment separated from the resource by one blank line.
	// That single blank line must be preserved in the output.
	/*********************************************************************/

	t.Run("block comment with blank line before block", func(t *testing.T) {
		lines := []string{
			"/*",
			" * This is a block comment",
			" */",
			"",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		expected := []string{
			"/*",
			" * This is a block comment",
			" */",
			"",
		}
		result := getNodeComment(lines, 4)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getNodeComment() = %v, want %v", result, expected)
		}
	})

	/*********************************************************************/
	// Consecutive single-line comments adjacent to the resource.
	// Both lines must be captured; no spurious blank line appended.
	/*********************************************************************/

	t.Run("consecutive single-line comments adjacent to block", func(t *testing.T) {
		lines := []string{
			"# First comment",
			"# Second comment",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		expected := []string{
			"# First comment",
			"# Second comment",
		}
		result := getNodeComment(lines, 2)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("getNodeComment() = %v, want %v", result, expected)
		}
	})

	/*********************************************************************/
	// Block with no preceding comment at all.
	// The function must return an empty slice (not nil panic).
	/*********************************************************************/

	t.Run("block with no comment", func(t *testing.T) {
		lines := []string{
			`variable "other" {`,
			"  default = 1",
			"}",
			`resource "aws_instance" "foo" {`,
			"}",
		}
		result := getNodeComment(lines, 3)
		if len(result) != 0 {
			t.Errorf("getNodeComment() = %v, want empty slice", result)
		}
	})
}

func TestParseHclFileUsesInjectedFileSystem(t *testing.T) {
	// Save and restore the original filesystem
	originalFS := FS
	originalAFS := AFS
	t.Cleanup(func() {
		FS = originalFS
		AFS = originalAFS
	})

	// Inject an in-memory filesystem
	SetFileSystem(afero.NewMemMapFs())

	// Write a minimal .tf file into the in-memory FS
	tfPath := "/test/main.tf"
	tfContent := []byte("resource \"null_resource\" \"example\" {}\n")
	if err := AFS.MkdirAll("/test", 0755); err != nil {
		t.Fatalf("could not create directory: %v", err)
	}
	if err := AFS.WriteFile(tfPath, tfContent, 0644); err != nil {
		t.Fatalf("could not write tf file: %v", err)
	}

	body, err := parseHclFile(tfPath)
	if err != nil {
		t.Fatalf("parseHclFile() returned unexpected error: %v", err)
	}
	if body == nil {
		t.Fatal("parseHclFile() returned nil body, expected non-nil")
	}
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
