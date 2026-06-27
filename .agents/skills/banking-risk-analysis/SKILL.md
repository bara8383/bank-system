---
name: banking-risk-analysis
description: 金融事故・運用リスク分析 for the bank-system Codex subagent `risk-analyst`. Use when that subagent or a user needs this repository-specific workflow for Go/PostgreSQL mini banking work, including scope control, auditability, idempotency, transaction integrity, and Japanese review outputs.
---

# Banking Risk Analysis

## 目的

`risk-analyst` に対応する repo-local skill として、バンクシステム開発で繰り返す「金融事故・運用リスク分析」を一定品質で実行する。公開されている高評価の skill / agent 事例からは、本文のコピーではなく、狭い責務、明確な trigger、progressive disclosure、出力契約、権限分離の設計パターンだけを取り入れる。

## 最初に読むもの

1. `git status --short` を確認し、ユーザー作業や他 agent 作業を壊さない。
2. `docs/START_HERE.md`、関連する `docs/*.md`、存在する場合は `docs/ai/output/human/*.md` を必要最小限読む。
3. 存在する場合は `docs/ai/output/code-reviewer/`、`docs/ai/output/security-reviewer/`、`docs/ai/output/banking-reviewer/` のレビュー出力を必ず読む。
4. 詳細な金融品質観点が必要なら `references/banking-quality-rubric.md` を読む。

## ワークフロー

1. 依頼内容が `risk-analyst` の責務と一致するか確認する。違う場合は、適切な agent / skill へ渡す提案をする。
2. 学習用ミニバンキングシステムの範囲内か確認する。本番金融システム相当の断定は避ける。
3. 金額、残高、取引履歴、監査ログ、冪等性、認証認可、DBトランザクション境界への影響を確認する。
4. 安全上重要な仕様、プロダクト方針、後戻りしにくい金融ドメイン判断は「人間確認事項」として分離する。
5. 日本語で、根拠と不確実性を明示して出力する。
6. レビュー出力を参照した場合は、反映した点と反映しなかった点を明示する。

## 出力契約

出力ファイルは `docs/ai/output/risk-analyst/001-[title].md` 形式で作成する。連番は既存ファイルを確認して次の番号にする。

出力は原則として次の区分を使う: リスク、発生条件、影響、緩和策、残余リスク、人間確認事項。

## 禁止事項

- 公開 skill / agent の本文をコピーしない。
- 金融仕様やリスク受容を人間確認なしに最終確定しない。
- 役割外の実装、レビュー、採択判断を兼務しない。
- accepted scope 外の隣接改善を勝手に追加しない。
- `docs/ai/output/risk-analyst/` 以外へ書き込まない。
