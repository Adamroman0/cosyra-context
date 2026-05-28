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
