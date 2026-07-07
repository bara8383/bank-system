# implementer output: 2026-07-07-001

## 参照した accepted scope

- `docs/ai/cycles/2026-07-07-001/planner.md` の accepted scope「unknown error 用の safe failure category と audit fallback helper」を参照した。
- 実装対象は `internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`README.md`、`docs/security-notes.md` に限定した。
- 非対象として明記された HTTP route / handler、public API error schema、監査ログ永続化、PostgreSQL、migration、repository、認証、認可、冪等性、業務 API、reversal は実装していない。

## 変更内容

- `internal/domain/failure_reason.go`
  - `FailureReasonInternalError = "internal_error"` を追加した。
  - `FailureReason.Validate()` の allow-list に `FailureReasonInternalError` を追加した。
  - 既存 `FailureReasonFromError(err error) (FailureReason, bool)` は、未知 error と `nil` を `"", false` にする既存 semantics を維持した。
  - 監査ログ / safe structured log 用 fallback helper として `SafeFailureReasonFromError(err error) (FailureReason, bool)` を追加した。
  - helper comment で、raw `err.Error()`、raw request body、secret 等を保存・返却しないこと、および public API response / HTTP status code の最終仕様ではないことを明記した。
- `internal/domain/failure_reason_test.go`
  - supported reason validation に `internal_error` を追加した。
  - unsafe / unknown reason rejection は維持し、secret-like value が invalid のままであることを確認した。
  - `FailureReasonFromError` の未知 error / `nil` 非分類テストを維持した。
  - `SafeFailureReasonFromError` のテストを追加し、既知 domain error、wrapped domain error、既知 1 件 + unknown 1 件の joined error、`nil`、未知 non-nil error の挙動を確認した。
- `README.md`
  - `FailureReasonFromError` と `SafeFailureReasonFromError` の用途差を追記した。
  - `SafeFailureReasonFromError` は unknown non-nil error を raw message ではなく `internal_error` へ寄せる audit / safe structured log 用 helper であり、HTTP error response / status code、監査ログ永続化、DB schema は未実装であることを維持した。
- `docs/security-notes.md`
  - 監査 `failure_reason` では既知 domain error を固定分類、未知 non-nil error を `internal_error` のような固定分類へ寄せる方針を追記した。
  - `internal_error` に DB 接続文字列、SQL、stack trace、request body、password、token、secret、CSRF token、セッションID、個人情報、未加工の自由入力値を保存・返却しないことを明記した。
  - public API response 用の code / message / HTTP status は別途設計し、audit `failure_reason` を利用者向け message として扱わない方針を明記した。

## scope 適合性

- accepted scope の主目的である「既存 `FailureReasonFromError` の semantics を維持しつつ、監査ログ / safe structured log 用に unknown non-nil error を safe category へ寄せる helper」を実装した。
- `nil` は failure として分類せず、`"", false` を返す挙動を維持した。
- 未知 non-nil error は raw message ではなく `FailureReasonInternalError` / `internal_error` に分類するようにした。
- README と security docs では、今回の helper が audit / safe structured log 用 fallback であり、public API response の最終仕様ではないことを明記した。
- 変更ファイルは accepted scope 内に限定した。

## 実装しなかったこと

- HTTP route、handler、request / response schema、status code mapping、public API error format は追加していない。
- 利用者向け message、i18n、error code 体系全体は確定していない。
- 監査ログ table、audit repository、audit service、監査ログ永続化、成功 / 失敗監査ログ transaction は実装していない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、row lock は作成していない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC、admin 代行処理は実装していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装していない。
- 取引履歴 row、`balance_after` 永続化、transfer request 状態遷移、冪等性キー処理、request body hash 算出は実装していない。
- `reversal` / 取消 / 組戻し / 訂正は実装していない。
- 複数 domain sentinel error を含む joined error の優先順位表は確定していない。テストは accepted scope に沿って「既知 1 件 + unknown 1 件」に限定した。
- `internal_error` を利用者へ必ず返す API code として固定していない。

## テスト結果

- `gofmt -w internal/domain/failure_reason.go internal/domain/failure_reason_test.go`: 成功。
- `go test ./...`: 成功。
- `git diff --check`: 成功。
- `rg -n "err\.Error\(\)|internal_error|FailureReasonInternalError|SafeFailureReasonFromError|password|token|secret|CSRF|session|request body" internal/domain README.md docs/security-notes.md`: 確認済み。raw unknown error message を failure category として返す実装はなく、secret / raw request body を保存・返却しない説明になっていることを確認した。
- `git diff --name-only`: 変更ファイルが `README.md`、`docs/security-notes.md`、`internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`docs/ai/cycles/2026-07-07-001/implementer.md` に限定されていることを確認した。

## 作業仮定

- `FailureReasonFromError` は「既知 domain error を分類できるか」を判定する低レベル helper として維持する。
- `SafeFailureReasonFromError` は audit / safe structured log 用 fallback helper とし、public API response body / HTTP status code の最終仕様ではない。
- unknown non-nil error は、監査 `failure_reason` として raw message を保存・返却せず、`internal_error` の固定分類へ寄せる。
- `internal_error` の詳細調査には、将来 correlation ID、安全な構造化ログ、運用ログ保管場所を別途設計して使う。今回の helper は詳細文字列を扱わない。
- `nil` は failure ではないため `"", false` とする。
- `errors.Join` については、既知 domain error 1 件と unknown error 1 件を含む場合に既知分類へ写像できることだけを確認し、複数 domain sentinel error 間の優先順位は外部仕様として確定しない。
