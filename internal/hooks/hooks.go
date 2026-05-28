package hooks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adamroman0/cosyra-context/internal/checkpoint"
	"github.com/adamroman0/cosyra-context/internal/cosyra"
	"github.com/adamroman0/cosyra-context/internal/privacy"
	"github.com/adamroman0/cosyra-context/internal/store"
)

type StartResult struct {
	Context string
}

type StopResult struct {
	TurnKey string
}

func HandleStart(projectRoot string, agent string, input io.Reader) (*StartResult, error) {
	if !cosyra.ValidTool(agent) {
		return nil, fmt.Errorf("unknown agent %q", agent)
	}
	cfg, err := cosyra.LoadConfig(projectRoot)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || !cfg.ToolEnabled(agent) {
		return &StartResult{}, nil
	}

	payload := readPayload(input)
	now := time.Now().UTC()
	agentSessionID := firstNonEmpty(extractString(payload, "session_id"), extractString(payload, "sessionId"), os.Getenv("COSYRA_AGENT_SESSION_ID"), randomID("agent"))
	cosyraSessionID := firstNonEmpty(os.Getenv("COSYRA_SESSION_ID"), randomID("cosyra"))

	db, err := store.Open(context.Background(), filepath.Join(projectRoot, cosyra.DirName))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := db.UpsertAgentSession(context.Background(), store.AgentSession{
		ProjectRoot:     projectRoot,
		CosyraSessionID: cosyraSessionID,
		Agent:           agent,
		AgentSessionID:  agentSessionID,
		Status:          "active",
		StartedAt:       now,
		UpdatedAt:       now,
		EndedAt:         nil,
	}); err != nil {
		return nil, err
	}

	contextBytes, _ := os.ReadFile(filepath.Join(projectRoot, cosyra.DirName, cosyra.ContextFile))
	return &StartResult{Context: string(contextBytes)}, nil
}

func HandleStop(projectRoot string, agent string, input io.Reader) (*StopResult, error) {
	if !cosyra.ValidTool(agent) {
		return nil, fmt.Errorf("unknown agent %q", agent)
	}
	cfg, err := cosyra.LoadConfig(projectRoot)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || !cfg.ToolEnabled(agent) {
		return &StopResult{}, nil
	}

	raw, payload := readRawPayload(input)
	now := time.Now().UTC()
	agentSessionID := firstNonEmpty(extractString(payload, "session_id"), extractString(payload, "sessionId"), os.Getenv("COSYRA_AGENT_SESSION_ID"), "unknown")
	cosyraSessionID := firstNonEmpty(os.Getenv("COSYRA_SESSION_ID"), randomID("cosyra"))
	turnKey := firstNonEmpty(extractString(payload, "turn_id"), extractString(payload, "turnId"), extractString(payload, "uuid"), hashPayload(raw))
	summary := privacy.Sanitize(extractSummary(payload))

	db, err := store.Open(context.Background(), filepath.Join(projectRoot, cosyra.DirName))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := db.UpsertAgentSession(context.Background(), store.AgentSession{
		ProjectRoot:     projectRoot,
		CosyraSessionID: cosyraSessionID,
		Agent:           agent,
		AgentSessionID:  agentSessionID,
		Status:          "idle",
		StartedAt:       now,
		UpdatedAt:       now,
		EndedAt:         nil,
	}); err != nil {
		return nil, err
	}
	if err := db.UpsertTurn(context.Background(), store.Turn{
		ProjectRoot:    projectRoot,
		Agent:          agent,
		AgentSessionID: agentSessionID,
		TurnKey:        turnKey,
		Status:         "completed",
		StartedAt:      nil,
		CompletedAt:    &now,
		Summary:        summary,
		MetadataJSON:   "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return nil, err
	}
	if err := checkpoint.Update(context.Background(), db, checkpoint.UpdateInput{
		ProjectRoot:    projectRoot,
		Agent:          agent,
		AgentSessionID: agentSessionID,
		TurnKey:        turnKey,
		Summary:        summary,
		UpdatedAt:      now,
	}); err != nil {
		return nil, err
	}
	return &StopResult{TurnKey: turnKey}, nil
}

func readPayload(input io.Reader) map[string]any {
	_, payload := readRawPayload(input)
	return payload
}

func readRawPayload(input io.Reader) ([]byte, map[string]any) {
	raw, err := io.ReadAll(io.LimitReader(input, 2<<20))
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return nil, map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw, map[string]any{}
	}
	return raw, payload
}

func extractString(payload map[string]any, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}

func extractSummary(payload map[string]any) string {
	for _, key := range []string{"last_agent_message", "last_assistant_message", "summary", "message"} {
		if value := extractString(payload, key); value != "" {
			return truncate(value, 2000)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func randomID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func hashPayload(raw []byte) string {
	if len(raw) == 0 {
		return randomID("turn")
	}
	sum := sha256.Sum256(raw)
	return "sha256_" + hex.EncodeToString(sum[:8])
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
