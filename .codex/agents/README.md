# バンクシステム開発用 Codex Subagents

このディレクトリには、Go + PostgreSQL を中心にした銀行/金融トランザクション学習用バンクシステムのための Codex Subagent 定義を配置します。agent 本文はすべて日本語で記述し、TOML の key 名のみ Codex 仕様に合わせて英語のままにします。

## 方針

- subagent は `planner`、`implementer`、reviewer 群の 3 ロールで運用します。
- ユーザーは毎サイクル、`planner`、`implementer`、reviewer 群を並列 spawn できます。
- agent 同士は直接同期せず、`docs/ai/cycles/<cycle-id>/` 配下の Markdown 成果物を介して連携します。
- `planner` は改善案と accepted scope を作り、`implementer` は accepted scope だけを実装します。
- reviewer 群は専門観点ごとに repo 全体または直近実装差分をレビューし、次サイクル planner への入力を残します。

## Codex TOML 方針

- 各 agent は `.codex/agents/<agent-name>.toml` に 1 ファイル 1 agent として配置します。
- 各 agent は `name`, `description`, `developer_instructions` を必ず持ちます。
- `model_reasoning_effort` は金融リスク、設計判断、レビュー品質を重視して原則 `high` を使います。
- `sandbox_mode` は、各 agent が `docs/ai/cycles/<cycle-id>/` に Markdown 出力を保存できるよう `workspace-write` を使います。
- 実装担当以外の agent は、ソースコードや設計文書を直接変更せず、書き込みを cycle 成果物に限定します。

## 各 agent の役割

| Agent | 役割 | 出力 | 実装可否 |
| --- | --- | --- | --- |
| `planner` | repo 現状、人間レビュー、reviewer 群の出力、未実装領域から MVP/改善案を作り、採択/却下/保留と accepted scope を決める。 | `docs/ai/cycles/<cycle-id>/planner.md` | 不可 |
| `implementer` | 同一 cycle の `planner.md` にある accepted scope だけを分解して実装する。accepted scope がなければ blocked を記録する。 | `docs/ai/cycles/<cycle-id>/implementer.md` | 可 |
| `code-reviewer` | Go/PostgreSQL/トランザクション設計/保守性/テスト観点で repo 全体または実装差分をレビューする。 | `docs/ai/cycles/<cycle-id>/code-reviewer.md` | 不可 |
| `security-reviewer` | 認証、認可、入力検証、ログ、秘密情報、SQL injection、権限境界、監査証跡をレビューする。 | `docs/ai/cycles/<cycle-id>/security-reviewer.md` | 不可 |
| `banking-reviewer` | 残高、元帳、取引履歴、冪等性、状態遷移、銀行ドメイン、金融事故リスクをレビューする。 | `docs/ai/cycles/<cycle-id>/banking-reviewer.md` | 不可 |

## Artifact protocol

- 各サイクルは `docs/ai/cycles/<cycle-id>/` を使います。
- `<cycle-id>` はユーザー指定があればそれを使い、指定がなければ `YYYY-MM-DD-001` のように日付と連番で作ります。
- `planner.md` の必須項目: repo現状、入力レビュー、改善候補、採択、却下、保留、accepted scope、実装しないこと、人間確認事項。
- `implementer.md` の必須項目: 参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項。
- reviewer 出力の必須項目: Finding、根拠、影響、推奨修正、次サイクル planner への入力。

## 並列 spawn ルール

- `planner` は常に実行できます。
- `implementer` は同一 cycle の `planner.md` に accepted scope がない場合、実装せず `blocked: accepted scope not found` を記録します。
- reviewer 群は実装差分があれば差分レビューを優先し、実装差分がなければ repo 全体レビューを行います。
- 各 agent は作業開始時に `git status --short`、README、AGENTS.md、docs 配下の設計文書、human notes、cycle 成果物を確認し、ユーザー作業や他 agent 作業を壊しません。

## Repo-local skills

| Agent | 対応 skill | 用途 |
| --- | --- | --- |
| `planner` | `.agents/skills/banking-planning` | repo-grounded planning、採択判断、accepted scope 作成。 |
| `implementer` | `.agents/skills/scoped-banking-implementation` | accepted scope のみを小さく実装し、scope がなければ blocked を記録する。 |
| `code-reviewer` | `.agents/skills/banking-code-review` | Go/PostgreSQL/設計/保守性/テストレビュー。 |
| `security-reviewer` | `.agents/skills/banking-security-review` | 金融系セキュリティレビュー。 |
| `banking-reviewer` | `.agents/skills/banking-ledger-review` | 元帳、残高、取引履歴、ドメイン、金融事故リスクレビュー。 |

## 推奨ワークフロー

1. ユーザーが同じ `<cycle-id>` を指定して `planner`、`implementer`、reviewer 群を spawn する。
2. `planner` が repo と過去成果物を読み、`planner.md` に accepted scope を出す。
3. `implementer` は `planner.md` に accepted scope があれば実装し、なければ blocked を記録する。
4. reviewer 群は repo 全体または実装差分を並列レビューし、それぞれの Markdown に次サイクル planner への入力を残す。
5. 次サイクルの `planner` は前サイクルの reviewer 出力と human notes を読み、次の改善案と accepted scope を作る。

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
