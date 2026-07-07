# security-reviewer: 2026-07-07-001

## レビュー範囲

- `.codex/agents/security-reviewer.toml` に定義された security-reviewer として、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、`AGENTS.md`、`README.md`、`docs/START_HERE.md`、主要な `docs/*.md`、`docs/ai/output/human/*.md`、同一 cycle の `planner.md` / `implementer.md` を確認した。
- repo-local skill `.agents/skills/banking-security-review/SKILL.md` を参照し、認証、認可、入力検証、ログ、秘密情報、SQL injection、権限境界、監査証跡の観点でレビューした。
- agent 間の直接同期は行わず、入力は `docs/ai/cycles/2026-07-07-001/` の成果物、human notes、現在の実装差分に限定した。
- 直近実装差分は `HEAD~1..HEAD` として確認し、`internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`README.md`、`docs/security-notes.md`、`docs/ai/cycles/2026-07-07-001/implementer.md` を優先レビューした。
- 今回の書き込みは本ファイル `docs/ai/cycles/2026-07-07-001/security-reviewer.md` のみに限定した。

## Finding

### 1. Blocking finding なし

- 重大度: なし
- 今回差分は、unknown non-nil error を raw error message ではなく `internal_error` へ寄せる audit / safe structured log 用 fallback helper、テスト、README / security docs の説明追加に限定されている。
- 新規 HTTP route、handler、request / response schema、DB 接続、SQL、repository、migration、認証、認可、Cookie / CSRF、監査ログ永続化、外部通信、秘密情報の読み書きは追加されていない。
- `FailureReasonFromError` は既知 domain sentinel error のみを固定分類へ写像し、未知 error と `nil` は従来どおり `"", false` のままにしている。
- `SafeFailureReasonFromError` は、既知 domain error では同じ固定分類を返し、未知 non-nil error では `FailureReasonInternalError` / `internal_error` を返すため、helper 自体が `err.Error()`、raw request body、password、token、secret、CSRF token、session ID、未加工の自由入力値を failure category として返す挙動は確認しなかった。
- `FailureReason.Validate()` は allow-list に `internal_error` を追加しつつ、空文字、未知値、既存 sentinel error の raw `Error()` 文字列、secret 風文字列を拒否する設計を維持している。
- README と `docs/security-notes.md` は、今回の helper を監査ログ / safe structured log 用の分類候補として説明し、public API response / HTTP status、監査ログ永続化、DB schema は未実装であることを維持している。

### 2. 複合 error の分類優先順は service / audit 実装前に明文化が必要

- 重大度: Low（現時点では service / DB / audit 永続化が未実装のため非 blocking）
- `SafeFailureReasonFromError` は内部で `FailureReasonFromError` を先に呼ぶため、未知 error と既知 domain error が同じ error chain / joined error に含まれる場合、既知 domain error の分類が優先される。
- 追加テストでも「既知 1 件 + unknown 1 件」の `errors.Join` は既知分類へ写像されることを確認している。
- この挙動は accepted scope の範囲では妥当で、raw unknown error message を出さないという主目的には合っている。一方で、将来 service / repository が domain validation error と DB / infrastructure error を安易に join すると、監査 `failure_reason` が `internal_error` ではなく `account_not_active` や `invalid_amount` のような業務分類になり、内部障害・攻撃兆候・運用事故の検知粒度が下がる可能性がある。
- 現時点では DB / repository / service が存在せず、複合 error の運用分類仕様も未確定なので blocking ではないが、監査ログ実装前に「domain validation error と infrastructure error を同一 failure として join するか」「複合時に `internal_error` を優先する条件があるか」「追加 metadata / correlation ID で内部障害を別途観測するか」を決める必要がある。

## 根拠

- `git status --short` は作業開始時点で空であり、未コミットのユーザー変更は確認しなかった。
- `git log --oneline --decorate -12` で直近 commit が `16c1ec8 Add safe failure fallback reason` であることを確認した。
- `git diff --name-status HEAD~1..HEAD` / `git diff --stat HEAD~1..HEAD` で、直近実装差分が `README.md`、`docs/ai/cycles/2026-07-07-001/implementer.md`、`docs/security-notes.md`、`internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go` に限定されていることを確認した。
- `git diff --find-renames HEAD~1..HEAD -- ...` で、`FailureReasonInternalError`、`SafeFailureReasonFromError`、テスト、README / security docs の説明を確認した。
- `internal/domain/failure_reason.go` の確認結果:
  - `FailureReasonInternalError = "internal_error"` が追加されている。
  - `FailureReason.Validate()` は固定分類値のみを allow-list で許可している。
  - `FailureReasonFromError` は既知 domain sentinel error の mapping を維持し、default では `"", false` を返している。
  - `SafeFailureReasonFromError` は既知 domain error を既存 helper に委譲し、`nil` は未分類、未知 non-nil error は `FailureReasonInternalError, true` を返している。
  - コメントで、audit `failure_reason` / safe structured log 向けであり、`err.Error()`、raw request body、secret 等を保存・露出しないこと、public API response / HTTP status の最終契約ではないことを明記している。
- `internal/domain/failure_reason_test.go` の確認結果:
  - supported reason validation に `internal_error` が追加されている。
  - unsafe / unknown reason rejection は維持され、secret 風値が invalid のままである。
  - `FailureReasonFromError` は未知 error / `nil` を未分類のままにするテストを維持している。
  - `SafeFailureReasonFromError` は既知 domain error、wrapped domain error、既知 1 件 + unknown 1 件の joined error、`nil`、未知 non-nil error をテストしている。
  - unknown error のテストには secret 風文字列が含まれるが、production code への secret 出力ではなく、raw error message が返らないことを確認する unit test 上の固定文字列である。
- `docs/security-notes.md` は、未知 non-nil error を raw error message ではなく `internal_error` のような固定分類へ寄せる方針、`internal_error` に DB 接続文字列・SQL・stack trace・request body・password・token・secret・CSRF token・セッションID・個人情報・未加工自由入力値を保存 / 返却しない方針、public API response は別設計とする方針を追記している。
- `README.md` は、`SafeFailureReasonFromError` が audit / safe structured log 用 fallback helper であり、利用者向け HTTP error response / status code、監査ログ永続化、DB schema が未実装であることを説明している。
- `rg -n "err\.Error\(\)|SafeFailureReasonFromError|FailureReasonFromError|FailureReasonInternalError|internal_error|password|token|secret|CSRF|session|request body" internal/domain README.md docs/security-notes.md` で、unknown error の raw message を failure category として返す実装はなく、secret / raw request body を保存・返却しない説明になっていることを確認した。
- `go test ./...` は成功した。
- `git diff --check HEAD~1..HEAD` は成功した。

## 影響

- 今回差分により、後続の監査ログ / safe structured log 実装で未知 error の `err.Error()` をそのまま `failure_reason` に保存・返却するリスクは下がる。
- `internal_error` が allow-list に入ったことで、未知 non-nil error を安全な固定分類として扱う入口ができ、DB 接続文字列、SQL、stack trace、request body、password、token、secret、CSRF token、session ID、未加工の自由入力値を分類値として混入させる事故を防ぎやすくなる。
- 既存 `FailureReasonFromError` の semantics が維持されたため、既知 domain error だけを分類したい caller と、unknown error を safe fallback したい audit / safe log caller を分けられる。
- ただし、この helper は認証、認可、CSRF、owner / role authorization、監査ログ永続化、DB transaction、冪等性、SQL injection 対策そのものを実装するものではない。業務 API 追加時に helper の存在だけで安全と誤解すると、権限境界や監査証跡が未完成のまま公開されるリスクがある。
- 複合 error の分類優先順を未定義のまま service / repository 実装へ進むと、内部障害を含む failure が業務 domain 分類として監査され、運用監視・インシデント調査時に内部障害の兆候を見落とすリスクがある。

## 推奨修正

- 今回差分への blocking 修正は不要。
- 次に service / repository / audit writer を実装する前に、複合 error の扱いを小さく設計する。
  - domain validation error と infrastructure / DB error を同じ error に join してよい場面、してはいけない場面を明記する。
  - join された error に known domain error と unknown infrastructure error が同居する場合、監査 `failure_reason` を domain 分類にするか `internal_error` にするか、または `failure_reason` と別の safe metadata で内部障害を示すかを決める。
  - `errors.Join` の複数 sentinel 優先順位を外部仕様として扱うか、実装詳細に留めるかを決める。
- HTTP handler / public API response 実装前に、public API 用 error code / message / HTTP status と audit `failure_reason` を分離して設計する。
  - `internal_error` は監査 / safe structured log 用 fallback であり、利用者へ必ず返す code / message とは限らないことを維持する。
  - `balance_overflow`、`invalid_balance_state` など内部状態寄りの分類を外部 response に出すか、抽象化するかを決める。
- 監査ログ実装時は、`SafeFailureReasonFromError` の返値だけでなく、safe な correlation ID、request ID、operation type、actor、target、result を組み合わせ、raw error message や raw request body を保存しない設計を維持する。
- 入金・出金・振込 handler / service を追加する前に、security gate 順序を docs 化する。
  1. request size / content type / schema validation。
  2. 認証済み actor の確認。
  3. Cookie 利用時の CSRF token 検証。
  4. owner / role authorization。
  5. 口座存在・口座状態 gate（`EnsureAccountCanTransact` は認可ではなく状態確認）。
  6. 金額・取引種別 validation。
  7. 冪等性キー検証。
  8. PostgreSQL transaction / row lock / 残高計算 / 取引履歴作成。
  9. 成功監査ログは成功時 transaction に含め、失敗監査ログは rollback 後の独立 transaction で保存する。
- SQL / DB 実装時は、failure category を SQL 文字列連結に使わず、DB 書き込みは parameterized query / prepared statement 相当の安全な境界を使う。

## 次サイクル planner への入力

1. 優先候補: 複合 error と監査分類の最小方針を docs 化する。
   - known domain error と unknown infrastructure error が同居した場合の `failure_reason` 優先順を決める。
   - `internal_error` を使う条件、domain 分類を使う条件、safe metadata / correlation ID で内部障害を補足する条件を分ける。
   - raw `err.Error()`、SQL、DB 接続文字列、stack trace、request body、password、token、secret、CSRF token、session ID を保存・返却しないルールを維持する。
2. 優先候補: 入金・出金・振込 service 前の security gate 順序を docs 化する。
   - 認証、CSRF、owner / role authorization、口座ステータス、入力 validation、冪等性、行ロック、残高計算、成功 / 失敗監査ログの順序を固定する。
   - `EnsureAccountCanTransact` は認可ではなく口座状態 gate、`SafeFailureReasonFromError` は監査 / safe structured log 用 fallback と位置づける。
3. 優先候補: public API error response と audit `failure_reason` の mapping を分離設計する。
   - public API では利用者向け message / HTTP status / code を別契約にする。
   - audit では固定分類と safe metadata を使い、未知 error で raw message fallback しない。
4. 優先候補: 失敗監査ログの最小 schema / repository 方針を検討する。
   - `failure_reason` の enum / CHECK constraint 候補、`request_body_hash`、`ip_address` / `user_agent` の最大長・正規化、対象未確定時の `target_id` null 許容を整理する。
   - 失敗監査ログを業務 transaction rollback 後の独立 transaction で残す方針を、成功監査ログの transaction 境界と分けて扱う。
5. 継続保留: Cookie session + CSRF token の最小認証方針を docs 化する。
   - 認証 cookie 属性、CSRF token の送受信方法、ログアウト、session ID をログ / 監査ログへ保存しない方針を決める。
