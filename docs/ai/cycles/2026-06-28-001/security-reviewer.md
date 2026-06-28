# security-reviewer: 2026-06-28-001

## レビュー範囲

- cycle-id: `2026-06-28-001`
- レビュー種別: repo-wide review
- 理由: `git status --short` と `git diff --stat` で、レビュー対象となる未コミット実装差分は確認されなかったため。
- 参照した主な入力:
  - `README.md`
  - `AGENTS.md`
  - `.codex/agents/README.md`
  - `docs/ai/cycles/README.md`
  - `.agents/skills/banking-security-review/SKILL.md`
  - `docs/START_HERE.md`
  - `docs/design-principles.md`
  - `docs/domain-model.md`
  - `docs/mvp.md`
  - `docs/data-model.md`
  - `docs/use-cases.md`
  - `docs/security-notes.md`
  - `docs/test-strategy.md`
  - `docs/decision-logs/2026-06-27-subagent-parallel-cycle.md`
  - `docs/ai/cycles/2026-06-28-001/`、`2026-06-28-002/`、`2026-06-28-003/`、`2026-06-28-004/` の Markdown 成果物
  - `cmd/server/main.go`
  - `internal/httpapi/router.go`
  - `cmd/server/main_test.go`
  - `internal/httpapi/router_test.go`

## 前提

- 現在の実装済み API は `GET /healthz` の固定レスポンスのみ。
- PostgreSQL 接続、DB schema、migration、認証、認可、ユーザー登録、顧客、口座、入出金、振込、残高照会、取引履歴照会、監査ログ、冪等性キー処理は未実装。
- `cmd/server/main.go` は既定で `127.0.0.1:8080` に待ち受け、`BANK_SYSTEM_HTTP_ADDR` による明示的な override と HTTP timeout を持つ。
- 本レビューは学習用ミニバンキングシステムとしての security review であり、本番金融システム相当の適合性を保証しない。

## Finding

### Finding 1: 業務 API 追加前の認証・認可仕様が実装可能な粒度まで確定していない

- 重大度: High
- 人間確認要否: あり

#### 根拠

- `docs/design-principles.md` と `docs/security-notes.md` は、認証済みユーザーだけが業務データへアクセスし、顧客ユーザーは自分に紐づく口座だけ操作できると定めている。
- `docs/mvp.md` はユーザー登録、認証、残高照会、取引履歴照会、監査ログ確認を MVP に含めている。
- 一方で、Cookie session か Bearer token か、CSRF 方針、パスワードハッシュ方式、ログアウト・失効、ログイン失敗時の rate limit、最小 RBAC 表、管理者と運用担当者の権限分離は未確定である。
- 現在の実装は `/healthz` のみで、認証・認可 middleware や業務 API はまだ存在しない。

#### 影響

- 業務 API を先に追加すると、各 handler が個別判断で認可を実装し、水平権限不備や過大な管理者権限が入りやすい。
- 残高照会、取引履歴照会、入出金、振込、監査ログ照会はすべて機微な金融操作であり、認証・認可境界が後付けになると修正範囲が大きい。

#### 攻撃/事故シナリオ

1. 顧客 A が URL や request body の `account_id` を顧客 B の口座 ID に差し替える。
2. handler が「ログイン済みか」だけを確認し、DB 上の口座所有関係を確認しない。
3. 顧客 A が顧客 B の残高・取引履歴を閲覧、または不正な出金・振込を実行できる。

#### 推奨修正

- 業務 API 実装前に、docs scope として以下を小さく確定する。
  - MVP の認証方式、パスワードハッシュ方式、セッションまたは token の有効期限・失効・ログアウト方針。
  - Cookie を使う場合の Secure / HttpOnly / SameSite / CSRF 方針。
  - `customer`、`admin`、必要なら `operator` の最小権限マトリクス。
  - URL や request body の `account_id` を信用せず、DB 上の所有関係で水平権限を検証する方針。

#### 次サイクル planner への入力

- accepted scope 候補: 「MVP 認証方式と RBAC / 水平権限チェック方針を `docs/security-notes.md` または新規設計文書に具体化する」。
- 業務 API 実装は、この scope が完了するまで採択しないことを推奨する。

### Finding 2: 入力検証、エラー応答、ログマスキング、SQL injection 防止の共通標準が不足している

- 重大度: Medium
- 人間確認要否: 一部あり

#### 根拠

