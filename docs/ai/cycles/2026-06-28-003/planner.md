# planner: 2026-06-28-003

## repo現状

### 作業開始時に確認したもの

- `git status --short`: 作業開始時点では未コミット変更なし。
- `.codex/agents/README.md`: planner は repo 現状、reviewer 出力、未実装領域から改善案を作り、`docs/ai/cycles/<cycle-id>/planner.md` に accepted scope を出す。implementer は同一 cycle の accepted scope だけを実装する。
- `docs/ai/cycles/README.md`: cycle artifact protocol と、planner 出力の必須項目を確認。
- `AGENTS.md`: 学習用の銀行・金融システムであり、Go + REST + PostgreSQL を前提とする。作業ルールは「小さく実装する」「実装前に既存コードを確認する」「設計判断は docs/ に記録する」「README を最新状態に保つ」。
- `README.md`: 現在の実装範囲は、Go 標準ライブラリのみの最小 REST API server と `GET /healthz`。DB、認証、業務 API、監査ログ、冪等性キー処理は未実装。
- `docs/START_HERE.md`: 最初のゴールは、顧客登録、口座作成、入金、出金、振込、残高・取引履歴照会、監査ログを安全に扱えるミニバンキングシステム。
- `docs/mvp.md`: MVP はユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録を含む。
- `docs/domain-model.md`: 顧客、ログインユーザー、口座、残高、取引、振込依頼、監査ログ、認証、認可、トランザクションなどの用語を確認。
- `docs/data-model.md`: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs` の初期テーブル候補と、金額整数、残高非負、冪等性キー一意制約案を確認。
- `docs/use-cases.md`: UC-001 から UC-008 までの正常系・異常系を確認。入出金・振込では、口座状態、権限、正の整数金額、残高不足、冪等性、ロールバックが重要。
- `docs/design-principles.md`: 残高非負、金額整数、取引履歴、監査ログ、認証認可、原子性、二重実行防止、状態遷移、調査可能なエラーを確認。
- `docs/security-notes.md`: 認証、認可、監査、秘密情報、個人情報、入力検証、今後の対策を確認。
- `docs/test-strategy.md`: 金額計算、残高更新、取引履歴、監査ログ、異常系、振込原子性、冪等性、認証認可を重点的にテストする方針を確認。
- 過去 cycle `2026-06-28-001` と `2026-06-28-002`: planner、implementer、code-reviewer、security-reviewer、banking-reviewer を確認。
- 既存コード: `go.mod`、`cmd/server/main.go`、`internal/httpapi/router.go`、`internal/httpapi/router_test.go` を確認。
- TODO/FIXME: 明示的な `TODO` / `FIXME` は見つからない。未定義・未実装事項は docs と cycle reviewer 出力に記録されている。

### 実装済み

- Go module は `module bank-system` として作成済み。
- `cmd/server/main.go` に、標準ライブラリ `net/http` を使う最小 HTTP server がある。
- `internal/httpapi/router.go` に、`GET /healthz` のみを提供する router / handler がある。
- `/healthz` は固定 JSON `{"status":"ok"}` を返し、`GET` 以外は `405 Method Not Allowed` と `Allow: GET` で拒否する。
- `internal/httpapi/router_test.go` に、`/healthz` の成功、未対応 method 拒否、固定レスポンスの不要情報非露出を検証する `httptest` ベースの unit test がある。
- `README.md` に、現状、起動方法、テスト方法、未実装機能、学習用であり本番金融システムではない旨が記録されている。

### 設計済みだが未実装

- MVP 業務機能: ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- 初期データモデル候補: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs`。
- 重要原則: 金額整数、残高非負、残高変更と取引履歴の整合性、監査ログ、認証認可、振込の原子性、冪等性、状態遷移。
- テスト戦略: 金額・残高・取引履歴・監査ログ・振込原子性・冪等性・認証認可を重点確認する方針。

### 未設計または具体化不足

