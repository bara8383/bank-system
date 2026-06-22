# Codex Subagents Development Loop

This directory defines a small, project-local Codex Subagents setup for an iterative product-development loop. The agents are intentionally limited to four standing roles so the process stays understandable, reviewable, and controlled by a human decision-maker.

## Design references

This setup follows the current Codex custom-agent shape: one TOML file per agent with `name`, `description`, and `developer_instructions`, plus optional fields such as `model_reasoning_effort`, `sandbox_mode`, and `nickname_candidates`. The role design also borrows proven patterns from community agent and skill libraries: explicit delegation, narrow responsibility boundaries, read-only review agents, structured outputs, quality gates, and human approval before implementation.

The result is not a copy of any external agent pack. It is a minimal workflow tailored to this repository.

## Purpose

The goal is to make Codex useful across the full improvement loop without allowing it to silently make large product or architecture decisions. Codex should help propose, implement, review, and package changes, while the human keeps authority over what is adopted and whether the product outcome is acceptable.

## Human and Codex responsibilities

### Human responsibilities

1. Decide whether each proposed improvement is adopted, rejected, or deferred.
2. Review the implemented product directly and judge whether the outcome is acceptable.
3. Provide product feedback that can seed the next proposal cycle.

### Codex responsibilities

1. Propose clear improvement options with tradeoffs.
2. Convert accepted decisions into implementation-ready scope.
3. Implement only accepted options.
4. Review implementation quality before human product review.
5. Package the change so the human can review it efficiently.
6. Turn human review feedback into the next set of improvement options.

## Development flow

1. Codex: `improvement-proposer` produces improvement proposals.
2. Human: accepts, rejects, or defers each proposal.
3. Codex: `implementer` implements only the accepted proposals.
4. Codex: `quality-reviewer` reviews the implementation for material risks.
5. Codex: `product-review-packager` prepares a human review package.
6. Human: uses the product and reviews the outcome.
7. Codex: `improvement-proposer` uses the human review to propose the next improvements.
8. Repeat from step 1.

## Agent responsibilities

### `improvement-proposer`

Use this agent when the loop needs possible next work. It analyzes the current state, recent review feedback, and unresolved issues, then returns A/B/C/Hold-style options. Each option should include purpose, expected effect, cost, risk, and adoption decision points.

This agent must not implement anything. It should require a human decision before work proceeds.

### `implementer`

Use this agent only after the human explicitly accepts one or more proposals. It implements the accepted scope with minimal, safe changes and avoids unrelated refactoring or opportunistic cleanup.

This agent must not implement rejected or deferred options. If the accepted scope is ambiguous, it should ask for clarification.

### `quality-reviewer`

Use this agent after implementation and before human product review. It checks for security issues, bugs, regressions, test gaps, maintainability problems, unintended behavior, and plausible performance concerns. Architecture risk is reviewed only when the change affects boundaries, data flow, or coupling.

This agent is read-only. It may recommend fixes but must not edit code.

### `product-review-packager`

Use this agent after implementation and quality review. It turns the technical change into a practical review guide for the human product reviewer: what changed, where to look, how to verify it, known issues, and decision points.

This agent is read-only. It must not make implementation or design changes.

## When to spawn each agent

- Spawn `improvement-proposer` at the start of a cycle, after human feedback, or when the team needs scoped options.
- Spawn `implementer` only after the human has explicitly listed accepted options.
- Spawn `quality-reviewer` once implementation is complete and before the human product review.
- Spawn `product-review-packager` after quality review findings are resolved or intentionally accepted.

Do not spawn agents automatically just because a task is large. Use explicit delegation prompts and keep each agent focused on its role.

## Prompt examples

### Example 1: propose improvements

```text
Spawn improvement-proposer.
Analyze the current repository state, recent changes, and known product direction.
Do not implement anything.
Return improvement proposals as A/B/C options with impact, cost, risk, and decision points.
Wait for my decision.
```

### Example 2: implement only accepted options

```text
Spawn implementer.
Implement only the options I explicitly accepted:
- A: <採用内容>
- C: <採用内容>
Do not implement rejected or deferred options.
Keep changes minimal.
After implementation, summarize changed files and validation commands.
```

### Example 3: quality review

```text
Spawn quality-reviewer.
Review the latest implementation before human product review.
Focus on security, bugs, regressions, tests, maintainability, unintended behavior, and performance.
Do not edit code.
Return findings by severity with evidence and recommended fixes.
```

### Example 4: package for human review

```text
Spawn product-review-packager.
Prepare a review package for a human product reviewer.
Include what changed, where to look, how to verify, known issues, screenshots/URLs if available, and decision points.
Do not edit code.
```

## Notes and cautions

- Human approval is the gate between proposal and implementation.
- Keep accepted work small enough to review safely.
- Treat rejected and deferred options as out of scope until the human changes the decision.
- Review agents should prefer material issues over style-only feedback.
- If the product cannot be run locally, the review package should clearly mark environment limitations and provide the closest available validation steps.
- If an agent discovers that the chosen scope is unsafe or unclear, it should stop and ask for a human decision instead of expanding scope.
