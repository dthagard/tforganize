package sort

import "testing"

func FuzzIsStartOfComment(f *testing.F) {
	seeds := []string{
		"#",
		"//",
		"/*",
		"/**",
		"# comment",
		"// comment",
		"/* block comment",
		"  # indented",
		"  // indented",
		"/",
		"*",
		"",
		"   ",
		"not a comment",
		"also not # inline",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, line string) {
		isStartOfComment(line)
	})
}

func FuzzIsEndOfComment(f *testing.F) {
	seeds := []string{
		"*/",
		"**/",
		"  */",
		"***************************************************************/",
		"*",
		"/",
		"",
		"   ",
		"not a comment",
		"#",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, line string) {
		isEndOfComment(line)
	})
}
