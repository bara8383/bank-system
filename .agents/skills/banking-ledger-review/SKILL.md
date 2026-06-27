---
name: banking-ledger-review
description: 元帳・残高・取引履歴・ドメイン・金融事故リスクレビュー for the bank-system Codex subagent `banking-reviewer`. Use when reviewing repo-wide or implementation-diff banking correctness under docs/ai/cycles/<cycle-id>/banking-reviewer.md.
---

# Banking Ledger Review

## 目的

`banking-reviewer` に対応する repo-local skill として、バンクシステム開発で繰り返す「元帳・残高・取引履歴・ドメイン・金融事故リスクレビュー」を一定品質で実行する。ドメイン分析とリスク分析の観点を吸収し、repo 全体または直近実装差分をレビューする。

## 最初に読むもの

1. `git status --short` を確認し、ユーザー作業や他 agent 作業を壊さない。
2. `docs/START_HERE.md`、関連する `docs/*.md`、存在する場合は `docs/ai/output/human/*.md` を必要最小限読む。
3. `docs/ai/cycles/` 配下の同一 cycle 成果物を確認し、実装差分があれば差分レビュー、なければ repo 全体レビューを行う。
4. 詳細な金融品質観点が必要なら `references/banking-quality-rubric.md` を読む。

## ワークフロー

1. 依頼内容が `banking-reviewer` の責務と一致するか確認する。違う場合は、適切な agent / skill へ渡す提案をする。
2. 学習用ミニバンキングシステムの範囲内か確認する。本番金融システム相当の断定は避ける。
3. 口座、残高、元帳、取引、取消、組戻し、監査、状態遷移、業務用語の一貫性を確認する。
4. 二重送金、残高不整合、競合更新、監査ログ欠落、不正操作などの金融事故リスクを確認する。
5. 金額、残高、取引履歴、監査ログ、冪等性、認証認可、DBトランザクション境界への影響を確認する。
6. 安全上重要な仕様、プロダクト方針、後戻りしにくい金融ドメイン判断は「人間確認事項」として分離する。
7. 日本語で、根拠と不確実性を明示して出力する。

## 出力契約

出力ファイルは `docs/ai/cycles/<cycle-id>/banking-reviewer.md` とする。

出力は原則として次の区分を使う: Finding、根拠、影響、推奨修正、次サイクル planner への入力。必要に応じて事故シナリオ、人間確認事項を含める。

## 禁止事項

- 公開 skill / agent の本文をコピーしない。
- 金融仕様やリスク受容を人間確認なしに最終確定しない。
- 役割外の実装、レビュー、採択判断を兼務しない。
- accepted scope 外の隣接改善を勝手に追加しない。
- `docs/ai/cycles/<cycle-id>/banking-reviewer.md` 以外へ書き込まない。
