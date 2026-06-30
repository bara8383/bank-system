# implementer: 2026-06-30-001

## 参照した accepted scope

同一 cycle の `docs/ai/cycles/2026-06-30-001/planner.md` の accepted scope「残高演算 helper で開始 `Balance` も再検証する」を参照した。

対象ファイルは accepted scope に明記された次の 4 ファイルに限定した。

- `internal/domain/money.go`
- `internal/domain/money_test.go`
- `README.md`
- `docs/ai/cycles/2026-06-30-001/implementer.md`

## 変更内容

- `AddBalance(balance Balance, amount Amount)` の先頭で `balance.Validate()` を呼ぶようにした。
  - 負の開始 `Balance` の場合は演算を継続せず、元の `balance` と `ErrBalanceMustBeNonNegative` を返す。
  - 開始 `Balance` が valid な場合は、従来どおり `amount.Validate()`、overflow 検出、加算を行う。
- `SubtractBalance(balance Balance, amount Amount)` の先頭で `balance.Validate()` を呼ぶようにした。
  - 負の開始 `Balance` の場合は演算を継続せず、元の `balance` と `ErrBalanceMustBeNonNegative` を返す。
  - 開始 `Balance` が valid な場合は、従来どおり `amount.Validate()`、残高不足判定、減算を行う。
- unit test を追加した。
  - `AddBalance` が負の開始 `Balance` を拒否し、元の `balance` を返すこと。
  - `SubtractBalance` が負の開始 `Balance` を拒否し、元の `balance` を返すこと。
  - 負の開始 `Balance` と invalid `Amount` が同時に渡った場合、`ErrBalanceMustBeNonNegative` が優先されることを `AddBalance` / `SubtractBalance` の両方で確認した。
- README の現在の実装範囲を更新し、残高加算・減算 helper が取引金額だけでなく開始残高も再検証することを追記した。

## scope 適合性

- accepted scope の実装対象 1〜6 の範囲内で実装した。
- 変更ファイルは accepted scope で許可された 4 ファイルに限定した。
- 新しい error code は追加せず、既存の `ErrBalanceMustBeNonNegative` / `ErrAmountMustBePositive` / `ErrInsufficientBalance` / `ErrBalanceOverflow` を維持した。
- `Balance{}` は引き続き 0 円残高として valid に扱う。
- invalid starting balance の検出を invalid amount より優先する validation 順序を test で固定した。

## 実装しなかったこと

accepted scope の「実装しないこと」に従い、次は実装していない。

- HTTP route、handler、request / response schema
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager
- 認証、認可、Cookie session、CSRF token、ログアウト
- 監査ログ永続化、監査ログ分類、outbox、非同期補償
- 冪等性キー、`transfer_requests` 状態遷移、request body hash、同一キー衝突処理
- 取引金額上限、残高上限、日次上限
- PostgreSQL 行ロック、lock order、deadlock retry / fail 方針
- 多通貨、為替、小数、丸め、手数料、利息、取消 / reversal
- 外部依存ライブラリ

## テスト結果

- `gofmt -w internal/domain/money.go internal/domain/money_test.go`
  - 成功。
- `go test ./...`
  - 成功。
  - `bank-system/cmd/server`、`bank-system/internal/domain`、`bank-system/internal/httpapi` が pass。
- `rg -n "float32|float64" internal/domain`
  - 一致なし。exit code 1 は「浮動小数点利用なし」の期待結果として扱った。
- `git diff --name-only`
  - `README.md`
  - `docs/ai/cycles/2026-06-30-001/implementer.md`
  - `internal/domain/money.go`
  - `internal/domain/money_test.go`
- `git ls-files --others --exclude-standard`
  - 出力なし。

## 作業仮定

- 負の開始 `Balance` は通常の外部 package 経路では作れないが、同一 package の mapper / test / 将来の repository helper では起こり得る破損値として扱う。
- `AddBalance` / `SubtractBalance` は、呼び出し側境界だけに依存せず、自身の入口で `Balance` と `Amount` の両方を検証する責務を持つ。
- validation 順序は開始 `Balance` が先、`Amount` が後である。破損した既存残高は、リクエスト金額不備より優先して検出する。
- 業務上限や HTTP / DB / 認証 / 監査 / 冪等性仕様は今回の helper validation には含めない。
