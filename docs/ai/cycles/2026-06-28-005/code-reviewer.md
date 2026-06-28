# code-reviewer: 2026-06-28-005

## レビュー対象

- cycle-id: `2026-06-28-005`
- 役割: `code-reviewer`
- レビュー種別: repo-wide review
- 理由:
  - 作業開始時の `git status --short` は空で、実装差分は確認できなかった。
  - 作業中に同一 cycle の `docs/ai/cycles/2026-06-28-005/implementer.md` が存在することを確認したが、内容は `blocked: accepted scope not found` であり、Go ソースコード、README、設計文書、テストコード、設定ファイルの変更はない。
  - そのため、差分レビューではなく repo 全体レビューを行った。

## 確認した入力

- `.agents/skills/banking-code-review/SKILL.md`
- `.agents/skills/banking-code-review/references/banking-quality-rubric.md`
- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `git status --short`
- `README.md`
- `AGENTS.md`
- `docs/START_HERE.md`
- `docs/design-principles.md`
- `docs/domain-model.md`
- `docs/data-model.md`
- `docs/use-cases.md`
- `docs/mvp.md`
- `docs/security-notes.md`
- `docs/test-strategy.md`
- `docs/decision-logs/2026-06-27-subagent-parallel-cycle.md`
- `docs/ai/output/human/`: ディレクトリなし。追加の human notes は確認できなかった。
- `docs/ai/cycles/2026-06-28-001/` から `2026-06-28-004/` の cycle 成果物
- `docs/ai/cycles/2026-06-28-005/implementer.md`
- `cmd/server/main.go`
- `cmd/server/main_test.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_test.go`
- `go.mod`

## 実行した確認

- `git status --short`
- `git diff --stat`
- `git log --oneline --decorate -5`
- `rg --files docs`
- `rg` による TODO/FIXME、PostgreSQL、transaction、冪等性、`balance_after`、`transaction_type` 関連語の確認
- `go test ./...`

`go test ./...` は、この実行環境で `go` コマンドが見つからず実行できなかった。

## Finding

### Finding 1: cycle 004 の accepted scope が未実装のまま残っており、元帳・残高方向・transaction 境界の docs が実装可能な粒度に達していない

- 重大度: High

#### 根拠

- `docs/ai/cycles/2026-06-28-004/planner.md` は、`docs/design-principles.md`、`docs/data-model.md`、`docs/test-strategy.md` に対して、残高変更成功時の同一 DB transaction 方針、`transactions.transaction_type` ごとの残高増減方向、`transactions.balance_after` の意味、将来テスト観点を追記する accepted scope を定義している。
- 一方で `docs/ai/cycles/2026-06-28-004/implementer.md` は、accepted scope 不在として `blocked: accepted scope not found` を記録しており、実装・設計文書更新を行っていない。
- 現在の `docs/design-principles.md` は「残高変更に取引履歴を残す」「振込を原子的に扱う」という原則を持つが、入金、出金、振込それぞれで、どの更新を同一 DB transaction に含めるかはまだ具体化されていない。
- 現在の `docs/data-model.md` は `transactions.transaction_type` の候補と `balance_after` カラムを挙げるが、`deposit`、`withdrawal`、`transfer_debit`、`transfer_credit`、`reversal` の残高増減方向や、`balance_after` が口座ごとの取引適用直後残高であることを明文化していない。
- 現在の `docs/test-strategy.md` は原子性・冪等性を重点観点にしているが、残高更新と取引履歴作成の同一 transaction、`balance_after`、失敗時に残高と取引履歴が変わらないことの具体テストまでは落ちていない。

#### 影響

- このまま PostgreSQL schema、repository、入出金・振込 API 実装に進むと、handler / service / repository ごとに残高更新と取引履歴作成の境界がばらつく可能性が高い。
- `balance_after` の意味が曖昧なままだと、振込の出金側と入金側でどの口座の残高を記録するかが実装者依存になり、残高照会・明細表示・監査時の説明が難しくなる。
- `reversal` の扱いを未確定と明示しないまま実装が進むと、取消・訂正を既存取引の更新や削除で表現する事故につながりやすい。

#### 推奨修正

- 次 cycle planner は、cycle 004 planner の accepted scope を再採択するか、同等の小さな docs scope として採択し直す。
- 最低限、次を docs に反映してから DB schema / 業務 API 実装へ進む。
  - `transaction_type` ごとの残高増減方向。
  - `balance_after` は対象口座に当該取引を適用した直後の残高であり、0 以上であること。
  - 入金成功時は残高増加と `deposit` 取引履歴作成を同一 DB transaction に含めること。
  - 出金成功時は残高減少と `withdrawal` 取引履歴作成を同一 DB transaction に含めること。
  - 振込成功時は振込元残高減少、振込先残高増加、2 件の取引履歴、振込依頼成功更新を同一 DB transaction に含めること。
  - `reversal`、並行更新制御、冪等性キー scope、監査ログ書き込み失敗時の扱いは、人間確認事項として未確定のまま分離すること。

#### 次サイクル planner への入力

- accepted scope 候補: 「cycle 004 の元帳・残高方向・成功時 DB transaction 境界 docs 更新を再実行する」。
- この scope が完了するまで、入金、出金、振込、PostgreSQL migration、repository 実装は採択しないことを推奨する。

### Finding 2: Go skeleton は現行範囲では妥当だが、業務 API を追加する前の package 境界と error mapping がまだ未定義

- 重大度: Medium

#### 根拠

