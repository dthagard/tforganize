package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
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
	// Save and restore log level.
	origLevel := log.GetLevel()
	t.Cleanup(func() {
		log.SetLevel(origLevel)
		debug = false
	})

	debug = true
	toggleDebug(nil, nil)

	if log.GetLevel() != log.TraceLevel {
		t.Errorf("expected TraceLevel, got %v", log.GetLevel())
	}
}
