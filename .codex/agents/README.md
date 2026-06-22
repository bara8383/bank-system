# バンクシステム開発用 Codex Subagents

このディレクトリには、Go + PostgreSQL を中心にした銀行/金融トランザクション学習用バンクシステムのための Codex Subagent 定義を配置します。agent 本文はすべて日本語で記述し、TOML の key 名のみ Codex 仕様に合わせて英語のままにします。

## 参照した公開情報と取り入れ方

この構成は、公開されている agent 本文をコピーせず、以下の情報から設計パターンだけを抽象化してバンクシステム用に再設計したものです。

| 参照元 | ライセンス/位置づけ | 取り入れた設計パターン |
| --- | --- | --- |
| [OpenAI Codex 公式 Subagents ドキュメント](https://developers.openai.com/codex/subagents) | 公式仕様 | `.codex/agents/*.toml` に project-scoped agent を置くこと、必須 key を `name` / `description` / `developer_instructions` とすること、必要に応じて `model_reasoning_effort` / `sandbox_mode` / `nickname_candidates` を使うこと。 |
| [VoltAgent/awesome-codex-subagents](https://github.com/VoltAgent/awesome-codex-subagents) | MIT License | Codex-native TOML、狭い責務の agent 分割、reviewer/auditor は `read-only`、developer/engineer は `workspace-write` に分ける考え方。 |
| [wshobson/agents](https://github.com/wshobson/agents) | MIT License | 大きな workflow を単一 agent に詰め込まず、planner、architect、security、review、orchestrator、reporter などに分割する構成、ハーネスごとのネイティブ形式を尊重する考え方。 |
| その他の実運用向け multi-agent 事例 | 参考情報 | 明示的な delegation、実装前の scope 確認、レビュー結果の重大度分類、最終的な人間レビュー前提の report を残す運用。 |

取り入れたのは構造、責務分離、禁止事項、出力形式、sandbox 権限の分け方のみです。各 agent の本文は、このバンクシステムの既存ドキュメントと追加要件に合わせて日本語で再設計しています。

## Codex TOML 方針

- 各 agent は `.codex/agents/<agent-name>.toml` に 1 ファイル 1 agent として配置します。
- 各 agent は `name`, `description`, `developer_instructions` を必ず持ちます。
- `model_reasoning_effort` は、深いレビューや金融リスク判断が必要な agent では `high`、軽量な整理や報告では `medium` を使います。
- `sandbox_mode` は、提案・分析・レビューだけを行う agent では `read-only`、意思決定ログや実装/最終報告ファイルを作る agent では `workspace-write` を使います。
- `nickname_candidates` は Codex 表示名として扱いやすい ASCII 名にしています。agent 本文は日本語です。

## 各 agent の役割

| Agent | 役割 | sandbox_mode | 実装可否 |
| --- | --- | --- | --- |
| `product-planner` | 口座、入出金、振込、取引履歴、残高照会、監査ログなどの機能案、改善案、MVP案を整理する。 | `read-only` | 不可 |
| `domain-analyst` | 口座、残高、元帳、取引、取消、組戻し、監査など、銀行ドメインとして自然かを評価する。 | `read-only` | 不可 |
| `risk-analyst` | 二重送金、残高不整合、競合更新、監査ログ欠落、不正操作などの金融事故リスクを洗い出す。 | `read-only` | 不可 |
| `architecture-analyst` | Go、PostgreSQL、トランザクション設計、レイヤード/クリーンアーキテクチャ、API設計の観点で改善案を出す。 | `read-only` | 不可 |
| `ai-review-board` | 各提案を統合し、改善案を「採択」「却下」「保留」に分類し、実装用の accepted scope と意思決定ログを作る。 | `workspace-write` | 不可 |
| `implementer` | `ai-review-board` が採択した accepted scope だけを小さく安全に実装する。 | `workspace-write` | 可 |
| `code-reviewer` | 実装後のコード品質、責務分離、テスト不足、保守性をレビューする。 | `read-only` | 不可 |
| `security-reviewer` | 認証、認可、入力検証、ログ、秘密情報、SQL injection、権限境界をレビューする。 | `read-only` | 不可 |
| `banking-reviewer` | 残高更新、元帳、取引履歴、取消不能性、監査可能性、冪等性など銀行システムとしての妥当性をレビューする。 | `read-only` | 不可 |
| `final-report-writer` | 人間の最終レビュー用に変更内容、採択理由、リスク、未解決事項、確認ポイントを短く整理する。 | `workspace-write` | 不可 |

## Repo-local skills

OpenAI Codex の公式 Skills ドキュメントでは、skill は `SKILL.md` を持つディレクトリで、必要に応じて `references/`、`scripts/`、`assets/`、`agents/openai.yaml` を持てると説明されています。また、repo 共通で使う skill は `.agents/skills` に置けるため、このプロジェクトでは subagent と 1:1 で対応する repo-local skill を `.agents/skills/<skill-name>/` に配置します。

公開されている人気の agent / skill 事例からは、本文をコピーせず、次の設計パターンだけを採用します。

- 1 skill / 1 agent の狭い責務。
- `description` に具体的な trigger と境界を前方配置する progressive disclosure。
- planner / analyst / reviewer / implementer / reporter の責務分離。
- reviewer は読み取り専用、implementer は accepted scope のみ実装する権限分離。
- 金融リスク、セキュリティ、コード品質、元帳品質を別観点でレビューする分割。

| Agent | 対応 skill | 用途 |
| --- | --- | --- |
| `product-planner` | `.agents/skills/banking-product-planning` | MVP、機能候補、保留候補、人間確認事項を整理する。 |
| `domain-analyst` | `.agents/skills/banking-domain-analysis` | 銀行ドメイン用語、状態遷移、業務モデルの不整合を確認する。 |
| `risk-analyst` | `.agents/skills/banking-risk-analysis` | 二重送金、残高不整合、競合更新、監査欠落などの事故リスクを洗い出す。 |
| `architecture-analyst` | `.agents/skills/banking-architecture-analysis` | Go、PostgreSQL、トランザクション、API、レイヤード設計を評価する。 |
| `ai-review-board` | `.agents/skills/banking-review-board` | 各提案を採択、却下、保留に分類し、accepted scope を作る。 |
| `implementer` | `.agents/skills/scoped-banking-implementation` | accepted scope だけを小さく実装し、隣接改善を避ける。 |
| `code-reviewer` | `.agents/skills/banking-code-review` | コード品質、責務分離、テスト不足、Goらしさをレビューする。 |
| `security-reviewer` | `.agents/skills/banking-security-review` | 認証、認可、入力検証、秘密情報、SQL injection、監査証跡をレビューする。 |
| `banking-reviewer` | `.agents/skills/banking-ledger-review` | 残高更新、元帳、取引履歴、取消不能性、冪等性をレビューする。 |
| `final-report-writer` | `.agents/skills/banking-final-report` | 人間の最終レビュー用に変更、リスク、未解決事項を整理する。 |

各 subagent の `developer_instructions` には、対応 skill のパスを明記しています。subagent を呼び出すときは、必要に応じて `$banking-risk-analysis` のように明示指定するか、agent の責務に合う作業を依頼して暗黙選択させます。

## 推奨ワークフロー

1. `product-planner` が改善案と MVP 候補を出す。
2. `domain-analyst` が銀行ドメイン観点で評価する。
3. `risk-analyst` が金融事故リスクを評価する。
4. `architecture-analyst` が Go/PostgreSQL/設計観点で評価する。
5. `ai-review-board` が提案を統合し、「採択」「却下」「保留」と accepted scope を決め、`docs/decision-logs/` に意思決定ログを残す。
6. `implementer` が accepted scope だけを実装する。却下・保留・暗黙の改善・隣接改善は実装しない。
7. `code-reviewer`、`security-reviewer`、`banking-reviewer` が実装後レビューを行う。
8. 必要なら `ai-review-board` がレビュー結果を再度「採択」「却下」「保留」に整理する。
9. `final-report-writer` が人間向け最終レビュー資料を作る。
10. 人間が最終承認、方針の大きな修正、危険な判断の差し戻しを行う。

## 人間が見るべき最終レビュー観点

- 採択された改善案が、学習用バンクシステムの目的と範囲に合っているか。
- 残高更新、取引履歴、監査ログ、冪等性、認証認可、DBトランザクション境界に危険な曖昧さがないか。
- AI が安全上重要な仕様やプロダクト方針を過度に確定していないか。
- `ai-review-board` の採択・却下・保留理由が納得できるか。
- `implementer` が accepted scope 以外の隣接改善を実装していないか。
- reviewer が直接ファイル変更せず、指摘と改善提案に留めているか。
- 未解決事項や人間確認事項が次の作業に回せる粒度で残っているか。

## AIが採択してよいもの / 人間に確認すべきもの

### AIが採択してよいもの

- 既存ドキュメントの方針に明確に沿う小さな改善。
- テスト追加、命名改善、責務分離、エラーメッセージ整理など、プロダクト方針を変えない保守性向上。
- 金額を整数で扱う、トランザクション内で残高と取引履歴を更新するなど、既存の設計原則を具体化する実装。
- 監査ログ項目の不足を補うなど、既存方針の範囲内で安全性を高める変更。

### 人間に確認すべきもの

- MVP の範囲変更、対象ユーザー変更、外部銀行接続、多通貨対応などプロダクト方針を変える判断。
- 残高モデル、元帳モデル、取消/組戻し仕様、口座状態遷移など、後戻りが難しい金融ドメイン仕様。
- 認証方式、権限モデル、監査ログ保持方針、個人情報保持方針など安全上重要な仕様。
- データ削除、履歴改変、監査証跡の省略につながる判断。
- 大規模リファクタリング、DBスキーマ破壊的変更、移行手順が必要な変更。

## 意思決定ログ

`ai-review-board` は、採択・却下・保留の判断を `docs/decision-logs/` 配下に残します。テンプレートは `docs/decision-logs/TEMPLATE.md` を使用してください。
