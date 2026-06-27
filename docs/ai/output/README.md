# AI agent output

Codex subagents can store human notes under this directory. Active subagent cycle outputs are stored under `docs/ai/cycles/`.

## Human notes

Human advice and direction should be stored as Markdown under:

```text
docs/ai/output/human/
```

Each subagent must check this directory when it exists before starting work.

## Cycle outputs

Use `docs/ai/cycles/<cycle-id>/` for active planner/implementer/reviewer outputs. See `docs/ai/cycles/README.md`.
