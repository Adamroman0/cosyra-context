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

type AgentSession struct {
	ProjectRoot     string
	CosyraSessionID string
	Agent           string
	AgentSessionID  string
	Status          string
	StartedAt       time.Time
	UpdatedAt       time.Time
	EndedAt         *time.Time
}

type Turn struct {
	ProjectRoot    string
	Agent          string
	AgentSessionID string
	TurnKey        string
	Status         string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Summary        string
	MetadataJSON   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Checkpoint struct {
	ProjectRoot        string
	Context            string
	ContextBytes       int
	LastAgent          string
	LastAgentSessionID string
	UpdatedAt          time.Time
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

func (s *Store) UpsertAgentSession(ctx context.Context, session AgentSession) error {
	endedAt := sql.NullString{}
	if session.EndedAt != nil {
		endedAt.Valid = true
		endedAt.String = session.EndedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_sessions (
			project_root, cosyra_session_id, agent, agent_session_id, status,
			started_at, updated_at, ended_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_root, agent, agent_session_id) DO UPDATE SET
			cosyra_session_id = excluded.cosyra_session_id,
			status = excluded.status,
			updated_at = excluded.updated_at,
			ended_at = excluded.ended_at
	`, session.ProjectRoot, session.CosyraSessionID, session.Agent, session.AgentSessionID, session.Status,
		session.StartedAt.UTC().Format(time.RFC3339Nano),
		session.UpdatedAt.UTC().Format(time.RFC3339Nano),
		endedAt)
	return err
}

func (s *Store) UpsertTurn(ctx context.Context, turn Turn) error {
	startedAt := sql.NullString{}
	if turn.StartedAt != nil {
		startedAt.Valid = true
		startedAt.String = turn.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	completedAt := sql.NullString{}
	if turn.CompletedAt != nil {
		completedAt.Valid = true
		completedAt.String = turn.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	metadata := turn.MetadataJSON
	if metadata == "" {
		metadata = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO turns (
			project_root, agent, agent_session_id, turn_key, status,
			started_at, completed_at, summary, metadata_json, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_root, agent, agent_session_id, turn_key) DO UPDATE SET
			status = excluded.status,
			completed_at = excluded.completed_at,
			summary = excluded.summary,
			metadata_json = excluded.metadata_json,
			updated_at = excluded.updated_at
	`, turn.ProjectRoot, turn.Agent, turn.AgentSessionID, turn.TurnKey, turn.Status,
		startedAt, completedAt, turn.Summary, metadata,
		turn.CreatedAt.UTC().Format(time.RFC3339Nano),
		turn.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) UpsertCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO checkpoints (
			project_root, context, context_bytes, last_agent, last_agent_session_id, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_root) DO UPDATE SET
			context = excluded.context,
			context_bytes = excluded.context_bytes,
			last_agent = excluded.last_agent,
			last_agent_session_id = excluded.last_agent_session_id,
			updated_at = excluded.updated_at
	`, checkpoint.ProjectRoot, checkpoint.Context, checkpoint.ContextBytes, checkpoint.LastAgent,
		checkpoint.LastAgentSessionID, checkpoint.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ProjectCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&count)
	return count, err
}

func (s *Store) TurnCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM turns`).Scan(&count)
	return count, err
}

func (s *Store) PruneBefore(ctx context.Context, cutoff time.Time) error {
	cutoffText := cutoff.UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM turns WHERE updated_at < ?`, cutoffText); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM agent_sessions
		WHERE updated_at < ?
		  AND status NOT IN ('active', 'busy', 'waiting')
	`, cutoffText)
	return err
}

func DBSize(projectRoot string) (int64, error) {
	path := filepath.Join(projectRoot, ".cosyra", DBFileName)
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
