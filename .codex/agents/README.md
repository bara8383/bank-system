# バンクシステム開発用 Codex Subagents

このディレクトリには、Go + PostgreSQL を中心にした銀行/金融トランザクション学習用バンクシステムのための Codex Subagent 定義を配置します。agent 本文はすべて日本語で記述し、TOML の key 名のみ Codex 仕様に合わせて英語のままにします。

## 方針

- subagent は `planner`、`implementer`、reviewer 群の 3 ロールで運用します。
- cycle は `planner` -> `implementer` -> reviewer 群の直列で進めます。reviewer 群だけは、implementer の差分作成後に並列実行できます。
- agent 同士は直接同期せず、`docs/ai/cycles/<cycle-id>/` 配下の Markdown 成果物を介して連携します。
- `planner` は改善案と accepted scope を作り、`implementer` は accepted scope だけを実装します。
- reviewer 群は専門観点ごとに repo 全体または直近実装差分をレビューし、次サイクル planner への入力を残します。
- 人間の最終判断は PR レビューで行う前提とし、agent は未確定事項だけを理由に停止しません。未確定事項は作業仮定として明示し、差分と cycle 成果物に残します。

## Codex TOML 方針

- 各 agent は `.codex/agents/<agent-name>.toml` に 1 ファイル 1 agent として配置します。
- 各 agent は `name`, `description`, `developer_instructions` を必ず持ちます。
- 各 agent の既定モデルは `model = "gpt-5.4"` に統一します。
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
- `planner.md` の必須項目: repo現状、入力レビュー、改善候補、採択、却下、保留、accepted scope、実装しないこと、作業仮定。
- `implementer.md` の必須項目: 参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定。
- reviewer 出力の必須項目: Finding、根拠、影響、推奨修正、次サイクル planner への入力。

## Cycle rules

- 1 つの cycle は `planner` -> `implementer` -> reviewer 群の順で実行します。
- `planner` は最初に実行し、同一 cycle の `planner.md` に accepted scope を作ります。
- `implementer` は同一 cycle の accepted scope を実装します。直近完了済み cycle の accepted scope を使うのは、再実行や復旧などの例外時だけです。
- reviewer 群は `implementer` の差分作成後に並列実行できます。
- reviewer 群は実装差分があれば差分レビューを優先します。実装差分がない repo-wide review は、ユーザーが明示した場合または planner が次 scope 作成の入力として必要と判断した場合に限定します。
- requested cycle と例外 fallback のどちらにも accepted scope が存在しない場合だけ、`implementer` は `blocked: accepted scope not found` を記録します。
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

1. `planner` が repo と過去成果物を読み、`planner.md` に code-changing scope を 1 つ以上含む accepted scope を出す。
2. `implementer` は同一 cycle の accepted scope を実装し、必要な docs / README 更新とテストを同じ差分に含める。
3. `code-reviewer`、`security-reviewer`、`banking-reviewer` は、implementer の差分作成後に並列で実行し、それぞれの Markdown に次サイクル planner への入力を残す。
4. 次サイクルの `planner` は前サイクルの reviewer 出力と human notes を読み、次の改善案と accepted scope を作る。

## AIの作業仮定

PR レビューで人間が差分を確認する前提のため、agent は未確定事項を理由に停止しません。次の範囲は、既存 docs の方針に反しない限り AI が作業仮定を置いて採択・実装できます。

- 既存ドキュメントの方針に明確に沿う小さな改善。
- テスト追加、命名改善、責務分離、エラーメッセージ整理など、プロダクト方針を変えない保守性向上。
- 金額を整数で扱う、トランザクション内で残高と取引履歴を更新するなど、既存の設計原則を具体化する実装。
- 監査ログ項目の不足を補うなど、既存方針の範囲内で安全性を高める変更。
- MVP の実装を前に進めるための最小 DB schema、migration、repository、API skeleton。
- 認証、権限、監査ログ、冪等性、取消などの初期案。ただし、採用した仮定は docs と PR 差分で明示する。
- 破壊的変更や大きな仕様変更が必要な場合も、まずは小さな実装案または設計案として差分化し、PR レビューで修正可能にする。
