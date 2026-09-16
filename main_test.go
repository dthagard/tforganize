package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainHelp(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := output.Close(); err != nil {
			t.Errorf("close output: %v", err)
		}
	}()
	args, stdout, stderr := os.Args, os.Stdout, os.Stderr
	t.Cleanup(func() {
		os.Args, os.Stdout, os.Stderr = args, stdout, stderr
	})
	os.Args = []string{"tforganize", "--help"}
	os.Stdout, os.Stderr = output, output
	main()
	content, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"sort", "version"} {
		if !strings.Contains(string(content), command) {
			t.Fatalf("help does not advertise %s: %s", command, content)
		}
	}
}
