# security-reviewer: 2026-06-28-005

## レビュー範囲

- cycle-id: `2026-06-28-005`
- レビュー種別: repo-wide review
- 理由:
  - 作業開始時の `git status --short` は空だった。
  - 作業中に同一 cycle の `docs/ai/cycles/2026-06-28-005/implementer.md` が追加されたが、内容は `blocked: accepted scope not found` であり、Go ソースコード、README、設計文書、テスト、DB schema、migration の変更はない。
  - `git diff --stat` は空で、実装差分は確認できなかった。
- 参照した主な入力:
  - `AGENTS.md`
  - `.codex/agents/README.md`
  - `docs/ai/cycles/README.md`
  - `.agents/skills/banking-security-review/SKILL.md`
  - `.agents/skills/banking-security-review/references/banking-quality-rubric.md`
  - `README.md`
  - `docs/START_HERE.md`
  - `docs/design-principles.md`
  - `docs/domain-model.md`
  - `docs/data-model.md`
  - `docs/use-cases.md`
  - `docs/mvp.md`
  - `docs/security-notes.md`
  - `docs/test-strategy.md`
  - `docs/decision-logs/2026-06-27-subagent-parallel-cycle.md`
  - `docs/ai/output/human/`: ディレクトリなし。追加の human notes は確認できなかった。
  - `docs/ai/cycles/2026-06-28-001/` から `2026-06-28-004/` の cycle 成果物
  - `docs/ai/cycles/2026-06-28-005/implementer.md`
  - `docs/ai/cycles/2026-06-28-005/code-reviewer.md`
  - `docs/ai/cycles/2026-06-28-005/banking-reviewer.md`
  - `cmd/server/main.go`
  - `internal/httpapi/router.go`
  - `cmd/server/main_test.go`
  - `internal/httpapi/router_test.go`

## 前提

- 現在の実装済み API は `GET /healthz` の固定レスポンスのみ。
- PostgreSQL 接続、DB schema、認証、認可、ユーザー登録、顧客、口座、入出金、振込、残高照会、取引履歴照会、監査ログ、冪等性キー処理は未実装。
- `cmd/server/main.go` は既定で `127.0.0.1:8080` に待ち受け、`BANK_SYSTEM_HTTP_ADDR` により明示的に listen address を変更できる。
- 本レビューは学習用ミニバンキングシステムとしてのセキュリティレビューであり、本番金融システム相当の適合性を保証しない。

## Finding

### Finding 1: 業務 API 追加前の認証・認可・権限境界が実装可能な粒度まで確定していない

- 重大度: High
- 人間確認要否: あり

#### 根拠

- `docs/design-principles.md` は、口座、残高、取引履歴、振込依頼などの操作を認証前提とし、顧客ユーザーは自分に紐づく口座のみ操作できるとしている。
- `docs/security-notes.md` は、パスワードハッシュ、セッションまたは token の有効期限、管理者と運用担当者の権限分離、口座 ID 指定時の所有者確認を求めている。
- `docs/mvp.md` は、ユーザー登録、認証、残高照会、取引履歴照会、監査ログ記録を MVP に含めている。
- 一方で、Cookie session か Bearer token か、CSRF 方針、パスワードハッシュ方式、ログアウト・失効、ログイン失敗時の rate limit、最小 RBAC 表、管理者と運用担当者の権限分離は未確定である。
- 現在の Go 実装には、認証 middleware、actor 抽出、認可 helper、業務 API は存在しない。

#### 影響

- 認証・認可の共通方針なしに業務 API を追加すると、handler ごとの個別判断になり、水平権限不備が入りやすい。
- 口座 ID や顧客 ID を request から受け取る API で、DB 上の所有関係を確認しない実装が入ると、顧客 A が顧客 B の残高・取引履歴を参照または操作できる事故につながる。
- 管理者ロールを広く作りすぎると、入出金、振込、監査ログ閲覧、権限変更が単一権限に集中し、内部不正や誤操作の影響が大きくなる。

#### 推奨修正

- 業務 API 実装前に、MVP の認証方式と権限境界を docs 化する。
- 最低限、以下を明記する。
  - パスワードハッシュ方式と最低パラメータ。
  - Cookie session または Bearer token の選択、有効期限、失効、ログアウト方針。
  - Cookie を使う場合の Secure / HttpOnly / SameSite / CSRF 方針。
  - `customer`、`admin`、必要なら `operator` の最小権限表。
  - URL や request body の `account_id` を信用せず、DB 上の所有関係とロールで認可する方針。