- 現在の Go 実装は `cmd/server` と `internal/httpapi` のみで、`GET /healthz` の固定応答に限定されている。
- `cmd/server/main.go` は listen address、timeout、HTTP server 構築を扱い、`internal/httpapi/router.go` は routing と health handler を扱っている。現行範囲では責務分離は過不足ない。
- 一方で、MVP の顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会を追加する際の `domain` / `application service` / `repository` / `http handler` の境界はまだ docs にない。
- `docs/use-cases.md` と `docs/test-strategy.md` は、入力不正、未認証、認可不足、対象なし、状態不正、残高不足、冪等性、処理途中エラーを分けて扱う必要を示しているが、Go の error type と HTTP status / response body / 監査用 failure reason の対応表はまだない。

#### 影響

- 業務 handler が SQL、認可、残高計算、transaction 開始、監査ログ、HTTP 応答生成を直接持つ構造になると、テストが重くなり、transaction 境界も追跡しにくくなる。
- エラー分類が曖昧なままだと、残高不足や状態不正を `500` として返す、内部エラー詳細を利用者へ返す、監査ログの failure reason が実装箇所ごとに揺れる、といった保守性・調査性の問題が起きる。

#### 推奨修正

- 最初の業務 API 実装前に、最小 package layout と error mapping 方針を docs 化する。
- 方針としては、HTTP handler は request decode / 基本 validation / actor 抽出 / service 呼び出し / response mapping に寄せ、残高変更・認可・transaction 境界は application service 層へ寄せる。
- 業務エラーは少なくとも、入力不正、未認証、認可不足、対象なし、状態不正、残高不足、冪等性衝突、内部エラーを区別できる形にする。

#### 次サイクル planner への入力

- accepted scope 候補: 「業務 API 実装前の Go package layout と error mapping 設計を docs に追加する」。
- ただし、Finding 1 の元帳・transaction 境界 docs が未反映のため、順序としては Finding 1 を先に扱う方が安全。

### Finding 3: PostgreSQL migration / transaction manager / repository 境界が未具体化で、DB 実装に入るには前提が不足している

- 重大度: Medium

#### 根拠

- `AGENTS.md` は DB を PostgreSQL としている。
- `docs/data-model.md` は `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs` の候補と、残高非負、金額正数、口座番号一意、冪等性キー一意、外部キーの制約案を示している。
- しかし現行 repo には migration ツール、migration ファイル、PostgreSQL 接続コード、`database/sql` 利用、transaction manager、repository interface / implementation が存在しない。
- 並行出金・並行振込時に、行ロック、条件付き UPDATE、分離レベル、ロック順序のどれで残高を守るかも未確定である。

#### 影響

- DB schema を急いで作ると、後から冪等性 scope、残高競合制御、監査ログ境界、ID 型、口座番号採番方式に合わせて破壊的 migration が必要になりやすい。
- repository が transaction を受け取る設計なしに実装されると、残高更新、取引履歴作成、振込依頼更新、監査ログ記録が別々の connection / transaction で実行されるリスクがある。

#### 推奨修正

- PostgreSQL 実装に入る前に、DB 方針を docs scope として小さく決める。
- 最低限、migration ツール、ローカル DB 起動方法、test database 方針、transaction 開始責務、repository への transaction context の渡し方、残高競合制御方式を分けて検討する。
- schema 実装を採択する場合は、制約と index まで含めるが、冪等性 scope や取消仕様の未確定部分は人間確認事項として明示する。

#### 次サイクル planner への入力

- accepted scope 候補: 「PostgreSQL migration 方針と transaction manager / repository 境界の docs 化」。
- Finding 1 の transaction 境界 docs が反映された後に採択することを推奨する。

### Finding 4: 現行の health check / HTTP server 設定に修正必須の Go 不具合は見つからない

- 重大度: Info

#### 根拠

- `cmd/server/main.go` は既定 listen address を `127.0.0.1:8080` とし、`BANK_SYSTEM_HTTP_ADDR` による明示的 override を持つ。
- `http.Server` には `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` が設定されている。
- `internal/httpapi/router.go` は `/healthz` のみを提供し、`GET` 以外を `405 Method Not Allowed` と `Allow: GET` で拒否している。
- `/healthz` は固定 JSON のみを返し、DB 情報、秘密情報、環境変数、内部 path を返さない。
- `cmd/server/main_test.go` と `internal/httpapi/router_test.go` は、server config、timeout、health response、method 拒否、不要情報非露出を検証している。

#### 影響

- 現在の実装範囲では、Go の責務分離と README の説明は整合している。
- PostgreSQL、残高更新、取引履歴、監査ログ、冪等性、認証認可は未実装であり、コード上の金融事故バグは現時点では発火しない。

#### 推奨修正

- この finding に対する修正は不要。
- 業務 API を追加するまでは、`cmd/server` は起動設定、`internal/httpapi` は routing / handler という現在の分離を維持する。

#### 次サイクル planner への入力

- HTTP skeleton hardening は完了扱いでよい。
- 次 cycle は、health check の追加改善ではなく、元帳・transaction 境界、Go package / error mapping、PostgreSQL 方針、認証認可方針のいずれかを小さく採択する。

## テスト・確認結果

- `git status --short`: 作業開始時点では空。作業中に同一 cycle の `implementer.md` が確認されたが、内容は blocked artifact であり、実装差分ではない。
- `git diff --stat`: 空。
- `go test ./...`: 実行不可。`/bin/bash: line 1: go: command not found`。

## 人間確認事項

1. cycle 004 planner の accepted scope を次 cycle でそのまま再採択してよいか。
2. 並行出金・並行振込時の残高保護方式を、PostgreSQL 行ロック、条件付き UPDATE、または別方式のどれで学習・実装するか。
3. 冪等性キーの一意 scope を、依頼者、振込元口座、操作種別、request hash のどこまで含めるか。
4. 監査ログ書き込み失敗時に、残高変更を伴う業務処理を失敗させるか、別経路で補償するか。

