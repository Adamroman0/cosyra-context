package cosyra

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnableProjectCreatesConfigContextAndGitignore(t *testing.T) {
	project := t.TempDir()

	cfg, err := EnableProject(project, []string{"codex", "claude", "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled {
		t.Fatal("expected enabled config")
	}
	if len(cfg.EnabledTools) != 2 {
		t.Fatalf("expected deduped tools, got %v", cfg.EnabledTools)
	}

	if _, err := os.Stat(filepath.Join(project, ".cosyra", "config.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".cosyra", "context.md")); err != nil {
		t.Fatal(err)
	}

	gitignore, err := os.ReadFile(filepath.Join(project, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gitignore) != ".cosyra/\n" {
		t.Fatalf("unexpected gitignore: %q", string(gitignore))
	}
}

type recordingInstaller struct {
	installed   [][]string
	uninstalled [][]string
}

func (r *recordingInstaller) Install(projectRoot string, tools []string) error {
	r.installed = append(r.installed, tools)
	return nil
}

func (r *recordingInstaller) Uninstall(projectRoot string, tools []string) error {
	r.uninstalled = append(r.uninstalled, tools)
	return nil
}

func TestDisableProjectPurgeUninstallsHooks(t *testing.T) {
	project := t.TempDir()
	installer := &recordingInstaller{}
	if _, err := EnableProjectWithInstaller(project, []string{"claude", "codex"}, installer); err != nil {
		t.Fatal(err)
	}

	if err := DisableProjectWithInstaller(project, true, installer); err != nil {
		t.Fatal(err)
	}

	if len(installer.uninstalled) != 1 {
		t.Fatalf("expected exactly one uninstall call on purge, got %d", len(installer.uninstalled))
	}
	if got := installer.uninstalled[0]; len(got) != 2 {
		t.Fatalf("expected both tools uninstalled, got %v", got)
	}
	if _, err := os.Stat(filepath.Join(project, ".cosyra")); !os.IsNotExist(err) {
		t.Fatalf("expected .cosyra removed on purge, stat err = %v", err)
	}
}

func TestDisableProjectKeepsDataByDefault(t *testing.T) {
	project := t.TempDir()
	if _, err := EnableProject(project, []string{"claude"}); err != nil {
		t.Fatal(err)
	}
	if err := DisableProject(project, false); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(project)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Fatal("expected disabled config")
	}
	if _, err := os.Stat(filepath.Join(project, ".cosyra")); err != nil {
		t.Fatal(err)
	}
}