- `docs/security-notes.md` は、金額は正の整数、口座番号やログイン ID の形式検証、検索条件の上限、SQL やログへのユーザー入力の直接埋め込み禁止を挙げている。
- `docs/test-strategy.md` は、不正な入力値で server error ではなく検証エラーになること、パスワードや token がレスポンスやログに出ないことを求めている。
- ただし、REST API 全体で使う request body size limit、金額・口座番号・login_id・email・検索期間・page size の具体制約、エラー応答形式、認証失敗と認可失敗と存在しない resource の情報露出方針、request body 丸ごとログ禁止ルールは未定義である。
- PostgreSQL repository 実装時に、placeholder / prepared statement / query builder の利用方針もまだ明文化されていない。

#### 影響

- API 実装時に validation が handler ごとに分散し、検証漏れ、情報過多なエラー、ログへの機微情報混入、一覧・検索 API の大量取得による可用性低下が起きやすい。
- SQL 組み立て方針が曖昧だと、口座番号、ログイン ID、検索条件、監査ログ検索などの入力を SQL 文字列へ直接連結する実装が入り、SQL injection の確認が後追いになる。

#### 攻撃/事故シナリオ

1. 取引履歴検索 API が、期間や件数上限なしで検索条件を受け付ける。
2. handler が入力値を SQL 文字列へ直接連結する、または validation なしでログに出力する。
3. SQL injection、取引履歴の大量取得、ログへの認証情報混入、内部エラー露出が発生する。

#### 推奨修正

- 業務 API 実装前に、API 入力検証とエラー応答の最小標準を docs 化する。
- SQL は placeholder を使う方針、検索・一覧系は期間と件数の上限を必須にする方針、ログには password / token / 認証 header / request body 全文 / 口座番号全文を出さない方針を明記する。
- 顧客向けエラーと運用調査向けログの情報量を分ける。

#### 次サイクル planner への入力

- accepted scope 候補: 「REST API 入力検証・エラー応答・SQL injection 防止・ログマスキング方針を設計文書化する」。
- 監査ログや request logging を実装する場合は、この標準と同じ cycle か先行 cycle で扱う。

### Finding 3: 監査ログの記録境界、失敗時扱い、改ざん耐性が未確定

- 重大度: Medium
- 人間確認要否: あり

#### 根拠

- `docs/design-principles.md` と `docs/security-notes.md` は、ログイン、顧客登録、口座作成、入金、出金、振込、権限変更などの成功・失敗を監査ログへ残す方針を示している。
- `docs/data-model.md` には `audit_logs` テーブル候補があり、実行者、操作種別、対象、結果、失敗理由、IP address、User-Agent を含める案がある。
- 一方で、監査ログ書き込み失敗時に業務処理を失敗させるか、残高変更系と同一 DB transaction に含めるか、監査ログの保存期間、削除禁止、閲覧権限、改ざん検知、マスキング規則は未確定である。

#### 影響

- 残高は変わったが監査ログがない、失敗した攻撃痕跡が残らない、監査ログに password / token / 過剰な個人情報が残る、といった事故につながる。
- 監査ログ照会 API を後から追加する際、管理者や運用担当者の閲覧範囲が過大になる可能性がある。

#### 攻撃/事故シナリオ

1. 不正なログイン試行や権限外の振込試行が失敗する。
2. 失敗時監査ログが業務 transaction の rollback と一緒に消える、またはそもそも記録対象外になる。
3. 攻撃調査時に、誰がどの対象へ何を試みたかを追跡できない。

#### 推奨修正

- MVP では過剰な仕組みにせず、最低限として以下を文書化する。
  - 監査対象操作と、成功・失敗の両方を記録する範囲。
  - 残高変更系で監査ログ書き込みに失敗した場合の扱い。
  - password、token、生の認証 header、秘密鍵、request body 全文を監査ログに保存しないルール。
  - 監査ログの追記型・削除禁止の原則と、閲覧可能ロール。

#### 次サイクル planner への入力

- accepted scope 候補: 「監査ログ境界、失敗時扱い、マスキング、閲覧権限を MVP 設計に追加する」。
- 人間確認事項: 監査ログ書き込み失敗時に、金融系の重要操作を fail closed するか。

### Finding 4: 冪等性キーと DB トランザクション境界が未具体化で、二重実行防止を DB 制約として担保する方針が不足している

- 重大度: Medium
- 人間確認要否: あり

#### 根拠

- `docs/design-principles.md` は、同じ振込依頼が複数回処理されないよう冪等性キーを持たせ、同じキーの処理結果を再利用するとしている。
- `docs/data-model.md` は `transfer_requests.idempotency_key` の一意制約案を持つが、依頼者単位か振込元口座単位か、同一キーで金額・振込先が異なる場合の扱い、保存期間、`processing` 再送時の扱いは未確定である。
- 残高更新、取引履歴作成、振込依頼更新、成功時監査ログの DB transaction 境界も実装可能な粒度までは定まっていない。

#### 影響

