# implementer: 2026-07-01-001

## 参照した accepted scope

- `docs/ai/cycles/2026-07-01-001/planner.md` の accepted scope「口座ステータス domain helper を追加する」を参照した。
- 実装対象は `internal/domain/` の口座ステータス型・validation・残高変更可否 helper、対応 unit test、README 更新、この `implementer.md` 作成に限定した。
- 実装しないこととして、HTTP route / handler、業務 API、永続 entity、PostgreSQL、repository、transaction manager、行ロック、認証・認可、監査ログ、冪等性、金額上限、reversal などを scope 外として扱った。

## 変更内容

- `internal/domain/account.go` を追加し、`AccountStatus` と MVP 有効値 `active` / `suspended` / `closed` を定義した。
- `AccountStatus.Validate()` を追加し、MVP 有効値だけを許可し、空文字や未知 status を `ErrInvalidAccountStatus` で拒否するようにした。
- `EnsureAccountCanTransact(status AccountStatus) error` を追加し、残高変更系操作へ進めるのは `active` のみとした。`suspended` / `closed` は `ErrAccountNotActive` で拒否し、unknown status は `ErrInvalidAccountStatus` として区別した。
- `internal/domain/account_test.go` を追加し、status validation と残高変更可否 helper の正常系・異常系を確認した。
- `README.md` の現在の実装範囲を更新し、金額・残高 helper に加えて口座ステータス validation helper があることを明記した。

## scope 適合性

- 変更ファイルは accepted scope で許可された `internal/domain/`、`README.md`、`docs/ai/cycles/2026-07-01-001/implementer.md` に限定した。
- 口座ステータス helper は pure domain code とし、HTTP、DB、認証、認可、監査ログ、冪等性への依存を追加していない。
- docs 上の「有効」「停止中」「解約済み」は、実装上の `active` / `suspended` / `closed` に対応させた。
- unknown status は業務上の停止口座ではなく status validation error として扱い、`ErrAccountNotActive` と混同しないようにした。

## 実装しなかったこと

- HTTP route、handler、request / response schema は追加していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装していない。
- Account aggregate、Account ID、account number 採番、customer ID、owner relation、残高 field を持つ永続 entity は作成していない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、行ロックは実装していない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC は実装していない。
- 監査ログ永続化、audit `failure_reason` mapping、失敗監査ログの独立 transaction は実装していない。
- 冪等性キー、request body hash、transfer request 状態遷移は実装していない。
- 金額上限、残高上限、日次上限、reversal / 取消、多通貨、利息、手数料、外部銀行連携は実装していない。

## テスト結果

- ✅ `gofmt -w internal/domain/account.go internal/domain/account_test.go`
- ✅ `go test ./...`
- ✅ `git diff --name-only`
- ✅ `rg -n "AccountStatus|ErrAccount|EnsureAccountCanTransact" internal/domain README.md`

## 作業仮定

- MVP の口座状態の内部表現は planner accepted scope どおり `active` / `suspended` / `closed` とした。これは API schema や DB enum の最終決定ではない。
- `EnsureAccountCanTransact` の対象は、入金・出金・振込のように口座残高へ影響する操作とした。残高照会や取引履歴照会に同じ helper を使うかは今回決めていない。
- `suspended` と `closed` は、どちらも残高変更不可として同じ sentinel error `ErrAccountNotActive` を返す。利用者向けメッセージや監査分類で分けるかは次 cycle 以降の error mapping / audit design で扱う。
- unknown status はデータ破損または mapper 不備に近い扱いとして、業務上の停止/解約済み口座とは別の `ErrInvalidAccountStatus` を返す。
