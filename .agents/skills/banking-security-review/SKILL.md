---
name: banking-security-review
description: 金融系セキュリティレビュー for the bank-system Codex subagent `security-reviewer`. Use when reviewing repo-wide or implementation-diff security under docs/ai/cycles/<cycle-id>/security-reviewer.md.
---

# Banking Security Review

## 目的

`security-reviewer` に対応する repo-local skill として、バンクシステム開発で繰り返す「金融系セキュリティレビュー」を一定品質で実行する。repo 全体または直近実装差分をレビューし、次サイクル planner への入力を残す。

## 最初に読むもの

1. `git status --short` を確認し、ユーザー作業や他 agent 作業を壊さない。
2. `docs/START_HERE.md`、関連する `docs/*.md`、存在する場合は `docs/ai/output/human/*.md` を必要最小限読む。
3. `docs/ai/cycles/` 配下の同一 cycle 成果物を確認し、実装差分があれば差分レビューを優先する。実装差分がない repo 全体レビューは、ユーザーが明示した場合または次 cycle の planner 入力として必要な場合に行う。
4. 詳細な金融品質観点が必要なら `references/banking-quality-rubric.md` を読む。

## ワークフロー

1. 依頼内容が `security-reviewer` の責務と一致するか確認する。違う場合は、適切な agent / skill へ渡す提案をする。
2. 学習用ミニバンキングシステムの範囲内か確認する。本番金融システム相当の断定は避ける。
3. 金額、残高、取引履歴、監査ログ、冪等性、認証認可、DBトランザクション境界への影響を確認する。
4. 未確定仕様は停止条件ではなく作業仮定として扱い、その仮定がセキュリティ、監査性、権限境界に与えるリスクを指摘する。
5. 日本語で、根拠と不確実性を明示して出力する。

## 出力契約

出力ファイルは `docs/ai/cycles/<cycle-id>/security-reviewer.md` とする。

出力は原則として次の区分を使う: Finding、根拠、影響、推奨修正、次サイクル planner への入力。必要に応じて重大度、攻撃/事故シナリオ、作業仮定のリスクを含める。

## 禁止事項

- 公開 skill / agent の本文をコピーしない。
- 役割外の実装、レビュー、採択判断を兼務しない。
- accepted scope 外の隣接改善を勝手に追加しない。
- `docs/ai/cycles/<cycle-id>/security-reviewer.md` 以外へ書き込まない。