- アプリケーション層だけで冪等性を判定すると、並行リクエスト時に二重送金を許す可能性がある。
- 同一キー・異内容の再送を既存結果として返すと、利用者が意図しない振込結果を正常応答として受け取る。
- DB transaction 境界が曖昧だと、残高更新だけ成功し、取引履歴や振込依頼状態が欠ける事故につながる。

#### 攻撃/事故シナリオ

1. クライアントや攻撃者が同じ冪等性キーの振込リクエストを並行送信する。
2. DB の一意制約や transaction 内ロックなしにアプリケーション側で重複確認する。
3. 両方の処理が「未処理」と判断し、同じ振込が二重実行される。

#### 推奨修正

- 冪等性キーの一意スコープを `requested_by_user_id + source_account_id + idempotency_key` などに固定する。
- 同一キーで振込元、振込先、金額、通貨が異なる場合は conflict として拒否する。
- `transfer_requests` の一意制約と、残高更新・取引履歴・振込依頼状態更新を同一 DB transaction で扱う方針を docs に明記する。
- `processing` の再送時は二重実行せず、状態確認または安全な再試行方針を返す。

#### 次サイクル planner への入力

- accepted scope 候補: 「振込冪等性キーの一意スコープ、同一キー異内容時の扱い、DB 一意制約、成功時 transaction 境界を設計文書化する」。
- 人間確認事項: 冪等性キーの保存期間と、失敗済み依頼の再送可否。

### Finding 5: 現在の HTTP skeleton 自体には新規の High / Medium セキュリティ問題は確認されない

- 重大度: Informational
- 人間確認要否: なし

#### 根拠

- `cmd/server/main.go` は既定 listen address を `127.0.0.1:8080` とし、外部 interface で待ち受ける場合は `BANK_SYSTEM_HTTP_ADDR` の明示指定を必要とする。
- `cmd/server/main.go` は `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` を設定している。
- `internal/httpapi/router.go` の `/healthz` は `GET` のみを受け付け、`{"status":"ok"}` の固定 JSON だけを返す。
- `internal/httpapi/router_test.go` は、固定レスポンス、JSON content type、秘密情報・DB 情報・環境情報・内部 path らしき文字列の非露出を検証している。

#### 影響

- 現時点の実装範囲では、秘密情報漏えい、不正送金、水平権限不備、SQL injection、監査証跡欠落に直結する処理は存在しない。
- ただし `BANK_SYSTEM_HTTP_ADDR=:8080` による外部公開は可能であり、将来の業務 API 追加時には公開範囲と認証必須 endpoint の分類が必要である。

#### 推奨修正

- 現在の HTTP skeleton への修正必須事項はない。
- `/healthz` を将来 readiness 化する場合も、DB 接続文字列、環境変数、hostname、内部 path、stack trace、口座数、取引件数などを外部応答に含めない。
- 業務 API を同じ server に追加する前に、公開可能 endpoint と認証必須 endpoint を docs で分類する。

#### 次サイクル planner への入力

- 「health / readiness / metrics の公開範囲と情報露出ルール」を security design 候補として残す。
- コンテナや reverse proxy を導入する場合は、`127.0.0.1` 既定値と `:8080` 明示指定の意味を設計・README に同期する。

## 人間確認事項

1. MVP の認証方式を Cookie session + CSRF と Bearer token のどちらで学習・実装するか。
2. パスワードハッシュ方式と最低パラメータを何にするか。
3. `admin` が顧客口座の入金・出金・振込を代行できるか、また `operator` を MVP に含めるか。
4. 監査ログ書き込み失敗時に、残高変更や権限変更を失敗させるか。
5. 冪等性キーの一意スコープ、保存期間、同一キー異内容時の扱いをどうするか。
6. health / readiness / metrics endpoint を完全公開にするか、詳細情報を内部 network または認証済みに限定するか。

## 次サイクル planner への入力まとめ

- 最優先候補:
  1. MVP 認証方式と RBAC / 水平権限チェック方針の docs 化。
  2. API 入力検証、エラー応答、SQL injection 防止、ログマスキング方針の docs 化。
  3. 監査ログ境界、失敗時扱い、閲覧権限、マスキング方針の docs 化。
  4. 冪等性キーの一意スコープ、同一キー異内容時の扱い、DB transaction 境界の docs 化。
- 業務 API、DB schema、残高更新、振込、冪等性の実装は、上記の security design を少なくとも一部具体化してから小さく進めることを推奨する。

## テスト・確認結果

- `git status --short`: 開始時点では未コミット変更なし。
- `git diff --stat`: 実装差分なし。
- `go test ./...`: 実行不可。現在の実行環境では `go` コマンドが見つからなかった。
- 本レビューで書き込んだファイルは `docs/ai/cycles/2026-06-28-001/security-reviewer.md` のみ。
