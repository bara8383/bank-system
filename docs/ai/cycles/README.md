# AI cycle artifact protocol

Each development cycle uses one directory:

```text
docs/ai/cycles/<cycle-id>/
```

Use a user-provided `<cycle-id>` when available. Otherwise use a date and sequence such as `2026-06-27-001`.

## Files

- `planner.md`: repo現状、入力レビュー、改善候補、採択、却下、保留、accepted scope、実装しないこと、作業仮定。
- `implementer.md`: 参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定。
- `code-reviewer.md`: Finding、根拠、影響、推奨修正、次サイクル planner への入力。
- `security-reviewer.md`: Finding、根拠、影響、推奨修正、次サイクル planner への入力。
- `banking-reviewer.md`: Finding、根拠、影響、推奨修正、次サイクル planner への入力。

## Cycle rules

- Run each cycle sequentially as `planner` -> `implementer` -> reviewers.
- `planner` runs first and writes accepted scope for the same cycle.
- `implementer` implements the same-cycle accepted scope.
- Reviewer agents may run in parallel after `implementer` creates the implementation diff.
- Reviewers prefer implementation-diff review. Repo-wide review should be explicit or needed for the next planner input.
- Using the latest completed cycle's accepted scope is an exception for reruns or recovery only.
- If no accepted scope exists in the requested cycle or exception fallback cycle, `implementer` writes `blocked: accepted scope not found` to `implementer.md`.
- Agents coordinate through files in this directory, not through direct runtime synchronization.
- Agents do not stop for human-confirmation gates. Human judgment happens in PR review; agents record working assumptions in artifacts and diffs.
