package sort

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	writeDone := make(chan error, 1)
	go func() {
		_, writeErr := pw.WriteString(input)
		writeDone <- errors.Join(writeErr, pw.Close())
	}()
	t.Cleanup(func() {
		if err := pr.Close(); err != nil {
			t.Errorf("close stdin reader: %v", err)
		}
	})
	os.Stdin = pr

	// Capture stdout.
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create stdout pipe: %v", err)
	}
	t.Cleanup(func() {
		if err := outR.Close(); err != nil {
			t.Errorf("close stdout reader: %v", err)
		}
	})
	os.Stdout = outW

	sortErr := sortStdin(&Params{})
	closeErr := outW.Close()
	if err := <-writeDone; err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	if err := errors.Join(sortErr, closeErr); err != nil {
		t.Fatalf("sort stdin and close output: %v", err)
	}

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

func TestCommandStdinDispatch(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "implicit"
		if explicit {
			name = "explicit"
		}
		t.Run(name, func(t *testing.T) {
			input, err := os.CreateTemp(t.TempDir(), "stdin")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := input.Close(); err != nil {
					t.Errorf("close stdin: %v", err)
				}
			}()
			if _, err := input.WriteString("locals {\n z = 2\n a = 1\n}\n"); err != nil {
				t.Fatal(err)
			}
			if _, err := input.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			output, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := output.Close(); err != nil {
					t.Errorf("close stdout: %v", err)
				}
			}()
			oldIn, oldOut := os.Stdin, os.Stdout
			t.Cleanup(func() { os.Stdin, os.Stdout = oldIn, oldOut })
			os.Stdin, os.Stdout = input, output
			cmd := GetCommand()
			args := []string{}
			if explicit {
				args = []string{"-"}
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(output.Name())
			if err != nil {
				t.Fatal(err)
			}
			want := "locals {\n  a = 1\n  z = 2\n}\n"
			if string(got) != want {
				t.Fatalf("stdin output=%q, want %q", got, want)
			}
		})
	}
}

func TestCommandRequiresTargetWithoutPipe(t *testing.T) {
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := input.Close(); err != nil {
			t.Errorf("close stdin: %v", err)
		}
	}()
	original := os.Stdin
	t.Cleanup(func() { os.Stdin = original })
	os.Stdin = input
	cmd := GetCommand()
	cmd.SetArgs([]string{})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no target") {
		t.Fatalf("expected missing target error, got %v", err)
	}
}

func TestSortStdinReadFailure(t *testing.T) {
	input, err := os.CreateTemp(t.TempDir(), "closed")
	if err != nil {
		t.Fatal(err)
	}
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	original := os.Stdin
	t.Cleanup(func() { os.Stdin = original })
	os.Stdin = input
	err = sortStdin(&Params{})
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected wrapped closed stdin error, got %v", err)
	}
}

func TestCommandRejectsInvalidStdin(t *testing.T) {
	input, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := input.Close(); err != nil {
			t.Errorf("close stdin: %v", err)
		}
	}()
	if _, err := input.WriteString("resource {\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	original := os.Stdin
	t.Cleanup(func() { os.Stdin = original })
	os.Stdin = input
	cmd := GetCommand()
	cmd.SetArgs([]string{"-"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "stdin.tf") {
		t.Fatalf("expected stdin parse diagnostic, got %v", err)
	}
}

func TestCommandStopsAfterTargetError(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.tf")
	next := filepath.Join(dir, "next.tf")
	content := "locals {\n z = 2\n a = 1\n}\n"
	if err := os.WriteFile(bad, []byte("resource {\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(next, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := GetCommand()
	cmd.SetArgs([]string{"--inline", bad, next})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "bad.tf") {
		t.Fatalf("expected first target parse diagnostic, got %v", err)
	}
	got, err := os.ReadFile(next)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("later target changed after failure: %q", got)
	}
}
