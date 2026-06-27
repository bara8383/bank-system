---
name: scoped-banking-implementation
description: 採択済み scope のみを安全に実装 for the bank-system Codex subagent `implementer`. Use when implementer must read docs/ai/cycles/<cycle-id>/planner.md and either implement accepted scope or record blocked: accepted scope not found.
---

# Scoped Banking Implementation

## 目的

`implementer` に対応する repo-local skill として、バンクシステム開発で繰り返す「planner の accepted scope のみを安全に実装」を一定品質で実行する。並列 spawn された場合でも、同一 cycle の `planner.md` に accepted scope がなければ実装せず、blocked 状態を成果物として残す。

## 最初に読むもの

1. `git status --short` を確認し、ユーザー作業や他 agent 作業を壊さない。
2. `docs/START_HERE.md`、関連する `docs/*.md`、存在する場合は `docs/ai/output/human/*.md` を必要最小限読む。
3. 同一 cycle の `docs/ai/cycles/<cycle-id>/planner.md` を読む。
4. `planner.md` に accepted scope が存在しない、または空の場合は実装しない。
5. 詳細な金融品質観点が必要なら `references/banking-quality-rubric.md` を読む。

## ワークフロー

1. 依頼内容が `implementer` の責務と一致するか確認する。違う場合は、適切な agent / skill へ渡す提案をする。
2. 同一 cycle の accepted scope を対象、非対象、テスト方針に分解する。
3. accepted scope がない場合は `docs/ai/cycles/<cycle-id>/implementer.md` に `blocked: accepted scope not found` と実装しなかった理由を書く。
4. 学習用ミニバンキングシステムの範囲内か確認する。本番金融システム相当の断定は避ける。
5. 金額、残高、取引履歴、監査ログ、冪等性、認証認可、DBトランザクション境界への影響を確認する。
6. 安全上重要な仕様、プロダクト方針、後戻りしにくい金融ドメイン判断は「人間確認事項」として分離する。
7. 日本語で、根拠と不確実性を明示して出力する。

## 出力契約

出力ファイルは `docs/ai/cycles/<cycle-id>/implementer.md` とする。

出力は原則として次の区分を使う: 参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項。

blocked の場合は次の区分を使う: 参照した accepted scope、blocked、実装しなかったこと、次に必要な入力。

## 禁止事項

- 公開 skill / agent の本文をコピーしない。
- 金融仕様やリスク受容を人間確認なしに最終確定しない。
- 役割外の実装、レビュー、採択判断を兼務しない。
- accepted scope 外の隣接改善を勝手に追加しない。
- accepted scope がない状態で実装しない。
- 実装対象以外の書き込みは `docs/ai/cycles/<cycle-id>/implementer.md` に限定する。
