package checkpoint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adamroman0/cosyra-context/internal/cosyra"
	"github.com/adamroman0/cosyra-context/internal/store"
)

func TestUpdateWritesBoundedSanitizedContext(t *testing.T) {
	project := t.TempDir()
	if _, err := cosyra.EnableProject(project, []string{"claude"}); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(context.Background(), filepath.Join(project, ".cosyra"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := Update(context.Background(), db, UpdateInput{
		ProjectRoot:    project,
		Agent:          "claude",
		AgentSessionID: "s1",
		TurnKey:        "t1",
		Summary:        "used token=supersecretvalue123",
		UpdatedAt:      time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(project, ".cosyra", "context.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "supersecretvalue123") {
		t.Fatalf("context contains secret: %s", text)
	}
	if !strings.Contains(text, "Latest Handoff") {
		t.Fatalf("missing expected section: %s", text)
	}
	if len(data) > DefaultMaxContextBytes {
		t.Fatalf("context too large: %d", len(data))
	}
}
