# AI cycle artifact protocol

Each development cycle uses one directory:

```text
docs/ai/cycles/<cycle-id>/
```

Use a user-provided `<cycle-id>` when available. Otherwise use a date and sequence such as `2026-06-27-001`.

## Files

- `planner.md`: repo現状、入力レビュー、改善候補、採択、却下、保留、accepted scope、実装しないこと、人間確認事項。
- `implementer.md`: 参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項。
- `code-reviewer.md`: Finding、根拠、影響、推奨修正、次サイクル planner への入力。
- `security-reviewer.md`: Finding、根拠、影響、推奨修正、次サイクル planner への入力。
- `banking-reviewer.md`: Finding、根拠、影響、推奨修正、次サイクル planner への入力。

## Parallel rules

- `planner` is always runnable.
- `implementer` must not implement without an accepted scope in the same cycle's `planner.md`.
- If accepted scope is missing, `implementer` writes `blocked: accepted scope not found` to `implementer.md`.
- Reviewers prefer implementation-diff review when there is a diff; otherwise they perform repo-wide review.
- Agents coordinate through files in this directory, not through direct runtime synchronization.