#### 次サイクル planner への入力

- accepted scope 候補: 「MVP 認証方式、パスワードハッシュ、セッション/token、RBAC、水平権限チェック方針を設計文書化する」。
- 業務 API 実装は、この security design が少なくとも一部具体化されるまで採択しないことを推奨する。

### Finding 2: 入力検証、エラー応答、SQL injection 防止、ログマスキングの共通標準が不足している

- 重大度: Medium
- 人間確認要否: 一部あり

#### 根拠

- `docs/security-notes.md` は、金額は正の整数、口座番号やログイン ID の形式検証、検索条件の上限、SQL やログへのユーザー入力の直接埋め込み禁止を挙げている。
- `docs/test-strategy.md` は、不正な入力値で server error ではなく検証エラーになること、パスワードや token がレスポンスやログに出ないことを求めている。
- しかし、REST API 全体で使う request body size limit、金額・口座番号・login_id・email・検索期間・page size の具体制約、エラー応答形式、認証失敗と認可失敗と対象なしの情報露出方針、request body 全文ログ禁止ルールは未定義である。
- 現在は SQL 実装が存在しないため SQL injection は発火しないが、PostgreSQL 導入時の placeholder 使用方針や動的検索条件の組み立て方針もまだ具体化されていない。

#### 影響

- API 実装時に validation が handler ごとに分散し、検証漏れ、情報過多なエラー、ログへの機微情報混入が起きやすい。
- 取引履歴や監査ログの検索 API で期間・件数上限がないと、大量取得による可用性低下や過剰な金融情報露出につながる。
- SQL の組み立て方針が曖昧なまま repository を追加すると、文字列連結による SQL injection が混入しやすい。

#### 推奨修正

- 業務 API 実装前に、API 入力検証とエラー応答の最小標準を docs 化する。
- SQL は placeholder を使う方針、検索・一覧系は期間と件数の上限を必須にする方針を明記する。
- ログには password、token、認証 header、request body 全文、口座番号全文、過剰な個人情報を出さない方針を明記する。
- 顧客向けエラーと運用調査向けログの情報量を分ける。

#### 次サイクル planner への入力

- accepted scope 候補: 「REST API 入力検証・エラー応答・SQL injection 防止・ログマスキング方針を設計文書化する」。
- 認証設計と同時に扱う場合は、エラー応答でログイン ID の存在有無や口座所有関係を露出しない方針も含める。

### Finding 3: 監査ログの記録境界、失敗時証跡、閲覧権限、改ざん耐性が未確定

- 重大度: Medium
- 人間確認要否: あり

#### 根拠

- `docs/design-principles.md` と `docs/security-notes.md` は、ログイン、顧客登録、口座作成、入金、出金、振込、権限変更などの成功・失敗を監査ログへ残す方針を示している。
- `docs/data-model.md` の `audit_logs` 候補には、実行者、操作種別、対象、結果、失敗理由、IP address、User-Agent が含まれる。
- 一方で、監査ログ書き込み失敗時に業務処理を失敗させるか、残高変更系と同一 DB transaction に含めるか、業務拒否やシステム障害の失敗ログを rollback 後も残すか、保存期間、削除禁止、閲覧権限、改ざん検知、マスキング規則は未確定である。

#### 影響

- 残高は変わったが監査ログがない、失敗した攻撃・不正操作の痕跡が rollback で消える、監査ログに password / token / 過剰な個人情報が残る、といった事故につながる。
- 監査ログ照会 API を後から追加する際、管理者や運用担当者の閲覧範囲が過大になり、監査ログ自体が機微情報の漏えい源になる可能性がある。

#### 推奨修正

- MVP では過剰な仕組みにせず、最低限として以下を文書化する。
  - 監査対象操作と、成功・失敗の両方を記録する範囲。
  - 残高変更を伴う成功操作で、残高更新・取引履歴・振込依頼状態・成功監査ログをどう整合させるか。
  - 業務拒否やシステム障害の失敗ログを、業務 transaction rollback 後でも残す必要があるか。
  - password、token、生の認証 header、秘密鍵、request body 全文を監査ログに保存しないルール。
  - 監査ログの追記型・削除禁止の原則と、閲覧可能ロール。

#### 次サイクル planner への入力

- accepted scope 候補: 「監査ログ境界、失敗時扱い、マスキング、閲覧権限を MVP 設計に追加する」。
- 人間確認事項: 監査ログ書き込み失敗時に、残高変更や権限変更を fail closed するか。