- HTTP server の設定管理。現状は `Addr: ":8080"` 固定で、listen address の意図と変更方法が未定義。
- HTTP server timeout。`ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` が未設定。
- PostgreSQL migration ツール、DB 接続方法、transaction manager、repository 境界、ローカル DB 起動方法。
- 認証方式、パスワードハッシュ方式、セッション/トークン、CSRF、ログアウト、レート制限。
- RBAC の権限表、管理者・運用担当者の責務分離。
- `transaction_type` ごとの残高増減方向、`balance_after` の検証ルール。
- 冪等性キーの一意スコープ、リクエスト同一性検証、同一キー異内容時の扱い、保存期間。
- 振込ステータス遷移、失敗した振込依頼の保存境界、処理結果不明時の扱い。
- 並行出金・並行振込時の残高保護方式、ロック順序、デッドロック回避方針。
- 監査ログの transaction 境界、失敗時ログ、書き込み失敗時の業務処理継続可否、マスキング規則。
- API 入力検証、検索制限、エラー応答形式、ログ出力規則、データ分類。

### docs/実装不一致

- `README.md` は現状の実装範囲と一致しており、大きな不一致はない。
- docs は MVP 全体の設計方針を示しているが、実装は health check のみであり、業務機能は未実装として明示されている。
- `cmd/server/main.go` の `:8080` 固定は README の `localhost:8080` 例とは利用方法として整合するが、外部 interface で待ち受ける可能性がある点は README に明記されていない。

### レビュー未反映

- cycle 002 code-reviewer: skeleton は accepted scope に適合。次候補として、元帳・残高方向、認証/RBAC、PostgreSQL migration、HTTP エラー応答・入力検証・ログ/監査マスキング方針を挙げている。
- cycle 002 security-reviewer: High / Medium finding はない。一方で、HTTP server が全 interface で待ち受ける可能性と timeout 未設定が Low finding として記録され、次 cycle planner への入力として listen address / config 方針と HTTP server hardening が挙げられている。
- cycle 002 banking-reviewer: skeleton に金融ドメイン上のブロッカーはない。業務 API や DB schema に進む前に、元帳・残高方向・transaction 境界・冪等性・監査ログ境界を設計 scope として切り出すことを推奨している。

## 入力レビュー

### human notes

- `docs/ai/output/human/` は存在しないため、追加の human notes はない。

### cycle 002 implementer

- 最小 Go REST API skeleton、`GET /healthz`、handler test、README を実装済み。
- DB、認証、顧客、口座、入出金、振込、残高、取引履歴、監査ログ、冪等性キー処理は scope 外・未実装として記録済み。
- 未確認事項として、module 名、HTTP listen address、DB、認証、監査ログ、金融ドメイン仕様が挙げられている。

### cycle 002 code-reviewer

- 修正必須 finding はなし。
- 現在の `cmd/server` + `internal/httpapi` 構成は最小 skeleton として妥当。
- 次に設定管理を入れる cycle では、listen address を `:8080` 固定から安全な設定読み込みへ切り出すことが推奨されている。
- 業務 API 前に、認証・認可・エラー形式・監査ログ境界を docs で具体化することが推奨されている。

### cycle 002 security-reviewer

- `/healthz` は固定 JSON のみで、秘密情報、DB 接続情報、内部ファイルパス、stack trace、ホスト固有情報を返していない。
- Finding 1: `Addr: ":8080"` は全 network interface で待ち受ける可能性があり、開発用既定値として安全側かコンテナ互換かの方針が未確定。
- Finding 2: `http.Server` の timeout が未設定。業務 API 追加前に少なくとも `ReadHeaderTimeout` を設定することが推奨されている。
- 人間確認事項として、listen address の既定値、health/readiness endpoint の公開範囲、業務 API 前の認証・認可・CSRF/Bearer token 方針が挙げられている。

### cycle 002 banking-reviewer

