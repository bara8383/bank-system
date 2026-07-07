# security-reviewer: 2026-07-06-001

## レビュー範囲

- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、`.agents/skills/banking-security-review/SKILL.md` を確認し、security-reviewer の成果物を本ファイルのみに限定した。
- `docs/START_HERE.md`、`README.md`、`docs/security-notes.md`、`docs/design-principles.md`、`docs/data-model.md`、`docs/ai/output/human/*.md`、同一 cycle の `planner.md` / `implementer.md` を必要範囲で確認した。
- 実装差分は `HEAD^..HEAD` を対象にし、`internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`README.md`、`docs/security-notes.md`、`docs/ai/cycles/2026-07-06-001/implementer.md` を優先レビューした。
- 観点は、認証、認可、入力検証、ログ、秘密情報、SQL injection、権限境界、監査証跡である。

## Finding

### 1. Blocking finding なし

- 重大度: なし
- 今回差分は `internal/domain` の pure Go helper、unit test、README / security docs / implementer artifact の更新に限定されており、新規 HTTP route、DB 接続、SQL、認証、認可、Cookie / CSRF、ログ出力、外部通信、秘密情報処理は追加されていない。
- `FailureReason.Validate()` は allow-list 方式で固定分類値のみを許可し、空文字、未知値、既存 sentinel error の raw `Error()` 文字列、secret 風の自由入力値を拒否している。
- `FailureReasonFromError(error)` は `errors.Is` で既知の domain sentinel error だけを固定分類へ写像し、`nil`、未知 error、対象外 error は未分類として返すため、今回 helper 自体が raw error message や request body を分類値として返す挙動は確認しなかった。

### 2. Public API と audit / structured log の分類粒度は後続 cycle で分離確認が必要

- 重大度: Low（今回差分では HTTP response 未実装のため非 blocking）
- `FailureReason` は将来の API response、audit `failure_reason`、安全な構造化ログで再利用し得る分類候補として追加されている。
- ただし、`balance_overflow` や `invalid_balance_state` は監査・運用調査向けには有用だが、将来そのまま利用者向け public API response に返すと、内部表現や永続化境界の異常に近い情報を必要以上に露出する可能性がある。
- 今回は HTTP error response / status code mapping が未実装であり、README でも未実装と明記されているため、現時点の直接リスクではない。

## 根拠

- 差分確認:
  - `git status --short` は作業開始時点で空であり、既存の未コミット変更は確認しなかった。
  - `git diff --stat HEAD^..HEAD` と `git diff --name-status HEAD^..HEAD` で、直近実装差分が accepted scope 内の 5 ファイルに限定されていることを確認した。
  - `git diff --find-renames HEAD^..HEAD -- internal/domain/failure_reason.go internal/domain/failure_reason_test.go README.md docs/security-notes.md` で、failure reason helper、テスト、README、security notes の内容を確認した。
- 入力検証:
  - `FailureReason.Validate()` は `switch` による allow-list で、`invalid_amount`、`invalid_balance_state`、`insufficient_balance`、`balance_overflow`、`invalid_account_status`、`account_not_active`、`invalid_transaction_type` のみを valid としている。
  - invalid 時は sentinel error `ErrInvalidFailureReason` を返し、自由入力値をそのまま valid category にしない。
- ログ / 秘密情報:
  - helper のコメントは raw request body、secret、token、session ID、未検証の自由入力を含めないことを明示している。
  - `docs/security-notes.md` は、domain error 由来の `failure_reason` に固定分類値を使い、raw request body、password、token、secret、CSRF token、セッションID、未加工の自由入力値を保存・返却しない方針を追記している。
- 認証 / 認可 / 権限境界:
  - 今回差分は domain helper であり、認証済み actor、CSRF、owner / role authorization、admin 代行範囲、監査ログ閲覧権限は実装していない。
  - `README.md` は、認証、認可、HTTP error response、監査ログ、DB schema、業務 API が未実装であることを維持している。
- SQL injection:
  - SQL 文字列、DB driver、repository、migration は追加されていないため、今回差分で新たな SQL injection surface は確認しなかった。
- テスト:
  - `go test ./...` が成功した。
  - `internal/domain/failure_reason_test.go` は supported category、unsafe / unknown category 拒否、既知 sentinel error mapping、wrapped error mapping、`nil` / unknown error 未分類、mapped reason の `Validate()` 成功を確認している。

## 影響

- 今回差分により、後続の API response、監査ログ、安全な構造化ログで domain sentinel error の raw `Error()` 文字列や未加工 input を流用するリスクは下がる。
- `errors.Is` 対応により、service 層で context を付けて wrap された domain error でも安全な固定分類へ寄せられるため、後続の監査 `failure_reason` が揺れるリスクも下がる。
- 未知 error を未分類にする設計は、DB error、network error、内部 error、秘密情報を含む可能性がある error message を誤って利用者応答や監査分類へ昇格させない点で妥当である。
- 一方で、この helper は認証、認可、CSRF、owner / role check、口座状態 gate、冪等性、DB transaction、成功 / 失敗監査ログ永続化を保証しない。後続 service / handler で helper だけを導入しても、権限境界や監査証跡は完成しない。
- 将来 `FailureReason` を public API response にそのまま出す場合は、監査向け分類と利用者向け error code / message の粒度差を再確認しないと、内部状態異常や実装寄り分類を外部に出しすぎる可能性がある。

## 推奨修正

- 今回差分への blocking 修正は不要。
- 後続で HTTP error response を実装する前に、次を明確に分ける。
  - public API 用の安全な error code / message / status code。
  - audit `failure_reason` 用の運用調査向け固定分類。
  - structured log 用の安全な分類と、ログに出してよい追加 metadata。
- `FailureReasonFromError` の結果を利用する caller は、`ok == false` の場合に `err.Error()`、raw request body、password、token、secret、CSRF token、session ID、未加工の自由入力値を代替 category として保存・返却しないことを handler / service 設計で明示する。
- 入金・出金・振込 service / handler を追加する前に、security gate の順序を docs 化する。
  1. request size / content type / schema validation。
  2. 認証済み actor の確認。
  3. Cookie 利用時の CSRF token 確認。
  4. owner / role authorization（`EnsureAccountCanTransact` と混同しない）。
  5. 口座存在・口座状態 gate。
  6. 金額・取引種別 validation。
  7. 冪等性キー検証。
  8. PostgreSQL transaction / row lock / 残高計算 / 取引履歴作成。
  9. 成功監査ログは成功時 transaction に含め、失敗監査ログは rollback 後の独立 transaction で保存する。
- SQL / DB 実装時は、failure category を SQL 文字列連結に使わず、DB 書き込みは parameterized query / prepared statement 相当の安全な境界を使う。

## 次サイクル planner への入力

1. 優先候補: 入金・出金・振込 service 前の security gate 順序を docs 化する。
   - 認証、CSRF、owner / role authorization、口座ステータス、入力 validation、冪等性、行ロック、残高計算、成功 / 失敗監査ログの順序を固定する。
   - `EnsureAccountCanTransact` は認可ではなく口座状態 gate、`FailureReasonFromError` は失敗分類 helper として位置づける。
2. 優先候補: public API error response と audit `failure_reason` の mapping を分けて設計する。
   - audit では今回の `FailureReason` constants を利用候補にする。
   - public API では `invalid_balance_state` や `balance_overflow` をそのまま返すか、より抽象的な `internal_error` / `invalid_request` に寄せるかを決める。
   - 未知 error の fallback で raw `err.Error()` を保存・返却しないルールを明記する。
3. 優先候補: 失敗監査ログの最小 schema / repository 方針を検討する。
   - `failure_reason` の enum / CHECK constraint 候補、`request_body_hash`、`ip_address` / `user_agent` の最大長・正規化、対象未確定時の `target_id` null 許容を整理する。
   - 失敗監査ログを独立 transaction で残す実装方針を、業務 transaction rollback と混同しないようにする。
4. 優先候補: Cookie session + CSRF token の最小認証方針を docs 化する。
   - 認証 cookie 属性、CSRF token の送受信方法、ログアウト、session ID をログ / 監査ログへ保存しない方針を決める。
5. 継続保留: `reversal` / 取消 / 組戻し / 訂正は MVP 初期の valid transaction type に含めない方針を維持し、扱う場合は権限、二重取消防止、監査、対象取引との関連を別 cycle で設計する。
