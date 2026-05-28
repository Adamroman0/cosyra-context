# Cosyra Context

Cosyra Context is a minimal CLI for sharing bounded project context between AI
coding tools such as Claude Code, Codex, and OpenCode.

The CLI is intentionally project-scoped. `cosyra on` enables context sharing for
the current project, writes local state under `.cosyra/`, and installs
Cosyra-owned hooks for the selected tools. `.cosyra/` is ignored by Git by
default.

## Commands

```bash
cosyra on
cosyra off
cosyra status
```

Internal hook commands are installed by `cosyra on`:

```bash
cosyra hook start --agent claude
cosyra hook stop --agent claude
```

## Privacy

Cosyra does not store raw transcripts by default. Hook payloads are sanitized
before metadata or context checkpoints are written.