### Finding 4: `BANK_SYSTEM_HTTP_ADDR` による外部公開と health / readiness / metrics の公開範囲ルールが未整理

- 重大度: Low
- 人間確認要否: あり

#### 根拠

- `cmd/server/main.go` は既定 listen address を `127.0.0.1:8080` にしているため、通常のローカル実行では安全側に倒れている。
- 一方で、`BANK_SYSTEM_HTTP_ADDR=:8080` を指定すると外部 interface で待ち受けられる。
- `README.md` は、外部 interface で待ち受ける場合は将来の業務 API 追加前に認証・認可・公開範囲を確認するよう注意している。
- 現在の `/healthz` は固定レスポンスのみだが、将来 readiness や metrics を追加する場合の公開範囲、詳細情報の出し分け、認証要否は未確定である。

#### 影響

- 現時点では `/healthz` しかなく、秘密情報や業務データを返さないため直接影響は低い。
- 将来、同じ server に未認証または認可不十分な業務 API を追加した状態で `BANK_SYSTEM_HTTP_ADDR=:8080` を使うと、開発環境やコンテナ環境から業務 API が意図せず到達可能になる可能性がある。
- readiness / metrics が DB 状態、内部 path、接続先、口座数、取引件数などを返すと、情報露出につながる。

#### 推奨修正

- 業務 API、DB readiness、metrics を追加する前に、公開可能 endpoint と認証必須 endpoint を分類する。
- `/healthz` は固定の liveness に限定し、詳細 readiness は内部 network 限定または認証済みにするかを人間確認する。
- コンテナや reverse proxy を導入する場合は、`127.0.0.1` 既定値と `:8080` 明示指定の意味を README または設計文書に同期する。

#### 次サイクル planner への入力

- accepted scope 候補: 「health / readiness / metrics の公開範囲と情報露出ルールを security design として整理する」。
- 業務 API 実装 cycle では、`BANK_SYSTEM_HTTP_ADDR=:8080` が未認証 API 公開につながらないかを checklist に入れる。

### Finding 5: 現在の HTTP skeleton 自体には新規の High / Medium セキュリティ問題は確認されない

- 重大度: Informational
- 人間確認要否: なし

#### 根拠

- `cmd/server/main.go` は既定 listen address を `127.0.0.1:8080` とし、外部 interface で待ち受ける場合は `BANK_SYSTEM_HTTP_ADDR` の明示指定を必要とする。
- `cmd/server/main.go` は `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` を設定している。
- `internal/httpapi/router.go` の `/healthz` は `GET` のみを受け付け、固定 JSON `{"status":"ok"}` だけを返す。
- `internal/httpapi/router_test.go` は、固定レスポンス、JSON content type、秘密情報・DB 情報・環境情報・内部 path らしき文字列の非露出を検証している。
- repository 内に SQL 実装、認証情報処理、secret 読み込み、業務データ処理は存在しない。

#### 影響

- 現在の実装範囲では、秘密情報漏えい、不正送金、水平権限不備、SQL injection、監査証跡欠落に直結する処理は存在しない。
- ただし、金融業務 API が未実装であるため、システム全体として認証済み・認可済みの銀行操作を安全に実行できる段階ではない。

#### 推奨修正

- 現在の HTTP skeleton への修正必須事項はない。
- `/healthz` を将来 readiness 化する場合も、DB 接続文字列、環境変数、hostname、内部 path、stack trace、口座数、取引件数などを外部応答に含めない。
- 次に実装へ進む場合は、業務 API ではなく、認証/RBAC または API 入力・ログ・監査方針の docs scope を優先することを推奨する。

#### 次サイクル planner への入力

- Go skeleton / health check / HTTP server hardening は一旦完了扱いでよい。
- 次 cycle は、業務 API 追加前の security design から 1 つに絞って accepted scope を作る。

### Finding 6: cycle 004 の元帳・成功時 transaction 境界 docs が未反映で、監査ログ境界の前提も固まっていない

- 重大度: Medium
- 人間確認要否: あり

#### 根拠

