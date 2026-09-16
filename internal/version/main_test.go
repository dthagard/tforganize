package version

import (
	"io"
	"os"
	"strings"
	"testing"

	info "github.com/dthagard/tforganize/internal/info"
)

func TestVersionCommand(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	t.Cleanup(func() { os.Stdout = oldStdout })
	os.Stdout = w

	cmd := GetCommand()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		os.Stdout = oldStdout
		t.Fatalf("version command returned error: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = oldStdout

	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	output := string(buf)

	if !strings.Contains(output, info.AppVersion) {
		t.Errorf("version output %q does not contain AppVersion %q", output, info.AppVersion)
	}

	expected := "tforganize " + info.AppVersion + "\n"
	if output != expected {
		t.Errorf("version output = %q, want %q", output, expected)
	}
}
