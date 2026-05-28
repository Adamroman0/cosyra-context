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

`cosyra on` resolves the current Git root, confirms the project path, shows a
tool checklist, creates `.cosyra/`, and adds `.cosyra/` to `.gitignore`.

Internal hook commands are installed by `cosyra on`:

```bash
cosyra hook start --agent claude
cosyra hook stop --agent claude
```

## Privacy

Cosyra does not store raw transcripts by default. Hook payloads are sanitized
before metadata or context checkpoints are written.

Local state:

```text
.cosyra/
  config.json
  context.md
  cosyra.db
  hooks/
```

Retention defaults:

- `context.md` is bounded to roughly 12 KB.
- Event metadata is retained for at most 7 days.
- Raw transcript capture is not implemented in the MVP.

## Development

```bash
go test ./...
go build -o dist/cosyra ./cmd/cosyra
```

The npm package is a thin wrapper around prebuilt binaries under `dist/`.
Homebrew packaging lives under `packaging/homebrew/`.
