# security-reviewer: 2026-06-28-004

## レビュー範囲

- cycle-id: `2026-06-28-004`
- レビュー種別: repo-wide review
- 理由: `git status --short` の開始時点では実装差分なし。作業中に同一 cycle の `implementer.md` が追加されたが、内容は `blocked: accepted scope not found` であり、ソースコード、README、設計文書、テスト、DB schema、migration の変更はない。
- 参照した主な入力:
  - `AGENTS.md`
  - `.codex/agents/README.md`
  - `docs/ai/cycles/README.md`
  - `docs/START_HERE.md`
  - `docs/design-principles.md`
  - `docs/domain-model.md`
  - `docs/data-model.md`
  - `docs/use-cases.md`
  - `docs/security-notes.md`
  - `docs/test-strategy.md`
  - `docs/ai/cycles/2026-06-28-001/security-reviewer.md`
  - `docs/ai/cycles/2026-06-28-002/security-reviewer.md`
  - `docs/ai/cycles/2026-06-28-003/security-reviewer.md`
  - `docs/ai/cycles/2026-06-28-004/implementer.md`
  - `README.md`
  - `cmd/server/main.go`
  - `internal/httpapi/router.go`
  - `cmd/server/main_test.go`
  - `internal/httpapi/router_test.go`

## 前提

- 現在の実装済み API は `GET /healthz` の固定レスポンスのみ。
- PostgreSQL 接続、DB schema、認証、認可、顧客、口座、入出金、振込、残高照会、取引履歴照会、監査ログ、冪等性キー処理は未実装。
- `cmd/server/main.go` は既定で `127.0.0.1:8080` に待ち受け、`BANK_SYSTEM_HTTP_ADDR` による明示的な override と HTTP timeout を持つ。
- 本レビューは学習用ミニバンキングシステムとしての security review であり、本番金融システム相当の適合性を保証しない。

## Finding

### Finding 1: 業務 API 追加前の認証・認可仕様がまだ実装可能な粒度まで確定していない

- 重大度: High
- 人間確認要否: あり

#### 根拠

- `docs/design-principles.md` と `docs/security-notes.md` は、認証済みユーザーのみが業務データへアクセスし、顧客は自分に紐づく口座だけ操作できると定めている。
- `docs/mvp.md` はユーザー登録、認証、残高照会、取引履歴照会、監査ログ確認を MVP に含めている。
- 一方で、Cookie session か Bearer token か、CSRF 方針、パスワードハッシュ方式、ログアウト・失効、ログイン失敗時の rate limit、最小 RBAC 表、管理者と運用担当者の権限分離は未確定である。
- 現在の実装は `/healthz` のみで、認証・認可 middleware や業務 API はまだ存在しない。

#### 影響

- 業務 API を先に追加すると、各 handler が個別判断で認可を実装し、水平権限不備や過大な管理者権限が入りやすい。
- 残高照会、取引履歴照会、入出金、振込、監査ログ照会はすべて機微な金融操作であり、認証・認可境界が後付けになると修正範囲が大きい。

#### 推奨修正

- 次 cycle で、実装前の docs scope として以下を小さく確定する。
  - MVP の認証方式、パスワードハッシュ方式、セッションまたは token の有効期限・失効・ログアウト方針。
  - Cookie を使う場合の Secure / HttpOnly / SameSite / CSRF 方針。
  - `customer`、`admin`、必要なら `operator` の最小権限マトリクス。
  - URL や request body の `account_id` を信用せず、DB 上の所有関係で水平権限を検証する方針。

#### 次サイクル planner への入力

- accepted scope 候補: 「MVP 認証方式と RBAC / 水平権限チェック方針を `docs/security-notes.md` または新規設計文書に具体化する」。
- 業務 API 実装は、この scope が完了するまで採択しないことを推奨する。

### Finding 2: 入力検証、エラー応答、ログマスキングの共通標準が不足している

- 重大度: Medium
- 人間確認要否: 一部あり

#### 根拠

- `docs/security-notes.md` は、金額は正の整数、口座番号やログイン ID の形式検証、検索条件の上限、SQL やログへのユーザー入力の直接埋め込み禁止を挙げている。
- `docs/test-strategy.md` も、不正な入力値で server error ではなく検証エラーになること、パスワードや token がレスポンスやログに出ないことを求めている。
- ただし、REST API 全体で使う request body size limit、金額・口座番号・login_id・email・検索期間・page size の具体制約、エラー応答形式、認証失敗と認可失敗と存在しない resource の情報露出方針、request body 丸ごとログ禁止ルールは未定義である。

#### 影響

- API 実装時に validation が handler ごとに分散し、検証漏れ、情報過多なエラー、ログへの機微情報混入、一覧・検索 API の大量取得による可用性低下が起きやすい。
- PostgreSQL 導入後に SQL 組み立て方針が曖昧だと、SQL injection 防止が reviewer の後追い確認に依存する。

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

#### 推奨修正

- MVP では過剰な仕組みにせず、最低限として以下を文書化する。
  - 監査対象操作と、成功・失敗の両方を記録する範囲。
  - 残高変更系で監査ログ書き込みに失敗した場合の扱い。
  - password、token、生の認証 header、秘密鍵、request body 全文を監査ログに保存しないルール。
  - 監査ログの追記型・削除禁止の原則と、閲覧可能ロール。

#### 次サイクル planner への入力

- accepted scope 候補: 「監査ログ境界、失敗時扱い、マスキング、閲覧権限を MVP 設計に追加する」。
- 人間確認事項: 監査ログ書き込み失敗時に、金融系の重要操作を fail closed するか。

### Finding 4: 現在の HTTP skeleton 自体には新規の High / Medium セキュリティ問題は確認されない

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
2. `admin` が顧客口座の入金・出金・振込を代行できるか、また `operator` を MVP に含めるか。
3. 監査ログ書き込み失敗時に、残高変更や権限変更を失敗させるか。
4. 監査ログに IP address、User-Agent、request id、対象口座、失敗理由をどこまで残し、どの項目をマスクするか。
5. health / readiness / metrics endpoint を完全公開にするか、詳細情報を内部 network または認証済みに限定するか。

## 次サイクル planner への入力まとめ

- 最優先候補:
  1. MVP 認証方式と RBAC / 水平権限チェック方針の docs 化。
  2. API 入力検証、エラー応答、SQL injection 防止、ログマスキング方針の docs 化。
  3. 監査ログ境界、失敗時扱い、閲覧権限、マスキング方針の docs 化。
- 業務 API、DB schema、残高更新、振込、冪等性の実装は、上記の security design を少なくとも一部具体化してから小さく進めることを推奨する。

## テスト・確認結果

- `git status --short`: 開始時点では実装差分なし。作業中に `docs/ai/cycles/2026-06-28-004/implementer.md` が作成されたが、内容は実装 blocked の artifact であり、実装差分ではない。
- `git diff --stat`: 実装差分なし。
- `go test ./...`: 実行不可。現在の実行環境では `go` コマンドが見つからなかった。
- 本レビューで書き込んだファイルは `docs/ai/cycles/2026-06-28-004/security-reviewer.md` のみ。
