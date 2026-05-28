package hooks

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/adamroman0/cosyra-context/internal/cosyra"
	"github.com/adamroman0/cosyra-context/internal/store"
)

func TestHandleStopIsIdempotentByTurnKey(t *testing.T) {
	project := t.TempDir()
	if _, err := cosyra.EnableProject(project, []string{"claude"}); err != nil {
		t.Fatal(err)
	}
	payload := bytes.NewBufferString(`{"session_id":"s1","turn_id":"t1","last_agent_message":"done"}`)
	if _, err := HandleStop(project, "claude", payload); err != nil {
		t.Fatal(err)
	}
	payload = bytes.NewBufferString(`{"session_id":"s1","turn_id":"t1","last_agent_message":"done again"}`)
	if _, err := HandleStop(project, "claude", payload); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(context.Background(), filepath.Join(project, ".cosyra"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	count, err := db.TurnCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one idempotent turn, got %d", count)
	}
}
