package checkpoint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adamroman0/cosyra-context/internal/cosyra"
	"github.com/adamroman0/cosyra-context/internal/privacy"
	"github.com/adamroman0/cosyra-context/internal/store"
)

const DefaultMaxContextBytes = 12 * 1024

type UpdateInput struct {
	ProjectRoot    string
	Agent          string
	AgentSessionID string
	TurnKey        string
	Summary        string
	UpdatedAt      time.Time
}

func Update(ctx context.Context, db *store.Store, input UpdateInput) error {
	contextText := buildContext(input, readExistingActivity(input.ProjectRoot))
	contextText = boundContext(contextText, DefaultMaxContextBytes)

	path := filepath.Join(input.ProjectRoot, cosyra.DirName, cosyra.ContextFile)
	if err := writeAtomic(path, []byte(contextText), 0o600); err != nil {
		return err
	}
	return db.UpsertCheckpoint(ctx, store.Checkpoint{
		ProjectRoot:        input.ProjectRoot,
		Context:            contextText,
		ContextBytes:       len([]byte(contextText)),
		LastAgent:          input.Agent,
		LastAgentSessionID: input.AgentSessionID,
		UpdatedAt:          input.UpdatedAt,
	})
}

func buildContext(input UpdateInput, previousActivity []string) string {
	summary := strings.TrimSpace(privacy.Sanitize(input.Summary))
	if summary == "" {
		summary = "Turn completed."
	}
	line := fmt.Sprintf("- %s %s `%s` `%s`: %s",
		input.UpdatedAt.UTC().Format(time.RFC3339),
		input.Agent,
		input.AgentSessionID,
		input.TurnKey,
		oneLine(summary))

	activity := append([]string{line}, previousActivity...)
	if len(activity) > 25 {
		activity = activity[:25]
	}

	return fmt.Sprintf(`# Cosyra Context

Last updated: %s
Last agent: %s
Last session: %s

## Active Thread
No explicit active thread captured yet.

## Durable Facts
No durable facts captured yet.

## Decisions
No decisions captured yet.

## Latest Handoff
%s

## Open Items
No open items captured yet.

## Recent Relevant Activity
%s
`, input.UpdatedAt.UTC().Format(time.RFC3339), input.Agent, input.AgentSessionID, summary, strings.Join(activity, "\n"))
}

func readExistingActivity(projectRoot string) []string {
	data, err := os.ReadFile(filepath.Join(projectRoot, cosyra.DirName, cosyra.ContextFile))
	if err != nil {
		return nil
	}
	text := string(data)
	marker := "## Recent Relevant Activity\n"
	index := strings.Index(text, marker)
	if index < 0 {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(text[index+len(marker):], "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			lines = append(lines, line)
		}
	}
	return lines
}

func boundContext(contextText string, maxBytes int) string {
	if len([]byte(contextText)) <= maxBytes {
		return contextText
	}
	lines := strings.Split(contextText, "\n")
	for len(lines) > 0 && len([]byte(strings.Join(lines, "\n"))) > maxBytes {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func oneLine(input string) string {
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.Join(strings.Fields(input), " ")
	if len(input) > 240 {
		return input[:240]
	}
	return input
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
