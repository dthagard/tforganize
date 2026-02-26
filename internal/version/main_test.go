package version

import (
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
	os.Stdout = w

	cmd := GetCommand()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		os.Stdout = oldStdout
		t.Fatalf("version command returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	r.Close()
	output := string(buf[:n])

	if !strings.Contains(output, info.AppVersion) {
		t.Errorf("version output %q does not contain AppVersion %q", output, info.AppVersion)
	}

	expected := "tforganize " + info.AppVersion + "\n"
	if output != expected {
		t.Errorf("version output = %q, want %q", output, expected)
	}
}
