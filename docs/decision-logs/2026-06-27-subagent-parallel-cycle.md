# 意思決定ログ: 2026-06-27-subagent-parallel-cycle

## 概要

- 日付: 2026-06-27
- 対象範囲: Codex subagent 構成と開発サイクル
- 作成者/agent: Codex

## 入力

- 人間の方針: `planner`, `implementer`, reviewer 系を spawn して作業させる運用を繰り返したい。
- 人間の方針: reviewer 系は複数存在し、専門性を分けて repo 全体をレビューする。
- 人間の方針: subagent は並列起動を前提にしたい。

## 判断結果

### 採択

| 改善案 | 採択理由 | accepted scope | 実装しないこと |
| --- | --- | --- | --- |
| subagent を `planner`, `implementer`, reviewer 群の 3 ロールへ統合する | 運用サイクルを単純化しつつ、reviewer の専門性は維持できるため | `.codex/agents` を `planner`, `implementer`, `code-reviewer`, `security-reviewer`, `banking-reviewer` に整理する | reviewer を単一 agent に統合しない |
| cycle artifact protocol を導入する | 並列 spawn では runtime 同期に頼ると順序依存が破綻しやすいため | `docs/ai/cycles/<cycle-id>/` に planner/implementer/reviewer の成果物を置く | agent 間の直接同期や暗黙の状態共有 |
| `implementer` は accepted scope 不在時に blocked を記録する | 並列起動時に planner 出力より先に implementer が走っても誤実装を防ぐため | `blocked: accepted scope not found` を `implementer.md` に記録する | accepted scope なしの実装 |

### 却下

| 改善案 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| reviewer を 1 agent に統合する | 専門性を分けて repo 全体レビューしたいという人間方針に合わないため | reviewer 並列運用の負荷が専門性の利点を上回る場合 |
| 提案統合専用の独立 board を残す | planner が採択/却下/保留と accepted scope 作成まで担う方が cycle が短くなるため | planner の判断負荷が高くなり、採択品質が落ちる場合 |

### 保留

| 改善案 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| cycle-id の自動採番スクリプトを追加する | まずは Markdown protocol だけで運用できるため | 手動採番が負担になるか | 運用後に必要なら追加 |

## リスクと緩和策

- リスク: 並列起動時、`implementer` が `planner.md` 作成前に実行される。
- 緩和策: accepted scope がなければ実装せず blocked を記録する。
- リスク: reviewer 出力が分散し、planner が見落とす。
- 緩和策: 各 reviewer の必須項目に「次サイクル planner への入力」を入れる。

## 関連リンク/ファイル

- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `.agents/skills/banking-planning/SKILL.md`
