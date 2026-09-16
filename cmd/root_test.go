package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestConfigSelection(t *testing.T) {
	for _, tc := range []struct {
		name          string
		gitFile       bool
		nested        bool
		outsideGit    bool
		outerRepo     bool
		projectConfig string
		homeConfig    string
		explicit      bool
		env           string
		args          []string
		wantComment   bool
	}{
		{name: "project before home", projectConfig: "remove-comments: true\n", homeConfig: "remove-comments: false\n"},
		{name: "nested invocation", nested: true, projectConfig: "remove-comments: true\n", homeConfig: "remove-comments: false\n"},
		{name: "worktree root", gitFile: true, nested: true, projectConfig: "remove-comments: true\n", homeConfig: "remove-comments: false\n"},
		{name: "outside Git uses cwd", outsideGit: true, projectConfig: "remove-comments: true\n", homeConfig: "remove-comments: false\n"},
		{name: "outside Git ignores parent config", outsideGit: true, outerRepo: true, homeConfig: "remove-comments: false\n", wantComment: true},
		{name: "nearest Git root stops discovery", outerRepo: true, homeConfig: "remove-comments: false\n", wantComment: true},
		{name: "home fallback", homeConfig: "remove-comments: true\n"},
		{name: "no config", wantComment: true},
		{name: "project does not merge home", projectConfig: "compact-empty-blocks: true\n", homeConfig: "remove-comments: true\n", wantComment: true},
		{name: "explicit before project", projectConfig: "remove-comments: true\n", explicit: true, wantComment: true},
		{name: "environment before config", projectConfig: "remove-comments: true\n", env: "false", wantComment: true},
		{name: "flag before environment", projectConfig: "remove-comments: false\n", env: "false", args: []string{"--remove-comments=true"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			home := filepath.Join(dir, "home")
			project := filepath.Join(dir, "project")
			for _, path := range []string{home, project} {
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			}
			writeFile := func(path, content string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.outerRepo {
				writeFile(filepath.Join(dir, ".tforganize.yaml"), "remove-comments: true\n")
				if !tc.outsideGit {
					if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
						t.Fatal(err)
					}
				}
			}
			if tc.gitFile {
				writeFile(filepath.Join(project, ".git"), "gitdir: ../worktree-metadata\n")
			} else if !tc.outsideGit {
				if err := os.Mkdir(filepath.Join(project, ".git"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if tc.projectConfig != "" {
				writeFile(filepath.Join(project, ".tforganize.yaml"), tc.projectConfig)
			}
			if tc.homeConfig != "" {
				writeFile(filepath.Join(home, ".tforganize.yaml"), tc.homeConfig)
			}
			cwd := project
			if tc.nested {
				cwd = filepath.Join(project, "modules")
				if err := os.Mkdir(cwd, 0755); err != nil {
					t.Fatal(err)
				}
				// A nested config must not shadow the Git-root config.
				writeFile(filepath.Join(cwd, ".tforganize.yaml"), "remove-comments: false\n")
			}
			original, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(cwd); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Chdir(original); err != nil {
					t.Fatal(err)
				}
			})
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("TFORGANIZE_REMOVE_COMMENTS", tc.env)
			t.Setenv("TFORGANIZE_COMPACT_EMPTY_BLOCKS", "")
			writeFile("main.tf", "# sentinel-comment\nresource \"null_resource\" \"example\" {\n}\n")
			args := []string{"sort", "--inline", "main.tf"}
			if tc.explicit {
				writeFile("explicit.yaml", "remove-comments: false\n")
				args = append(args, "--config=explicit.yaml")
			}
			args = append(args, tc.args...)
			rc := NewRootCommand()
			rc.baseCmd.SetArgs(args)
			if err := rc.baseCmd.Execute(); err != nil {
				t.Fatal(err)
			}
			out, err := os.ReadFile("main.tf")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(out), "sentinel-comment") != tc.wantComment {
				t.Fatalf("want comment preserved=%v; got:\n%s", tc.wantComment, out)
			}
		})
	}
}

func TestPlainFormatterFormat(t *testing.T) {
	f := &PlainFormatter{}
	entry := &log.Entry{Message: "hello world"}
	out, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}
	expected := "hello world\n"
	if string(out) != expected {
		t.Errorf("Format() = %q, want %q", string(out), expected)
	}
}

func TestToggleDebugEnabled(t *testing.T) {
	logger := log.StandardLogger()
	origLevel, origFormatter, origOutput, origDebug := logger.Level, logger.Formatter, logger.Out, debug
	t.Cleanup(func() {
		log.SetLevel(origLevel)
		log.SetFormatter(origFormatter)
		log.SetOutput(origOutput)
		debug = origDebug
	})
	var output bytes.Buffer
	log.SetOutput(&output)
	debug = true
	toggleDebug(nil, nil)
	log.Trace("trace-sentinel")
	if !strings.Contains(output.String(), "trace-sentinel") {
		t.Fatalf("debug logging omitted trace message: %q", output.String())
	}
	output.Reset()
	debug = false
	toggleDebug(nil, nil)
	log.Info("plain-sentinel")
	if output.String() != "plain-sentinel\n" {
		t.Fatalf("non-debug logging is not plain: %q", output.String())
	}
}

func TestExecuteExitStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{name: "success", args: []string{"--help"}},
		{name: "failure", args: []string{"--unknown-flag"}, want: 1},
		{name: "check failure", args: []string{"sort", "--check", "main.tf"}, want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("HOME", dir)
			t.Setenv("USERPROFILE", dir)
			content := "locals {\n z = 2\n a = 1\n}\n"
			if err := os.WriteFile("main.tf", []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			rc := NewRootCommand()
			rc.baseCmd.SetArgs(tc.args)
			rc.baseCmd.SetOut(&diagnostics)
			rc.baseCmd.SetErr(&diagnostics)
			status, calls := 0, 0
			rc.execute(func(code int) { status = code; calls++ })
			if status != tc.want || (calls != 0) != (tc.want != 0) || calls > 1 {
				t.Fatalf("exit status=%d calls=%d; want status=%d", status, calls, tc.want)
			}
			if tc.want == 0 && !strings.Contains(diagnostics.String(), "sort") {
				t.Fatalf("help omitted sort command: %q", diagnostics.String())
			}
			if tc.want == 1 && !strings.Contains(diagnostics.String(), "unknown-flag") {
				t.Fatalf("missing invalid flag diagnostic: %q", diagnostics.String())
			}
			got, err := os.ReadFile("main.tf")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != content {
				t.Fatalf("non-writing invocation changed input: %q", got)
			}
		})
	}
}

func TestConfigFailuresStopCommand(t *testing.T) {
	for _, name := range []string{"missing explicit config", "malformed config", "missing home", "deleted cwd", "invalid git marker"} {
		t.Run(name, func(t *testing.T) {
			if runtime.GOOS == "windows" && (name == "deleted cwd" || name == "invalid git marker") {
				t.Skip("requires Unix directory and symlink semantics")
			}
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("HOME", dir)
			t.Setenv("USERPROFILE", dir)
			rc := NewRootCommand()
			args := []string{"probe"}
			switch name {
			case "missing explicit config":
				args = append(args, "--config", filepath.Join(dir, "missing.yaml"))
			case "malformed config":
				path := filepath.Join(dir, "invalid.yaml")
				if err := os.WriteFile(path, []byte("invalid: [\n"), 0600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--config", path)
			case "missing home":
				t.Setenv("HOME", "")
				t.Setenv("USERPROFILE", "")
				t.Setenv("home", "")
			case "deleted cwd":
				t.Setenv("PWD", "")
				if err := os.Remove(dir); err != nil {
					t.Fatal(err)
				}
			case "invalid git marker":
				if err := os.Symlink(".git", filepath.Join(dir, ".git")); err != nil {
					t.Fatal(err)
				}
			}
			ran := false
			rc.baseCmd.AddCommand(&cobra.Command{
				Use: "probe",
				Run: func(*cobra.Command, []string) { ran = true },
			})
			rc.baseCmd.SetArgs(args)
			var diagnostics bytes.Buffer
			rc.baseCmd.SetOut(&diagnostics)
			rc.baseCmd.SetErr(&diagnostics)
			err := rc.baseCmd.Execute()
			if err == nil || ran {
				t.Fatalf("config failure must prevent execution: err=%v ran=%v", err, ran)
			}
			if name == "missing explicit config" && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing config should preserve filesystem error: %v", err)
			}
			if name == "deleted cwd" && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("deleted cwd should preserve filesystem error: %v", err)
			}
		})
	}
}

func TestBindCollectionFlags(t *testing.T) {
	for _, kind := range []string{"stringArray", "stringSlice"} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/explicit=%t", kind, explicit), func(t *testing.T) {
				cmd := &cobra.Command{Use: "probe"}
				var values []string
				if kind == "stringArray" {
					cmd.Flags().StringArrayVar(&values, "exclude", nil, "")
				} else {
					cmd.Flags().StringSliceVar(&values, "exclude", nil, "")
				}
				want := []string{"vendor/**", "generated/**"}
				v := viper.New()
				v.Set("exclude", want)
				if explicit {
					if err := cmd.ParseFlags([]string{"--exclude=chosen/**"}); err != nil {
						t.Fatal(err)
					}
					want = []string{"chosen/**"}
				}
				bindFlags(cmd, v)
				if !reflect.DeepEqual(values, want) {
					t.Fatalf("bound values=%q, want %q", values, want)
				}
			})
		}
	}
}