- 今回差分では、二重送金、残高マイナス、取引履歴欠落、片側だけ成功する振込などの事故シナリオは発火しない。
- 次に業務 API や DB schema に進む場合は、先に元帳・残高方向・transaction 境界・冪等性・監査ログ境界を設計 scope として切り出すことが強く推奨されている。
- 人間確認事項として、`transaction_type`、`reversal`、残高競合制御、冪等性キー一意スコープ、同一キー異内容時の扱い、監査ログ書き込み失敗時の扱いが挙げられている。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. HTTP server hardening と最小設定管理を実装する | `cmd/server/main.go` は `Addr: ":8080"` 固定で timeout 未設定。cycle 002 security-reviewer が Low finding として listen address と timeout を指摘。 | listen address の既定値、環境変数での変更、timeout 値、README 記載、テストが不足。 | 業務 API 追加前に server の公開範囲と低速 client 対策を小さく整えられる。金融仕様や DB schema を確定しないため後戻りリスクが低い。 | security-reviewer: network exposure と可用性。code-reviewer: 設定読み込みの過剰設計回避とテスト容易性。banking-reviewer: 金融仕様に触れないこと。 | 既定値は安全側の `127.0.0.1:8080` とし、コンテナ等で必要な場合だけ環境変数で明示的に `:8080` を許可する。設定値を `/healthz` に出さない。 |
| B. 元帳・残高方向・transaction 方針を docs に追加する | cycle 001/002 banking-reviewer が一貫して高優先で推奨。`docs/domain-model.md` と `docs/data-model.md` には概念はあるが、残高増減方向や検証ルールは未具体化。 | `transaction_type` の符号、`balance_after`、残高整合性検証、ロック方式、デッドロック回避、監査境界が未定義。 | 入出金・振込実装前の金融事故リスク低減に必要。 | banking-reviewer: 元帳整合性。code-reviewer: DB transaction 境界。security-reviewer: 監査境界。 | `reversal` や残高競合制御方式は後戻りしにくいため、人間確認事項を分離する。 |
| C. 認証・RBAC・セッション/token 方針を docs に追加する | `docs/mvp.md` と `docs/design-principles.md` は認証・認可を必須としている。cycle 001 security-reviewer も認証方式と RBAC 未確定を重要視。 | Cookie session / Bearer token、パスワードハッシュ、CSRF、ロール権限表、管理者作成方法が未定義。 | MVP の業務 API は認証認可なしでは安全に追加できない。 | security-reviewer: 水平権限不備、認証強度、秘密情報。code-reviewer: handler/service 境界。 | 安全上重要な仕様であり、人間確認なしに最終確定しない。 |
| D. PostgreSQL migration 方針と最小 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。code-reviewer は DB 制約の具体化を推奨。 | migration ツール、ID 型、制約、index、DB 起動方法、冪等性 scope、残高競合方式が未定義。 | 残高非負、金額正数、取引履歴、冪等性を DB 側でも守る土台になる。 | code-reviewer: migration と transaction。banking-reviewer: 残高・元帳・冪等性。security-reviewer: 個人情報とシークレット。 | schema は後戻りしにくい。元帳・認証・監査・冪等性の未確認事項を解消してから採択する。 |
| E. API エラー応答・入力検証・ログ/マスキング方針を docs に追加する | `docs/security-notes.md` は入力検証と秘密情報ログ禁止を示す。cycle 001/002 reviewer はエラー形式、validation、ログ/監査マスキングの具体化を推奨。 | エラー JSON 形式、request id、validation error 表現、検索上限、ログ出力項目、機微情報マスキングが未定義。 | 業務 API 追加時に情報漏えいとテストばらつきを防ぐ。 | security-reviewer: 情報露出。code-reviewer: エラー分類とテスト。banking-reviewer: 失敗時証跡。 | 監査ログ境界と重なるため、先に小さく API/ログ標準だけ切るか、監査方針 cycle と統合するかを検討する。 |

## 採択

### 採択: A. HTTP server hardening と最小設定管理を実装する

