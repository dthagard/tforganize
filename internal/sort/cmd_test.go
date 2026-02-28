package sort

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGetCommand(t *testing.T) {
	cmd := GetCommand()

	if cmd.Use == "" {
		t.Fatal("command Use must not be empty")
	}

	expectedFlags := []string{
		"group-by-type",
		"has-header",
		"header-pattern",
		"header-end-pattern",
		"keep-header",
		"inline",
		"output-dir",
		"remove-comments",
		"check",
		"recursive",
		"diff",
		"no-sort-by-type",
		"strip-section-comments",
		"compact-empty-blocks",
		"exclude",
	}

	for _, flag := range expectedFlags {
		if cmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("expected persistent flag %q to be registered", flag)
		}
	}
}

func TestSortStdinInlineError(t *testing.T) {
	err := sortStdin(&Params{Inline: true})
	if err == nil {
		t.Fatal("expected error when --inline is used with stdin, got nil")
	}
	if !strings.Contains(err.Error(), "inline") {
		t.Errorf("error %q should mention inline", err.Error())
	}
}

func TestSortStdinSuccess(t *testing.T) {
	// Save and restore os.Stdin and os.Stdout.
	origStdin := os.Stdin
	origStdout := os.Stdout
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})

	// Pipe HCL content into stdin.
	input := `resource "aws_instance" "web" {
  ami = "ami-web"
}

resource "aws_instance" "app" {
  ami = "ami-app"
}
`
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create stdin pipe: %v", err)
	}
	go func() {
		_, _ = pw.WriteString(input)
		pw.Close()
	}()
	os.Stdin = pr

	// Capture stdout.
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create stdout pipe: %v", err)
	}
	os.Stdout = outW

	if err := sortStdin(&Params{}); err != nil {
		outW.Close()
		t.Fatalf("sortStdin returned unexpected error: %v", err)
	}
	outW.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, outR); err != nil {
		t.Fatalf("could not read stdout: %v", err)
	}

	out := buf.String()
	appIdx := strings.Index(out, "app")
	webIdx := strings.Index(out, "web")
	if appIdx == -1 || webIdx == -1 {
		t.Fatalf("expected both app and web in output, got:\n%s", out)
	}
	if appIdx > webIdx {
		t.Errorf("app should come before web in sorted output")
	}
}

func TestGetCommandRunE(t *testing.T) {
	dir := t.TempDir()
	tfPath := dir + "/main.tf"
	content := `resource "aws_instance" "b" {
  ami = "ami-b"
}

resource "aws_instance" "a" {
  ami = "ami-a"
}
`
	if err := os.WriteFile(tfPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cmd := GetCommand()
	cmd.SetArgs([]string{"--output-dir", outDir, tfPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	outBytes, err := os.ReadFile(outDir + "/main.tf")
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	out := string(outBytes)
	aIdx := strings.Index(out, "\"a\"")
	bIdx := strings.Index(out, "\"b\"")
	if aIdx == -1 || bIdx == -1 || aIdx > bIdx {
		t.Errorf("expected a before b in sorted output:\n%s", out)
	}
}
