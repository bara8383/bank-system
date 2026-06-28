# code-reviewer: 2026-06-29-001

## レビュー対象

- `docs/ai/cycles/2026-06-29-001/planner.md`
- `docs/ai/cycles/2026-06-29-001/implementer.md`
- 実装差分:
  - `docs/design-principles.md`
  - `docs/data-model.md`
  - `docs/test-strategy.md`

## 確認した前提

- `.codex/agents/README.md` と `docs/ai/cycles/README.md` の artifact protocol を確認した。
- `.agents/skills/banking-code-review/SKILL.md` の code-reviewer 出力契約を確認した。
- `README.md`, `AGENTS.md`, `docs/START_HERE.md` を確認した。
- `docs/ai/output/human/` は存在せず、人間からの追加助言は確認できなかった。
- `git status --short` では `docs/design-principles.md`, `docs/data-model.md`, `docs/test-strategy.md` の変更と `docs/ai/cycles/2026-06-29-001/` の未追跡ファイルを確認した。

## 総評

今回の実装差分は、planner の accepted scope である「元帳・残高方向・成功時 transaction 境界を docs に具体化する」に概ね適合している。Go ソース、DB schema、migration、repository、transaction manager、業務 API、認証認可には踏み込んでおらず、docs-only の小さい改善として妥当である。

ただし、将来の PostgreSQL migration と transaction manager 実装で参照される粒度として、制約一覧と rollback テスト観点に拾い漏れやすい曖昧さが残っている。

## Findings

### Finding 1: Medium - `transactions.balance_after` の DB 制約候補が制約一覧に反映されていない

**根拠**

- `docs/data-model.md:91` では、`balance_after` は対象口座の更新後残高と一致し、0 以上であると明記されている。
- 一方で、同じ文書の主な制約案には `accounts.balance_amount >= 0` と `transactions.amount > 0` はあるが、`transactions.balance_after >= 0` が含まれていない（`docs/data-model.md:135` から `docs/data-model.md:143`）。
- planner の accepted scope は `balance_after` の意味と制約案を追記することを求めていた。

**影響**

将来の PostgreSQL migration 実装時に、実装者が「主な制約案」だけを参照すると、`transactions.balance_after` の非負制約を DB 側に入れ忘れる可能性がある。アプリケーション層では残高非負を守る前提でも、DB 制約が弱いとバグや手動操作による不正な履歴行を防ぎにくい。

**推奨修正**

次サイクルで `docs/data-model.md` の主な制約案に、少なくとも次を追加する。

- `transactions.balance_after` は 0 以上にする。
- 可能なら、`transactions.currency` と `accounts.currency` の一致、`transaction_type` の許可値、振込時の `related_transaction_id` / `transfer_request_id` の扱いは別 scope で制約候補として整理する。

**次サイクル planner への入力**

DB schema / migration に進む前に、`docs/data-model.md` の制約案を本文の金融不変条件と同期する小 scope を検討する。今回の範囲では `transactions.balance_after >= 0` の追記が最小。

### Finding 2: Medium - transaction rollback を検証するテスト観点がまだ実装可能な粒度まで分かれていない

**根拠**

- `docs/test-strategy.md:51` から `docs/test-strategy.md:59` では、残高更新と取引履歴作成が同じデータベーストランザクションで整合すること、出金失敗や振込失敗で片側だけ残らないことが追記されている。
- ただし、残高不足や入力不備のような業務拒否と、同一 DB transaction 内の途中エラーを注入して rollback されることの区別がない。
- Go/PostgreSQL 実装では、repository または transaction manager のテストで「残高更新後、取引履歴作成前にエラー」「振込元更新後、振込先更新前にエラー」のような途中失敗を明示的に検証する必要がある。

**影響**

業務上の失敗ケースだけをテストしていると、実装が実際に `BEGIN` / `COMMIT` / `ROLLBACK` 境界を正しく扱っているかを検出しにくい。特に振込では、片側残高だけ更新されたり、片側の取引履歴だけ作成されたりする事故を unit / integration test で再現できないまま実装が進む恐れがある。

**推奨修正**

次サイクルで `docs/test-strategy.md` に、業務拒否とは別に transaction rollback テスト観点を追記する。

- 入金: 残高更新後、取引履歴作成前の疑似エラーで残高も履歴も残らないこと。
- 出金: 残高減少後、取引履歴作成前の疑似エラーで残高が戻り、履歴も残らないこと。
- 振込: 振込元更新後、振込先更新前、片側取引履歴作成後などの疑似エラーで、2 口座残高、2 取引履歴、振込依頼状態がすべて rollback されること。

**次サイクル planner への入力**

DB 接続や repository 実装の accepted scope を作る前に、transaction manager / repository 層で注入可能なエラー点と rollback 検証方針を docs に分離する候補を検討する。これは実装方式の最終決定ではなく、テスト観点の具体化として扱える。

## 問題なしと判断した点

- `docs/design-principles.md` の成功時 transaction 境界は、入金、出金、振込の最小構成を明示しており、accepted scope と整合している。
- `docs/data-model.md` の `transaction_type` ごとの増減方向と `balance_after` の説明は、残高非負、正の整数金額、振込の二面性と矛盾していない。
- `reversal`、並行更新制御、冪等性キー詳細、監査ログ書き込み失敗時、認証認可は未確定として残されており、後戻りしにくい金融仕様を今回 scope で勝手に確定していない。
- Go ソースコードや DB 実装に踏み込んでいないため、現時点で Go の package 境界、SQL、repository 責務分離に直接の回帰はない。

## テスト確認

- レビュー対象は docs-only 変更のため、追加の業務テストはない。
- `command -v go` は出力なしで、現在の環境では `go` コマンドを確認できなかった。`go test ./...` は未実行。