- 理由: cycle 002 で最小 Go REST API skeleton は成立した。次に業務 API、DB、認証へ進む前に、現 server の公開範囲と timeout を小さく改善することは、既存 reviewer 入力に沿い、金融仕様を確定せず、後戻りリスクも低い。
- security-reviewer 入力への対応: listen address の既定値を安全側に倒し、必要時だけ環境変数で明示的に広い listen を許可する。`http.Server` に timeout を設定し、低速 client による可用性低下リスクを下げる。
- code-reviewer 入力への対応: 設定読み込みを最小関数に分離し、標準ライブラリのみで unit test 可能にする。過剰な config framework は導入しない。
- banking-reviewer 入力への対応: 残高、元帳、取引履歴、振込、冪等性、監査ログ、DB transaction には触れず、金融ドメイン仕様を暗黙に確定しない。
- README への対応: 起動時の既定 listen address と、外部 interface で待ち受けたい場合の環境変数を明記する。学習用であり本番金融システムではない警告は維持する。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| D. PostgreSQL migration 方針と最小 schema を作る | DB schema は後戻りしにくく、元帳方向、冪等性スコープ、監査境界、認証方式、migration ツールが未確定。今回の小さな hardening より設計判断が重い。 | 人間確認事項が整理され、元帳・残高方向・transaction 方針が docs で具体化された後。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| B. 元帳・残高方向・transaction 方針を docs に追加する | 業務 API / DB schema 前に必須だが、今回の accepted scope は cycle 002 security finding の小修正に限定する。 | `reversal` を MVP 前に定義するか。残高競合制御を行ロックか条件付き UPDATE のどちらにするか。 | 次 cycle 以降で高優先候補にする。 |
| C. 認証・RBAC・セッション/token 方針を docs に追加する | 認証方式は安全上重要で、人間確認なしに最終確定しない。今回の変更は `/healthz` と server 起動設定のみ。 | Cookie session か Bearer token か。CSRF、ログアウト、パスワードハッシュ、管理者作成方法、運用担当者ロール。 | 業務 API 実装前に docs scope として採択する。 |
| E. API エラー応答・入力検証・ログ/マスキング方針を docs に追加する | 業務 API 追加前には必要だが、現在の API は `/healthz` の固定応答のみ。監査ログ方針との関係もある。 | 失敗監査ログの保存境界、request id の扱い、ログに含める actor / IP / User-Agent、検索上限。 | 認証・業務 API または監査ログ設計の前に docs scope として採択する。 |

## accepted scope

### 目的

- 既存の最小 Go REST API server を、業務 API 追加前に小さく hardening する。
- 開発時の意図しない外部公開を避けるため、既定 listen address を安全側にする。
- 低速 client による接続保持リスクを下げるため、`http.Server` の timeout を設定する。
- 設定読み込みと server 構築を小さく分離し、unit test で確認できるようにする。
- 金融仕様、DB schema、認証方式、監査ログ方式は確定しない。

### 対象ファイル/領域

- `cmd/server/main.go`
  - server 設定読み込み、`http.Server` 構築、起動ログの最小整理。
- `cmd/server/main_test.go` または同等の server 設定 unit test。
  - 環境変数による address 設定、既定値、timeout 値を検証する。
- `README.md`
  - 起動方法、既定 listen address、外部 interface で待ち受ける場合の環境変数、注意事項を更新する。
- `docs/ai/cycles/2026-06-28-003/implementer.md`
  - 実装結果、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

### 実装対象

1. HTTP listen address の最小設定を追加する。
   - 既定値は `127.0.0.1:8080` とする。
   - 環境変数名は小さく分かりやすいものにする。例: `BANK_SYSTEM_HTTP_ADDR`。
   - 環境変数が空の場合は既定値を使う。
   - コンテナや外部接続が必要な開発者は、明示的に `BANK_SYSTEM_HTTP_ADDR=:8080` のように指定できるようにする。
2. `http.Server` に timeout を設定する。
   - 少なくとも `ReadHeaderTimeout` を設定する。
   - 学習用の暫定値として、`ReadHeaderTimeout: 5 * time.Second` を候補にする。
   - 必要であれば `ReadTimeout`、`WriteTimeout`、`IdleTimeout` も小さく妥当な値で設定する。ただし、過度に複雑な設定項目化はしない。
3. server 構築をテストしやすくする。
   - `main` の中に設定読み込みと `http.Server` literal を直書きしすぎず、`newServer`、`serverConfigFromEnv` などの小さな関数に切り出す。
   - 既存の `httpapi.NewRouter()` を引き続き利用する。
   - 外部ライブラリや config framework は導入しない。
4. unit test を追加する。
   - 環境変数未設定時に address が `127.0.0.1:8080` になること。
   - 環境変数設定時に address がその値になること。
   - `ReadHeaderTimeout` が 0 ではないこと。
   - 可能なら、設定変更後も `/healthz` handler が既存どおり固定 JSON を返すことは既存 test で維持されることを確認する。
