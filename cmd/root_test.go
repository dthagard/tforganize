package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRootCommand(t *testing.T) {
	rc := NewRootCommand()
	if rc.baseCmd == nil {
		t.Fatal("NewRootCommand returned nil baseCmd")
	}

	cmds := rc.baseCmd.Commands()
	names := make(map[string]bool)
	for _, c := range cmds {
		names[c.Name()] = true
	}
	for _, want := range []string{"sort", "version"} {
		if !names[want] {
			t.Errorf("expected sub-command %q to be registered", want)
		}
	}

	for _, flag := range []string{"config", "debug"} {
		if rc.baseCmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("expected persistent flag %q to be registered", flag)
		}
	}
}

func TestInitConfigNoFile(t *testing.T) {
	rc := NewRootCommand()
	rc.baseCmd.SetArgs([]string{"version"})
	if err := rc.baseCmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestInitConfigWithFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(cfgPath, []byte("group-by-type: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rc := NewRootCommand()
	rc.baseCmd.SetArgs([]string{"--config", cfgPath, "version"})
	if err := rc.baseCmd.Execute(); err != nil {
		t.Fatalf("Execute failed with --config: %v", err)
	}
}

func TestBindFlagsEnvVar(t *testing.T) {
	os.Setenv("TFORGANIZE_DEBUG", "true")
	defer os.Unsetenv("TFORGANIZE_DEBUG")

	rc := NewRootCommand()
	rc.baseCmd.SetArgs([]string{"version"})
	if err := rc.baseCmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}
