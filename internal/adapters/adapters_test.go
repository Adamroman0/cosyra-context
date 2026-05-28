package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallClaudeMergesSettings(t *testing.T) {
	project := t.TempDir()
	if err := installClaude(project, filepath.Join(project, ".cosyra", "hooks"), "/bin/cosyra"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(project, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "SessionStart") || !strings.Contains(string(data), "Stop") {
		t.Fatalf("expected Claude hooks, got %s", string(data))
	}
}

func TestRemoveMarkedBlock(t *testing.T) {
	content := "a\n# cosyra-context:start project=/tmp/p\nb\n# cosyra-context:end project=/tmp/p\nc"
	got := removeMarkedBlock(content, "/tmp/p")
	if got != "a\nc" {
		t.Fatalf("unexpected content: %q", got)
	}
}
