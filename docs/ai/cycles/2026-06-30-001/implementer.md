# implementer: 2026-06-30-001

## 参照した accepted scope

- `docs/ai/cycles/2026-06-30-001/planner.md` の accepted scope「domain 型の境界再検証 helper を追加する」を参照した。
- 対象ファイル/領域は `internal/domain/money.go`、`internal/domain/money_test.go`、`README.md`、本ファイルに限定した。
- 実装しないこととして指定された HTTP route、業務 API、PostgreSQL、認証・認可、監査ログ永続化、冪等性、上限値、行ロック、多通貨などは実装対象外とした。

## 変更内容

- `Amount.Validate() error` を追加し、0 以下の amount を `ErrAmountMustBePositive` で拒否する境界再検証 API を追加した。
- `Balance.Validate() error` を追加し、負の balance を `ErrBalanceMustBeNonNegative` で拒否する境界再検証 API を追加した。
- `NewAmount` / `NewBalance` は、それぞれ作成した値に対して `Validate()` を呼び、既存 sentinel error を返す挙動を維持しつつ validation rule を共有する形にした。
- `AddBalance` / `SubtractBalance` は不正 amount 判定を `Amount.Validate()` へ寄せ、エラー時に元 balance を返す既存挙動を維持した。
- `Amount.Validate()` / `Balance.Validate()` の unit test を追加した。
- README の現在の実装範囲に、constructor 以外の境界再検証用 `Validate()` method を提供することを追記した。

## scope 適合性

- accepted scope の domain package 内の小さな変更に限定し、HTTP / DB / API / 認証 / 監査 / 冪等性の仕様や実装は追加していない。
- `Amount{}` は invalid、`Balance{}` は 0 円残高として valid という planner の作業仮定に従った。
- エラーは既存 sentinel error のみを再利用し、新しい error code は追加していない。
- 金額上限、残高上限、日次上限など未確定の業務上限は `Validate()` に含めていない。

## 実装しなかったこと

- HTTP route、handler、request / response schema は追加していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装していない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager は作成していない。
- 認証、認可、Cookie session、CSRF token、ログアウトは実装していない。
- 監査ログ永続化、監査ログ分類、outbox、非同期補償は実装していない。
- 冪等性キー、`transfer_requests` 状態遷移、request body hash、同一キー衝突処理は実装していない。
- 取引金額上限、残高上限、日次上限、PostgreSQL 行ロック、lock order、deadlock 方針、多通貨、為替、小数、丸め、手数料、利息、取消 / reversal は実装していない。
- 外部依存ライブラリは追加していない。

## テスト結果

- ✅ `gofmt -w internal/domain/money.go internal/domain/money_test.go`
- ✅ `go test ./...`
- ✅ `rg -n "float32|float64" internal/domain` は一致なし。
- ✅ `git diff --name-only && git ls-files --others --exclude-standard` で、変更が accepted scope 内の `README.md`、`docs/ai/cycles/2026-06-30-001/implementer.md`、`internal/domain/money.go`、`internal/domain/money_test.go` に限定されていることを確認した。

## 作業仮定

- `Validate() error` は「値 object が現在の不変条件を満たすか」を確認する method とし、constructor の代替ではなく service / repository / persistence 境界の再検証補助として扱う。
- MVP 現時点の不変条件は、取引金額は 0 より大きい、残高は 0 以上、JPY の整数最小通貨単位に限定する。
- `Balance{}` は Go のゼロ値として発生し得るため、0 円残高として valid に扱う。
- `Amount{}` は 0 円取引を表すため invalid に扱う。
