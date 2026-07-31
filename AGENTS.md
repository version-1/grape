# AGENTS.md

All documentation must be written in English.

## Project Overview

`grape` is a Git wrapper designed for use by coding agents, with a focus on agent-friendly `git worktree` workflows. Its entry point is `cmd/grape`; core responsibilities are separated into `internal/app`, `internal/config`, `internal/ui`, and `internal/worktree`.

See `README.md` and `docs/references/` for user-facing documentation.

## Development Rules

- Use Go 1.22.
- Preserve existing package responsibilities: CLI control belongs in `internal/app`, and Git operations belong in `internal/worktree`.
- When changing public CLI arguments, exit codes, or the configuration format, verify compatibility and update documentation.
- `remove` and `reset` delete worktrees and local branches. Do not weaken target selection or confirmation safeguards.

## Common Commands

```sh
# Run all tests
go test ./...

# Build a local binary
make build

# Run the CLI directly
go run ./cmd/grape list
```

After making changes, run tests for the affected package and `go test ./...`.

## Change Checklist

- Format Go code with `gofmt`.
- When CLI behavior changes, update its tests, `README.md`, and `docs/references/`.
- For deletion-related changes, confirm that the main working tree and the branch checked out there remain excluded.
