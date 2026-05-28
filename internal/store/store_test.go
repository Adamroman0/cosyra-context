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

func TestPruneBeforeRemovesOldTurns(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().UTC()
	if err := s.UpsertProject(ctx, Project{Root: "/tmp/project", Enabled: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-10 * 24 * time.Hour)
	if err := s.UpsertTurn(ctx, Turn{
		ProjectRoot:    "/tmp/project",
		Agent:          "claude",
		AgentSessionID: "s1",
		TurnKey:        "old",
		Status:         "completed",
		CreatedAt:      old,
		UpdatedAt:      old,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.PruneBefore(ctx, now.Add(-7*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	count, err := s.TurnCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected pruned turn, got %d", count)
	}
}
