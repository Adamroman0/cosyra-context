package store

import (
	"context"
	"testing"
	"time"
)

func TestOpenMigratesAndStoresProject(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().UTC()
	if err := s.UpsertProject(ctx, Project{
		Root:      "/tmp/project",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAdapter(ctx, "/tmp/project", "claude", true, now); err != nil {
		t.Fatal(err)
	}
	count, err := s.ProjectCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one project, got %d", count)
	}
}
