# AI agent output

Codex subagents write their Markdown outputs under this directory.

## Output path

Use the following path format:

```text
docs/ai/output/[agent-name]/001-[title].md
```

- `[agent-name]` must match the subagent name, such as `product-planner` or `code-reviewer`.
- The number is a zero-padded sequence. Check existing files in the same agent directory and use the next number.
- `[title]` should be short, ASCII-friendly, and describe the topic.

## Human notes

Human advice and direction should be stored as Markdown under:

```text
docs/ai/output/human/
```

Each subagent must check this directory when it exists before starting work.

## Review feedback loop

Proposal and decision agents must check existing outputs from:

- `docs/ai/output/code-reviewer/`
- `docs/ai/output/security-reviewer/`
- `docs/ai/output/banking-reviewer/`

When those review outputs exist, proposal and decision agents must state what they reflected and what they did not reflect.
