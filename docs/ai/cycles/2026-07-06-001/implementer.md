# implementer: 2026-07-06-001

## 参照した accepted scope

- `docs/ai/cycles/2026-07-06-001/planner.md` の accepted scope「domain error を safe failure category に写像する Go helper を追加する」を参照した。
- 対象は `internal/domain` の failure reason helper と対応テスト、README、security docs、同一 cycle の implementer artifact に限定した。

## 変更内容

- `internal/domain/failure_reason.go` を追加し、`FailureReason` string type、safe category constants、`ErrInvalidFailureReason`、`FailureReason.Validate()`、`FailureReasonFromError(error)` を実装した。
- `FailureReasonFromError` は `errors.Is` で既存 domain sentinel error と wrapped error を判定し、既知 error のみを固定分類へ写像する。`nil`、未知 error、対象外 error は空文字と `false` を返す。
- `internal/domain/failure_reason_test.go` を追加し、次を確認した。
  - 定義済み category constants は `Validate()` に通る。
  - 空文字、未知値、既存 sentinel error の raw `Error()` 文字列、`password=...` のような secret 風値は `ErrInvalidFailureReason` で拒否される。
  - 既存 domain sentinel error は期待する `FailureReason` に写像される。
  - wrapped error も `errors.Is` により写像される。
  - `nil` と未知 error は未分類になる。
  - `FailureReasonFromError` で得た値は `Validate()` に通る。
- `README.md` を更新し、現在の実装範囲に safe failure category helper を追加したこと、および HTTP error response、監査ログ永続化、DB schema などは未実装であることを明記した。
- `docs/security-notes.md` を更新し、domain error に由来する監査 `failure_reason` は helper の固定分類値を使い、raw request body や secret 類、未加工の自由入力値を保存・返却しない方針を追記した。

## scope 適合性

- accepted scope の実装対象 1〜9 を満たした。
- 既存 sentinel error の文言や semantics は変更していない。
- 新規 helper は pure Go の domain helper に留め、HTTP / DB / auth / audit persistence には踏み込んでいない。
- 変更ファイルは accepted scope 内の `internal/domain`、`README.md`、`docs/security-notes.md`、`docs/ai/cycles/2026-07-06-001/implementer.md` に限定した。

## 実装しなかったこと

- HTTP route、handler、request / response schema、status code mapping は追加していない。
- API response body の最終形式、利用者向け message 文言、i18n、エラーコード体系全体は確定していない。
- 監査ログ table、audit repository、audit service、成功 / 失敗監査ログの永続化は実装していない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、row lock は作成していない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC、admin 代行処理は実装していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装していない。
- 取引履歴 row、`balance_after` 永続化、transfer request 状態遷移、冪等性キー処理は実装していない。
- `reversal` / 取消 / 組戻し / 訂正は実装していない。
- 金額上限、残高上限、日次上限は code に固定していない。

## テスト結果

- `gofmt -w internal/domain/failure_reason.go internal/domain/failure_reason_test.go`: 成功。
- `go test ./...`: 成功。
- `git status --short`: 変更範囲が accepted scope 内であることを確認。
- `git diff --check`: 成功。
- `rg -n "password|token|secret|CSRF|session|request body" internal/domain docs README.md`: 新規 helper / docs が raw secret や raw request body を保存する実装に読めないことを確認。

## 作業仮定

- `FailureReason` は将来の audit `failure_reason`、API error response、安全な構造化ログで共通利用できる候補だが、今回 scope では HTTP status code、response schema、利用者向け文言は確定しない。
- `ErrBalanceMustBeNonNegative` は利用者入力不備というより永続化境界や内部状態異常に近いため、planner の作業仮定どおり `invalid_balance_state` に分類した。
- `ErrAccountNotActive` は suspended / closed の詳細を外部へ出さない分類として `account_not_active` にまとめた。
- 未知 error は raw `Error()` 文字列を分類値として返さず、未分類として caller に扱わせる。
