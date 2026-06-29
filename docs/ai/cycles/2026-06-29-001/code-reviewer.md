# code-reviewer: 2026-06-29-001

## レビュー対象

- `docs/ai/cycles/2026-06-29-001/planner.md`
- `docs/ai/cycles/2026-06-29-001/implementer.md`
- 実装差分（`HEAD^..HEAD`）:
  - `docs/design-principles.md`
  - `docs/security-notes.md`
  - `docs/data-model.md`
  - `docs/test-strategy.md`
  - `docs/ai/cycles/2026-06-29-001/implementer.md`

## 確認した前提

- `.codex/agents/README.md` と `docs/ai/cycles/README.md` の artifact protocol を確認した。
- `.codex/agents/code-reviewer.toml` と `.agents/skills/banking-code-review/SKILL.md` の code-reviewer 責務、禁止事項、出力契約を確認した。
- `AGENTS.md`, `README.md`, `docs/START_HERE.md`, `docs/design-principles.md`, `docs/data-model.md`, `docs/domain-model.md`, `docs/test-strategy.md`, `docs/ai/output/human/001-human-review.md` を確認した。
- `git status --short` は作業開始時点で未コミット変更なしだった。
- 実装差分は docs-only で、Go ソース、DB schema、migration、repository、transaction manager、業務 API、認証/認可実装には変更がないことを確認した。

## 総評

今回の実装差分は、planner の accepted scope である「監査ログ境界、失敗時扱い、閲覧権限、マスキング方針の docs 具体化」と、前 cycle reviewer 指摘の小補修（`transactions.balance_after >= 0`、rollback テスト観点）に概ね適合している。

成功監査ログを残高更新・取引履歴・振込依頼状態更新と同じ PostgreSQL データベーストランザクションに含める方針、失敗時監査ログを rollback 後の独立トランザクションで残す方針、監査ログ照会を MVP では `admin` のみに限定する方針、raw request body や秘密情報を監査ログへ保存しない方針が docs 間で読み取りやすくなった。

一方で、将来の transaction manager / repository テストへ落とし込むには、「成功監査ログの書き込み失敗時に業務データを rollback すること」をテスト観点として明示しておくと、fail closed をレスポンス上の失敗扱いだけで実装してしまうリスクを下げられる。

## Findings

### Finding 1: Medium - 成功監査ログ書き込み失敗時の rollback 検証観点が明示不足

**根拠**

- `docs/design-principles.md` は、残高変更を伴う成功操作の成功監査ログを業務データ更新と同じ PostgreSQL データベーストランザクションに含めるとしている。
- 同じ文書は、監査ログ書き込み失敗時に対象業務処理を成功扱いにせず、成功監査ログが書けない残高変更や権限変更は MVP では fail closed とするとしている。
- `docs/test-strategy.md` の「監査ログ書き込み失敗」観点は「対象業務処理が成功扱いにならない」としているが、残高・取引履歴・振込依頼状態・成功監査ログが同一 transaction で rollback され、コミット済みの業務データが残らないことまでは明示していない。
- `docs/test-strategy.md` の途中失敗 rollback テストは、残高更新後・取引履歴作成前、振込元更新後・振込先更新前、片側取引履歴作成後、commit 前を挙げているが、成功監査ログ insert 失敗を注入点として明示していない。

**影響**

将来の Go/PostgreSQL 実装で、成功監査ログの insert 失敗を「API レスポンスは失敗にするが、残高更新や取引履歴はすでに commit 済み」のように扱ってしまう余地が残る。これは docs の「成功監査ログを業務データ更新と同じ DB transaction に含める」方針とずれ、残高変更が発生したのに成功監査ログがない状態、または失敗扱いの応答と実データ更新が食い違う状態を見逃しやすくする。

**推奨修正**

次サイクルで `docs/test-strategy.md` の監査ログテストまたは DB transaction 途中失敗の rollback テストに、次のような観点を追加する。

- 入金/出金/振込で、残高更新・取引履歴作成後に成功監査ログ insert を失敗させた場合、同一 DB transaction 全体が rollback されること。
- その場合、口座残高、取引履歴、振込依頼状態、成功監査ログが残らず、必要なら rollback 後に失敗監査ログを独立して残すこと。
- API/サービス層の結果が失敗扱いであることに加え、DB 上の副作用が残っていないことを integration test で検証すること。

**次サイクル planner への入力**

DB 接続、repository、transaction manager、監査ログ repository の accepted scope を作る前に、「成功監査ログ書き込み失敗を注入した rollback テスト」を `docs/test-strategy.md` へ明記する小 scope を検討する。監査ログ書き込み失敗時 fail closed は、レスポンス上の失敗ではなく、業務データ commit を伴わないことまで含む前提として扱う。

## 問題なしと判断した点

- `docs/design-principles.md` の成功時 transaction 境界は、入金、出金、振込について、残高更新、取引履歴、振込依頼状態、成功監査ログを同一 DB transaction に含める方針を明示しており、planner の accepted scope と整合している。
- `docs/design-principles.md` は、業務拒否、DB transaction 途中失敗、監査ログ書き込み失敗を分け、失敗時監査ログを業務データ更新の rollback と独立して残す必要があると明記しているため、前回までの「失敗時監査ログが不要に見える」曖昧さは概ね解消されている。
- `docs/security-notes.md` と `docs/data-model.md` は、監査ログに保存する属性、`failure_reason` の安全な分類、raw request body と秘密情報を保存しない方針、`request_body_hash` 候補を追加しており、監査ログの保守性と将来 schema 検討の入力として妥当である。
- `docs/data-model.md` の主な制約案に `transactions.balance_after >= 0` が追加され、前回 code-reviewer 指摘の制約一覧漏れは解消されている。
- `docs/test-strategy.md` は、業務拒否と DB transaction 途中失敗の rollback テストを分け、残高・取引履歴・成功監査ログ・失敗監査ログの検証観点を具体化している。
- 変更は docs-only であり、Go package 境界、HTTP server、router、DB 実装に直接の回帰はない。

## テスト確認

- `go test ./...`: 成功。
  - `ok   bank-system/cmd/server`
  - `ok   bank-system/internal/httpapi`
- `git diff --name-only HEAD^ HEAD`: 実装差分が accepted scope 内の docs-only 変更であることを確認した。
- `git diff -- docs/ai/cycles/2026-06-29-001/code-reviewer.md`: reviewer 成果物のみを更新していることを確認した。