5. README を更新する。
   - 既定の起動先が `127.0.0.1:8080` であることを明記する。
   - 外部 interface で待ち受ける場合の例として `BANK_SYSTEM_HTTP_ADDR=:8080 go run ./cmd/server` を記載する。
   - 外部 interface での待ち受けは、将来業務 API を追加する前に認証・公開範囲を確認すべきである旨を注意書きする。
6. `go test ./...` が成功する状態を維持する。
7. `docs/ai/cycles/2026-06-28-003/implementer.md` に、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

### 実装しないこと

- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログの業務 API は実装しない。
- PostgreSQL 接続、DB schema、migration、repository、transaction 処理は実装しない。
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウトは実装しない。
- 口座番号採番、冪等性キー処理、残高更新、取引履歴、監査ログの詳細仕様は確定しない。
- `transaction_type`、`reversal`、残高競合制御、監査ログ境界は今回確定しない。
- Docker、CI、lint、外部フレームワーク、外部ライブラリ、OpenAPI 仕様は導入しない。
- `/healthz` に DB 状態、環境変数、listen address、hostname、内部 path、stack trace、秘密情報を含めない。
- cycle 001 / 002 の成果物は編集しない。

### テスト方針

- `go test ./...` を実行する。
- server 設定の unit test では、環境変数の設定・復元を `testing.T.Setenv` などで局所化する。
- `httptest` ベースの既存 `/healthz` test を維持する。
- DB、外部ネットワーク、外部サービス、Docker を必要とするテストは作らない。
- server の手動起動確認をする場合は、長時間常駐プロセスを残さない。
- README に記載したコマンドと実際の挙動を一致させる。

### レビューで重点確認してほしい観点

- code-reviewer:
  - 設定読み込みと server 構築が過剰設計でなく、標準ライブラリのみで小さく実装されているか。
  - `main`、server 設定、router の責務が過度に混ざっていないか。
  - timeout 値が 0 のままになっていないか。
  - README の起動例と実装の既定値が一致しているか。
- security-reviewer:
  - 既定 listen address が意図せず全 interface を公開しない設定になっているか。
  - 外部 interface で待ち受ける場合は環境変数で明示的に指定する形になっているか。
  - `/healthz` に設定値、環境情報、秘密情報、内部情報を露出していないか。
  - timeout 設定が低速 client による接続保持リスクを下げる内容になっているか。
- banking-reviewer:
  - 今回の変更が残高、元帳、取引履歴、振込、冪等性、監査ログ、DB transaction の仕様を暗黙に確定していないか。
  - README が現状の未実装範囲を引き続き明確にしているか。
  - 次 cycle で元帳・残高方向・冪等性・監査境界を扱う必要が見落とされていないか。

## 実装しないこと

- planner として、ソースコード、DB schema、認証方式、金融仕様の実装・最終決定は行わない。
- 本ファイル `docs/ai/cycles/2026-06-28-003/planner.md` 以外へ書き込まない。
- cycle 001 / 002 の成果物を修正しない。
- ユーザー作業や他 agent 作業を revert しない。
- 保留事項や人間確認事項を accepted scope に混ぜない。

## 人間確認事項

1. 開発時の既定 listen address は、本 planner では安全側の `127.0.0.1:8080` を採択候補にした。コンテナ互換のため `:8080` を既定にすべき事情があるか。
2. `BANK_SYSTEM_HTTP_ADDR` という環境変数名でよいか。既存の運用命名規則がある場合は合わせる必要がある。
3. HTTP timeout の暫定値は、学習用として `ReadHeaderTimeout: 5s` を候補にした。将来、本番相当の要件を学ぶ段階で別途見直す必要がある。
4. health / readiness endpoint は将来も完全公開にするか。DB 等の詳細 readiness を追加する場合、内部 network 限定または認証必須にするか。
5. 業務 API を追加する前に、認証・認可・CSRF または bearer token 方針を docs scope として先に固めるか。
6. 次に金融ドメインへ進む前に、`transaction_type`、`reversal`、残高競合制御、冪等性キー一意スコープ、監査ログ書き込み失敗時の扱いをどの順番で確認するか。
