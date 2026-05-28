package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const DBFileName = "cosyra.db"

type Store struct {
	db *sql.DB
}

type Project struct {
	Root      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func Open(ctx context.Context, cosyraDir string) (*Store, error) {
	if err := os.MkdirAll(cosyraDir, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(cosyraDir, DBFileName))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DBPath(cosyraDir string) string {
	return filepath.Join(cosyraDir, DBFileName)
}

func (s *Store) configure(ctx context.Context) error {
	settings := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, statement := range settings {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			project_root TEXT PRIMARY KEY,
			enabled INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS installed_adapters (
			project_root TEXT NOT NULL,
			agent TEXT NOT NULL,
			enabled INTEGER NOT NULL,
			installed_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (project_root, agent),
			FOREIGN KEY (project_root) REFERENCES projects(project_root) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS agent_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_root TEXT NOT NULL,
			cosyra_session_id TEXT NOT NULL,
			agent TEXT NOT NULL,
			agent_session_id TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			ended_at TEXT,
			UNIQUE(project_root, agent, agent_session_id),
			FOREIGN KEY (project_root) REFERENCES projects(project_root) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_root TEXT NOT NULL,
			agent TEXT NOT NULL,
			agent_session_id TEXT NOT NULL,
			turn_key TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT,
			summary TEXT,
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(project_root, agent, agent_session_id, turn_key),
			FOREIGN KEY (project_root) REFERENCES projects(project_root) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS checkpoints (
			project_root TEXT PRIMARY KEY,
			context TEXT NOT NULL,
			context_bytes INTEGER NOT NULL,
			last_agent TEXT,
			last_agent_session_id TEXT,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (project_root) REFERENCES projects(project_root) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_sessions_project_agent_session ON agent_sessions(project_root, agent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_sessions_updated_at ON agent_sessions(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_turns_project_agent_session ON turns(project_root, agent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_turns_updated_at ON turns(updated_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func (s *Store) UpsertProject(ctx context.Context, project Project) error {
	now := project.UpdatedAt.UTC().Format(time.RFC3339Nano)
	created := project.CreatedAt.UTC().Format(time.RFC3339Nano)
	enabled := 0
	if project.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO projects (project_root, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(project_root) DO UPDATE SET
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`, project.Root, enabled, created, now)
	return err
}

func (s *Store) SetAdapter(ctx context.Context, projectRoot string, agent string, enabled bool, at time.Time) error {
	value := 0
	if enabled {
		value = 1
	}
	ts := at.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO installed_adapters (project_root, agent, enabled, installed_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(project_root, agent) DO UPDATE SET
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`, projectRoot, agent, value, ts, ts)
	return err
}

func (s *Store) ProjectCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&count)
	return count, err
}
