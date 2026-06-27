---
name: banking-planning
description: 並列サイクル用 planner workflow for the bank-system Codex subagent `planner`. Use when planner must read repo state, human notes, reviewer outputs, and produce MVP/improvement candidates plus accepted scope under docs/ai/cycles/<cycle-id>/planner.md.
---

# Banking Planning

## 目的

`planner` に対応する repo-local skill として、バンクシステム開発で繰り返す「repo 現状、human notes、reviewer 群のレビュー出力から次の改善案と accepted scope を作る」作業を一定品質で実行する。planner は企画と採択判断の責務を統合し、並列 spawn される `implementer` と reviewer 群に対して Markdown 成果物で同期する。

## 最初に読むもの

1. `git status --short` を確認し、ユーザー作業や他 agent 作業を壊さない。
2. `docs/START_HERE.md`、関連する `docs/*.md`、存在する場合は `README.md` と `docs/ai/output/human/*.md` を必要最小限読む。
3. `docs/ai/cycles/` 配下の過去 cycle を確認し、reviewer 出力、implementer 出力、未解決事項を把握する。
4. 既存コード、API/handler、service/usecase、repository、DB schema/migration、テストを確認し、実装済み機能と未実装領域を分ける。
5. `TODO`、`FIXME`、未使用の設計メモ、テスト不足、docs と実装の不一致を探索し、候補の根拠として使えるものを控える。
6. 詳細な金融品質観点が必要なら `references/banking-quality-rubric.md` を読む。

## ワークフロー

1. 同一 cycle の出力先 `docs/ai/cycles/<cycle-id>/planner.md` を決める。ユーザー指定があればそれを使い、指定がなければ当日の日付と連番を使う。
2. repo 現状を、実装済み、設計済みだが未実装、未設計、docs/実装不一致、レビュー未反映に分けて把握する。
3. 学習用ミニバンキングシステムの範囲内か確認する。本番金融システム相当の断定は避ける。
4. 候補ごとに、repo 上の根拠、現在の不足、MVP に入れる理由、reviewer 観点、実装時の注意を明示する。
5. 改善案を採択、却下、保留に分類する。保留や人間確認事項を accepted scope に混ぜない。
6. accepted scope は implementer が判断を追加せずに分解できる粒度にし、対象、非対象、テスト方針を明示する。
7. 金額、残高、取引履歴、監査ログ、冪等性、認証認可、DBトランザクション境界への影響を確認する。
8. 安全上重要な仕様、プロダクト方針、後戻りしにくい金融ドメイン判断は「人間確認事項」として分離する。
9. 日本語で、根拠と不確実性を明示して出力する。

## 出力契約

出力ファイルは `docs/ai/cycles/<cycle-id>/planner.md` とする。

出力は次の区分を使う: repo現状、入力レビュー、改善候補、採択、却下、保留、accepted scope、実装しないこと、人間確認事項。

`accepted scope` には最低限、目的、対象ファイル/領域、実装対象、実装しないこと、テスト方針、レビューで重点確認してほしい観点を含める。

## 禁止事項

- 公開 skill / agent の本文をコピーしない。
- repo 確認なしに一般論だけで候補を出さない。
- 実装、ソースコード変更、DBスキーマ確定、認証方式や金融仕様の最終決定を行わない。
- 金融仕様やリスク受容を人間確認なしに最終確定しない。
- `docs/ai/cycles/<cycle-id>/planner.md` 以外へ書き込まない。
