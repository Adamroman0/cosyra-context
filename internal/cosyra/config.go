package cosyra

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DirName        = ".cosyra"
	ConfigFileName = "config.json"
	ContextFile    = "context.md"
)

type Config struct {
	Version      int       `json:"version"`
	ProjectRoot  string    `json:"project_root"`
	Enabled      bool      `json:"enabled"`
	EnabledTools []string  `json:"enabled_tools"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ToolDetection struct {
	Name     string
	Detected bool
	Reason   string
}

func ResolveProjectRoot(project string) (string, error) {
	if project != "" {
		return filepath.Abs(project)
	}
	if root, err := gitRoot(); err == nil && root != "" {
		return root, nil
	}
	return os.Getwd()
}

func EnableProject(projectRoot string, tools []string) (*Config, error) {
	if err := os.MkdirAll(filepath.Join(projectRoot, DirName), 0o700); err != nil {
		return nil, err
	}
	if err := ensureGitignore(projectRoot); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	cfg := &Config{
		Version:      1,
		ProjectRoot:  projectRoot,
		Enabled:      true,
		EnabledTools: normalizeTools(tools),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if existing, err := LoadConfig(projectRoot); err == nil {
		cfg.CreatedAt = existing.CreatedAt
	}
	if err := writeJSONAtomic(configPath(projectRoot), cfg, 0o600); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(projectRoot, DirName, ContextFile)); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(filepath.Join(projectRoot, DirName, ContextFile), []byte("# Cosyra Context\n\nNo context captured yet.\n"), 0o600); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func DisableProject(projectRoot string, purge bool) error {
	if purge {
		return os.RemoveAll(filepath.Join(projectRoot, DirName))
	}
	cfg, err := LoadConfig(projectRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	cfg.Enabled = false
	cfg.UpdatedAt = time.Now().UTC()
	return writeJSONAtomic(configPath(projectRoot), cfg, 0o600)
}

func LoadConfig(projectRoot string) (*Config, error) {
	data, err := os.ReadFile(configPath(projectRoot))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) ContextPath() string {
	return filepath.Join(c.ProjectRoot, DirName, ContextFile)
}

func (c *Config) ToolEnabled(tool string) bool {
	for _, enabled := range c.EnabledTools {
		if enabled == tool {
			return true
		}
	}
	return false
}

func configPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, ConfigFileName)
}

func gitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ensureGitignore(projectRoot string) error {
	path := filepath.Join(projectRoot, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == DirName+"/" {
			return nil
		}
	}
	next := string(data)
	if next != "" && !strings.HasSuffix(next, "\n") {
		next += "\n"
	}
	next += DirName + "/\n"
	return os.WriteFile(path, []byte(next), 0o644)
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func AllTools() []string {
	return []string{"claude", "codex", "opencode"}
}

func ValidTool(tool string) bool {
	for _, candidate := range AllTools() {
		if candidate == tool {
			return true
		}
	}
	return false
}

func ToolDisplayName(tool string) string {
	switch tool {
	case "claude":
		return "Claude Code"
	case "codex":
		return "Codex"
	case "opencode":
		return "OpenCode"
	default:
		return tool
	}
}

func DetectTools() []ToolDetection {
	home, _ := os.UserHomeDir()
	checks := map[string][]string{
		"claude":   {"claude", filepath.Join(home, ".claude")},
		"codex":    {"codex", filepath.Join(home, ".codex")},
		"opencode": {"opencode", filepath.Join(home, ".config", "opencode"), filepath.Join(home, ".local", "share", "opencode")},
	}

	var out []ToolDetection
	for _, tool := range AllTools() {
		detection := ToolDetection{Name: tool}
		for i, candidate := range checks[tool] {
			if i == 0 {
				if _, err := exec.LookPath(candidate); err == nil {
					detection.Detected = true
					detection.Reason = "binary found"
					break
				}
				continue
			}
			if _, err := os.Stat(candidate); err == nil {
				detection.Detected = true
				detection.Reason = fmt.Sprintf("config found at %s", candidate)
				break
			}
		}
		out = append(out, detection)
	}
	return out
}

func normalizeTools(tools []string) []string {
	var out []string
	for _, tool := range tools {
		tool = strings.ToLower(strings.TrimSpace(tool))
		if tool == "" || !ValidTool(tool) {
			continue
		}
		out = append(out, tool)
	}
	sort.Strings(out)
	deduped := out[:0]
	var previous string
	for _, tool := range out {
		if tool == previous {
			continue
		}
		deduped = append(deduped, tool)
		previous = tool
	}
	return deduped
}