- 同一 cycle の `code-reviewer.md` と `banking-reviewer.md` は、cycle 004 planner が採択した「元帳・残高方向・成功時 DB transaction 境界」の docs 更新が未実装のまま残っていると指摘している。
- `docs/design-principles.md` は残高変更と取引履歴、振込の原子性、監査ログを原則としているが、入金・出金・振込それぞれで成功時監査ログを同一 DB transaction に含めるかは未確定である。
- `docs/data-model.md` は `transactions.transaction_type` と `audit_logs` の候補を持つが、残高変更・取引履歴・振込依頼状態・成功監査ログの整合性境界、失敗監査ログの rollback 外保存方針は未確定である。

#### 影響

- 元帳・残高方向が曖昧なまま監査ログ設計に進むと、どの残高変更に対してどの監査ログを必須にするかが実装者依存になる。
- 残高更新と取引履歴は同一 transaction に入っても、成功監査ログだけ別 transaction になると、残高は変わったが監査ログが欠落する状態が起き得る。
- 逆に、失敗監査ログを業務 transaction 内だけに置くと、rollback により権限外操作や攻撃試行の証跡が消える可能性がある。

#### 推奨修正

- cycle 004 の元帳・残高方向・成功時 transaction 境界 docs 更新を再採択する場合、security 観点として「監査ログ境界は未確定であり、人間確認事項として分離する」ことを明記する。
- 残高変更を伴う成功操作では、残高更新・取引履歴・振込依頼状態・成功監査ログのどこまでを同一 DB transaction に含めるかを別途設計する。
- 失敗監査ログは、業務 transaction の rollback と独立して残す必要があるかを人間確認事項にする。

#### 次サイクル planner への入力

- accepted scope 候補: 「cycle 004 の元帳・残高方向・成功時 DB transaction 境界 docs 更新を再採択し、監査ログ境界は未確定事項として明示する」。
- security design としては、元帳 docs 更新の次または同時に「監査ログ成功/失敗境界とマスキング方針」を扱うことを推奨する。

## 人間確認事項

1. MVP の認証方式を Cookie session + CSRF と Bearer token のどちらで学習・実装するか。
2. パスワードハッシュ方式と最低パラメータを何にするか。
3. `admin` が顧客口座の入金・出金・振込を代行できるか、また `operator` を MVP に含めるか。
4. 監査ログ書き込み失敗時に、残高変更や権限変更を失敗させるか。
5. 失敗時監査ログを、業務 transaction の rollback と独立して残す必要があるか。
6. 監査ログに IP address、User-Agent、request id、対象口座、失敗理由をどこまで残し、どの項目をマスクするか。
7. health / readiness / metrics endpoint を完全公開にするか、詳細情報を内部 network または認証済みに限定するか。

## 次サイクル planner への入力まとめ

- 最優先候補:
  1. cycle 004 の元帳・残高方向・成功時 DB transaction 境界 docs 更新の再採択。ただし監査ログ境界は未確定事項として分離する。
  2. MVP 認証方式、パスワードハッシュ、セッション/token、RBAC、水平権限チェック方針の docs 化。
  3. REST API 入力検証、エラー応答、SQL injection 防止、ログマスキング方針の docs 化。
  4. 監査ログ境界、失敗時扱い、閲覧権限、マスキング方針の docs 化。
- 業務 API、DB schema、残高更新、振込、冪等性の実装は、上記の security design と banking design を小さく具体化してから進めることを推奨する。
- 同一 cycle の `implementer.md` は `blocked: accepted scope not found` であるため、次に implementer を動かす前に planner が accepted scope を明記する必要がある。
- 同一 cycle の `code-reviewer.md` と `banking-reviewer.md` は、cycle 004 accepted scope の再採択を高優先で推奨している。security-reviewer としても、監査ログ境界を未確定事項として分離する前提で同意する。

## テスト・確認結果

- `git status --short`: 作業開始時点は空。作業中に `docs/ai/cycles/2026-06-28-005/implementer.md`、`code-reviewer.md`、`banking-reviewer.md` が追加された。最終確認時点では、私が変更していない `docs/ai/cycles/2026-06-28-001/*.md` の差分も表示されたため、ユーザーまたは他 agent の同時変更として扱い、戻していない。
- `git diff --stat`: レビュー種別を判定した時点では空。最終確認時点では `docs/ai/cycles/2026-06-28-001/*.md` に同時変更があるが、Go ソースコード、README、設計文書、テスト、DB schema、migration の実装差分は確認していない。
- `go test ./...`: 実行不可。現在の実行環境では `go` コマンドが見つからなかった。
- 本レビューで書き込んだファイルは `docs/ai/cycles/2026-06-28-005/security-reviewer.md` のみ。
