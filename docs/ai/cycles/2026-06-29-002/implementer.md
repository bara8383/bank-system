# implementer: 2026-06-29-002

## 参照した accepted scope

同一 cycle の `docs/ai/cycles/2026-06-29-002/planner.md` にある accepted scope「A. 金額・残高の最小 domain 型と validation を Go に追加する」を参照した。

実装対象として参照した範囲は次のとおり。

- Go domain 領域に、金額・残高 validation の小さな package を追加する。
- 金額は `int64` の整数最小通貨単位として扱い、MVP の通貨は JPY 固定とする。
- 取引金額は 0 より大きい値だけを有効にする。
- 残高は 0 以上を有効にする。
- 入金相当の加算 helper と、残高不足時にエラーを返す出金相当の減算 helper を追加する。
- unit test で、正の取引金額、0 円・負の取引金額拒否、0 円・正の残高、負の残高拒否、加算、残高内減算、残高不足減算時のエラーと元残高維持を確認する。
- README に、金額・残高 validation の domain 土台が追加されたことと、業務 API / DB / 認証などが引き続き未実装であることを反映する。

## 変更内容

- `internal/domain/money.go` を追加した。
  - `Amount` は JPY の整数最小通貨単位を `int64` で保持する。
  - `NewAmount` は 0 以下の取引金額を `ErrAmountMustBePositive` で拒否する。
  - `Balance` は JPY の整数最小通貨単位を `int64` で保持する。
  - `NewBalance` は負の残高を `ErrBalanceMustBeNonNegative` で拒否する。
  - `AddBalance` は入金相当の残高加算を行い、`int64` overflow 時は元残高と `ErrBalanceOverflow` を返す。
  - `SubtractBalance` は出金相当の残高減算を行い、残高不足時は元残高と `ErrInsufficientBalance` を返す。
- `internal/domain/money_test.go` を追加した。
  - accepted scope の unit test 観点を網羅し、helper にゼロ値の `Amount` が渡された場合と overflow 時も元残高を返すことを確認した。
- `README.md` を更新した。
  - 現在の実装範囲に `internal/domain` の金額・残高 validation helper を追加した。
  - 外部ライブラリ、DB 接続、認証、業務 API は未導入のままであることを維持した。

## scope 適合性

- 変更は、新規 Go domain package、新規 unit test、README、同 cycle の `implementer.md` に限定した。
- HTTP route、handler、request / response schema は追加していない。
- Go 標準ライブラリのみを使用し、外部依存は追加していない。
- 金額は `int64` 整数で扱い、`float32` / `float64` は使用していない。
- JPY 固定の前提を domain comment と `CurrencyJPY` 定数として表現したが、多通貨や通貨変換の実装は追加していない。
- 残高不足や overflow などのエラー時は元の `Balance` を戻り値として返し、呼び出し側が残高を変えない実装にできる API にした。

## 実装しなかったこと

- HTTP route、handler、request / response schema は実装していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装していない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager は実装していない。
- 認証、認可、ユーザー登録、パスワードハッシュ、Cookie session、CSRF token、ログアウトは実装していない。
- 監査ログ永続化、監査ログ正規化、監査ログ照会、outbox、非同期補償は実装していない。
- 冪等性キー、`transfer_requests` 状態遷移、request body hash、同一キー衝突処理は実装・確定していない。
- PostgreSQL 行ロック、ロック順序、デッドロック回避、分離レベルは実装・確定していない。
- 多通貨、為替、小数、丸め、手数料、利息、取消 / reversal は実装していない。

## テスト結果

- `gofmt -w internal/domain/money.go internal/domain/money_test.go`: 成功。
- `go test ./...`: 成功。
  - `bank-system/cmd/server`: ok。
  - `bank-system/internal/domain`: ok。
  - `bank-system/internal/httpapi`: ok。
- `rg -n "float32|float64" internal/domain`: 該当なし。`rg` の仕様上、マッチなしの終了コードは 1 だが、金額実装に浮動小数点型が含まれていないことを確認した。
- `git diff --name-only`: 変更範囲は `README.md`, `docs/ai/cycles/2026-06-29-002/implementer.md`, `internal/domain/money.go`, `internal/domain/money_test.go`。

## 作業仮定

- `Amount` / `Balance` は JPY の最小通貨単位を表す domain 型であり、MVP では 1 が 1 円を意味する。
- `Amount` のゼロ値は有効な取引金額ではないため、`AddBalance` / `SubtractBalance` でも 0 以下の amount を拒否する。
- `Balance` のゼロ値は 0 円残高として有効である。
- 加算時の `int64` overflow は、実取引では上限設計が別途必要だが、今回の小 helper では安全側に倒してエラーにし、元残高を返す。
- エラー型は現時点では sentinel error とし、HTTP error mapping や DB error mapping は業務 API / repository 実装時の別 scope とする。
