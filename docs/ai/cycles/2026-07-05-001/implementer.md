# implementer: 2026-07-05-001

## 参照した accepted scope

同一 cycle の `docs/ai/cycles/2026-07-05-001/planner.md` にある accepted scope「取引種別と残高反映の domain helper を追加する」を参照した。

実装対象として確認した範囲は次のとおり。

- `internal/domain/transaction.go` を追加し、MVP の取引種別を Go domain 層で表現する。
- valid な取引種別は `deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` の 4 種だけにする。
- 空文字、未知値、`reversal` は `ErrInvalidTransactionType` で拒否する。
- `ApplyTransaction` で、`balance.Validate()`、`amount.Validate()`、`transactionType.Validate()` の順に検証し、失敗時は元の `balance` を返す。
- `deposit` / `transfer_credit` は残高増加、`withdrawal` / `transfer_debit` は残高減少として実装する。
- `internal/domain/transaction_test.go` を追加し、valid / invalid transaction type、残高増加、残高減少、残高不足、invalid balance、invalid amount、invalid transaction type、overflow をテストする。
- README を更新し、現在の実装範囲と未実装機能リストを最新化する。

## 変更内容

- `internal/domain/transaction.go` を追加した。
  - `TransactionType` を string-based type として定義した。
  - `TransactionTypeDeposit`、`TransactionTypeWithdrawal`、`TransactionTypeTransferDebit`、`TransactionTypeTransferCredit` を追加した。
  - `ErrInvalidTransactionType` を追加した。
  - `TransactionType.Validate()` を追加し、MVP 4 種以外を拒否するようにした。
  - `ApplyTransaction` を追加し、取引種別ごとの残高増減方向を `AddBalance` / `SubtractBalance` に委譲するようにした。
- `internal/domain/transaction_test.go` を追加した。
  - MVP 4 種の validation 成功を確認した。
  - 空文字、未知値、`reversal` の validation 失敗を確認した。
  - `deposit` / `transfer_credit` が残高を増やすことを確認した。
  - `withdrawal` / `transfer_debit` が残高を減らすことを確認した。
  - 残高不足、invalid starting balance、invalid amount、invalid transaction type、deposit-like overflow で元の残高と既存 sentinel error または `ErrInvalidTransactionType` が返ることを確認した。
  - validation 順序が `balance`、`amount`、`transactionType` の順であることを確認した。
- `README.md` を更新した。
  - 現在の実装範囲に、取引種別 validation と `ApplyTransaction` による取引後残高計算を追記した。
  - `reversal` は取消仕様未確定のため valid type に含めていないことを明記した。
  - 取引履歴の永続化、transaction row 作成、`balance_after` の DB 保存、業務 API は未実装のままであることを明記した。

## scope 適合性

- 変更は accepted scope の対象ファイル/領域である `internal/domain/transaction.go`、`internal/domain/transaction_test.go`、`README.md`、`docs/ai/cycles/2026-07-05-001/implementer.md` に限定した。
- 実装は pure domain helper の追加に限定し、HTTP / DB / 認証 / 認可 / 監査ログ / 冪等性 / reversal の実装は追加していない。
- `ApplyTransaction` は transaction row や永続 entity を作成せず、将来 `transactions.balance_after` に保存し得る取引後残高の計算候補だけを返す helper とした。
- `reversal` は valid type に含めず、invalid transaction type として拒否するテスト対象に限定した。

## 実装しなかったこと

- `Transaction` aggregate、取引 ID、取引日時、説明、関連取引 ID、`balance_after` field を持つ永続 entity は作成していない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、行ロックは作成していない。
- 取引履歴 table への insert、監査ログ table への insert、成功 / 失敗監査ログの保存境界は実装していない。
- HTTP route、handler、request / response schema は追加していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装していない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC は実装していない。
- 冪等性キー、request body hash、transfer request 状態遷移は実装していない。
- `reversal` / 取消 / 組戻し / 訂正は実装していない。
- 金額上限、残高上限、日次上限は今回の code に固定していない。
- domain error / API response / audit `failure_reason` mapping は今回確定していない。

## テスト結果

- `gofmt -w internal/domain/transaction.go internal/domain/transaction_test.go`: 成功。
- `go test ./...`: 成功。
  - `bank-system/cmd/server`
  - `bank-system/internal/domain`
  - `bank-system/internal/httpapi`

## 作業仮定

- `TransactionType` の文字列値は `docs/data-model.md` の `transactions.transaction_type` 候補のうち、MVP 初期で方向が確定している 4 種に合わせた。
- `ApplyTransaction` が返す残高は、将来の `transactions.balance_after` に保存する値の計算候補だが、この cycle では DB 保存や transaction row 作成を行わない。
- validation / error 優先順位は accepted scope に従い、破損した starting balance を最初に検出するため `balance`、`amount`、`transactionType` の順にした。
- `transfer_debit` と `transfer_credit` は、1 件の振込における片側取引の残高方向だけを表す。この cycle では 2 口座の同一 DB transaction、lock 順序、冪等性キー、振込依頼状態は扱わない。
- `reversal` は `docs/data-model.md` では将来候補として残るが、human notes と accepted scope に従い MVP 初期 code では valid type に含めない。
