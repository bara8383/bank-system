# security-reviewer: 2026-06-28-003

## レビュー範囲

- 役割: `security-reviewer` として、同一 cycle の未コミット実装差分を優先して金融系セキュリティ観点で確認した。
- 主な確認対象:
  - `AGENTS.md`
  - `.codex/agents/README.md`
  - `docs/ai/cycles/README.md`
  - `docs/START_HERE.md`
  - `docs/ai/cycles/2026-06-28-003/planner.md`
  - `docs/ai/cycles/2026-06-28-003/implementer.md`
  - `docs/ai/cycles/2026-06-28-002/security-reviewer.md`
  - `README.md`
  - `cmd/server/main.go`
  - `cmd/server/main_test.go`
  - `internal/httpapi/router.go`
  - `internal/httpapi/router_test.go`
- 確認したコマンド:
  - `git status --short`
  - `rg --files docs | sort`
  - `git diff --stat`
  - `git diff -- README.md cmd/server/main.go`
  - `go test ./...`

## 前提

- 今回の accepted scope は「HTTP server hardening と最小設定管理」であり、listen address の既定値を安全側に倒し、HTTP timeout を追加し、README と unit test を同期することに限定されている。
- DB、認証、認可、顧客、口座、入出金、振込、残高、取引履歴、監査ログ、冪等性キー処理は今回も未実装である。
- 学習用ミニバンキングシステムのため、本レビューは本番金融システム相当の安全性を保証するものではなく、現在の小さな実装差分に対する repo-grounded な確認である。

## 結論

- High / Medium 相当のセキュリティ Finding はない。
- cycle 002 security-reviewer の Low finding だった「全 interface 待ち受けの可能性」と「HTTP server timeout 未設定」は、今回差分で概ね解消されている。
- 既定 listen address は `127.0.0.1:8080` になり、外部 interface で待ち受ける場合は `BANK_SYSTEM_HTTP_ADDR=:8080` のように明示する形になった。
- `http.Server` には `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` が設定され、低速 client による接続保持リスクは以前より下がっている。
- `/healthz` は引き続き固定 JSON のみで、設定値、環境変数、DB 接続情報、秘密情報、内部 path、stack trace を返す差分は確認されなかった。
- 残る主要リスクは今回 scope 外であり、業務 API 追加前に認証・認可・CSRF/Bearer token 方針、監査ログ境界、入力検証・エラー応答・ログマスキング方針を docs で具体化する必要がある。

## 確認済み事項

### 1. 既定 listen address はローカル限定に改善されている

- 根拠:
  - `cmd/server/main.go` で `defaultHTTPAddr = "127.0.0.1:8080"` が定義されている。
  - `serverConfigFromEnv` は `BANK_SYSTEM_HTTP_ADDR` が空の場合に `defaultHTTPAddr` を使う。
  - `cmd/server/main_test.go` は環境変数未設定時に既定 address が使われること、環境変数設定時に `:8080` が反映されることを検証している。
  - `README.md` は既定で `127.0.0.1:8080` に待ち受けること、外部 interface で待ち受ける場合は `BANK_SYSTEM_HTTP_ADDR=:8080 go run ./cmd/server` を明示することを説明している。
- 影響:
  - 開発者が共有ネットワーク上やクラウド VM 上で単に `go run ./cmd/server` を実行した場合に、意図せず全 network interface へ公開される可能性は下がった。
  - 将来の業務 API を同じ server に追加する前段として、公開範囲の既定値が安全側になったことは金融系の防御的設定として妥当である。
- 推奨修正:
  - 今回差分に対する修正必須事項はない。
  - 将来 Docker / compose / devcontainer を追加する場合は、コンテナ内の `127.0.0.1` とホスト到達性の違いを README または docs に明記し、必要な場合だけ `BANK_SYSTEM_HTTP_ADDR=:8080` を例示する。

### 2. HTTP server timeout は追加されている

- 根拠:
  - `cmd/server/main.go` の `serverConfigFromEnv` は `ReadHeaderTimeout: 5 * time.Second`、`ReadTimeout: 10 * time.Second`、`WriteTimeout: 10 * time.Second`、`IdleTimeout: 60 * time.Second` 相当の値を返す。
  - `newServer` はそれらの値を `http.Server` に設定している。
  - `cmd/server/main_test.go` は `newServer` が timeout を反映すること、既定 timeout が 0 でないことを検証している。
- 影響:
  - HTTP header や request の送信を遅延させる client による接続保持、goroutine / file descriptor 消費のリスクは、timeout 未設定だった前 cycle より低減している。
  - 金融系では可用性も重要なセキュリティ要件であり、業務 API 追加前に timeout の安全な暫定値が入ったことは妥当である。
- 推奨修正:
  - 今回差分に対する修正必須事項はない。
  - 将来、長時間処理や大きな request body を扱う API を追加する場合は、endpoint 特性に応じて timeout 値、request body size limit、context deadline、reverse proxy 側 timeout との整合を再確認する。

### 3. `/healthz` の情報露出は増えていない

- 根拠:
  - `internal/httpapi/router.go` の `/healthz` は引き続き `{"status":"ok"}` の固定レスポンスのみを返している。
  - 今回差分で `/healthz` に listen address、環境変数、timeout 値、hostname、DB 状態、内部 path、stack trace、秘密情報を追加する変更はない。
  - `internal/httpapi/router_test.go` は固定レスポンス、JSON content type、不要な operational detail の非露出を検証している。
  - `README.md` も `/healthz` が固定レスポンスのみを返し、環境変数、DB 接続情報、秘密情報、内部パスを返さないと説明している。
- 影響:
  - health check endpoint から秘密情報や内部構成が漏れるリスクは、現時点では低い。
- 推奨修正:
  - 今回差分に対する修正必須事項はない。
  - 将来 readiness endpoint を追加する場合は、依存サービスの詳細エラー、接続文字列、内部 path、stack trace を外部応答に含めず、詳細は適切にマスクしたログまたは内部向け監視に限定する。

## Finding

### Finding 1: 修正必須の新規セキュリティ Finding は確認されなかった

- 重大度: Informational
- 人間確認要否: なし。ただし次 cycle で扱う安全上重要な仕様には人間確認が必要。

#### 根拠

- 今回の未コミット実装差分は、`README.md`、`cmd/server/main.go`、`cmd/server/main_test.go`、`docs/ai/cycles/2026-06-28-003/implementer.md` に限定されている。
- 実装差分は accepted scope に沿って、listen address 既定値、環境変数による明示的な override、HTTP server timeout、README 説明、unit test を追加している。
- 業務 API、DB 接続、認証・認可、顧客情報、口座情報、金額、残高、取引履歴、監査ログ、冪等性キーを扱う処理は追加されていない。
- `go test ./...` は成功した。

#### 影響

- 現時点の差分から、秘密情報漏えい、不正送金、水平権限不備、SQL injection、監査証跡欠落などに直結する新規リスクは確認されなかった。
- ただし、金融業務 API が未実装であるため、システム全体として認証済み・認可済みの銀行操作を安全に実行できる段階ではない。

#### 推奨修正

- 今回差分に対する修正必須事項はない。
- 次に業務 API を追加する前に、少なくとも以下を docs scope として具体化する。
  1. 認証方式、セッション / token、CSRF または Bearer token 方針。
  2. RBAC と水平権限チェック方針。
  3. 入力検証、request body size limit、検索・一覧系の上限。
  4. エラー応答形式と情報露出禁止ルール。
  5. 監査ログに残す項目、マスキング、失敗時ログ、監査ログ書き込み失敗時の扱い。

#### 次サイクル planner への入力

- 今回の hardening は完了候補として扱ってよい。
- 次 cycle では、業務 API 実装に進む前の security design scope として「認証・認可・CSRF/Bearer token・RBAC 方針 docs 化」または「API エラー応答・入力検証・ログ/マスキング方針 docs 化」を高優先候補にする。

### Finding 2: `BANK_SYSTEM_HTTP_ADDR` は明示的に外部公開を許可できるため、将来の業務 API 追加時に運用ガードが必要

- 重大度: Low
- 人間確認要否: あり

#### 根拠

- `serverConfigFromEnv` は `BANK_SYSTEM_HTTP_ADDR` が設定されている場合、その値をそのまま `http.Server.Addr` に反映する。
- `README.md` はコンテナ実行などで外部 interface から到達させたい場合の例として `BANK_SYSTEM_HTTP_ADDR=:8080 go run ./cmd/server` を示している。
- 現在は `/healthz` のみで影響は限定的だが、同じ server に将来未認証または認可不十分な業務 API が追加されると、環境変数 1 つで公開範囲が広がる構成になる。

#### 影響

- 現時点では固定 `/healthz` のみのため直接的な情報漏えい・不正取引リスクは低い。
- 将来、認証・認可・公開範囲の整理より先に業務 API を追加した場合、開発環境やコンテナ環境で意図せず外部から業務 API に到達できる事故につながる可能性がある。

#### 推奨修正

- 今回 scope では、外部 interface を明示指定できること自体は accepted scope と README に沿っているため、コード修正は不要。
- 次 cycle 以降で、業務 API を追加する前に以下を docs に明記する。
  - 開発用 listen address と公開用 listen address の違い。
  - 公開可能 endpoint と認証必須 endpoint の分類。
  - `/healthz` と将来の readiness / metrics endpoint の公開範囲。
  - コンテナや reverse proxy を導入する場合の責務分担。
- 本番相当の学習段階に進む場合は、`BANK_SYSTEM_HTTP_ADDR=:8080` の利用条件、TLS / reverse proxy / network policy / firewall 前提を人間確認事項として扱う。

#### 次サイクル planner への入力

- 「公開範囲・認証必須 endpoint・health/readiness 公開方針」を次 cycle の security design 候補に入れる。
- 業務 API 実装 cycle を採択する前に、`BANK_SYSTEM_HTTP_ADDR=:8080` を使う開発・コンテナ手順が未認証 API 公開につながらないか確認する。

## 人間確認事項

1. `BANK_SYSTEM_HTTP_ADDR` による `:8080` 待ち受けを、今後も開発者が明示的に選べる方針でよいか。
2. 将来の health / readiness endpoint は完全公開にするか、詳細 readiness は内部 network 限定または認証必須にするか。
3. 業務 API 追加前に、Cookie session + CSRF と Bearer token のどちらを学習用 MVP の認証方式として検討するか。
4. RBAC の最小ロール、顧客本人の水平権限チェック、管理者・運用者の責務分離をどの docs cycle で確定するか。
5. 監査ログに IP address / User-Agent / request id / actor / 対象口座をどこまで残し、個人情報・秘密情報をどうマスクするか。

## 次サイクル planner への入力まとめ

- 今回の HTTP server hardening は、security-reviewer 観点では accepted scope に概ね適合しており、修正必須の新規 Finding はない。
- 次の優先候補:
  1. 認証・認可・CSRF/Bearer token・RBAC 方針の docs 化。
  2. API エラー応答・入力検証・request body size limit・ログマスキング方針の docs 化。
  3. health / readiness / metrics の公開範囲と情報露出ルールの docs 化。
  4. 監査ログ境界、失敗時監査ログ、監査ログ書き込み失敗時の扱いの docs 化。
- 業務 API、DB schema、残高更新、振込、冪等性を実装する前に、上記の security design と banking design を分けて小さく採択することを推奨する。

## テスト・確認結果

- `go test ./...`: 成功。
- `git status --short`: レビュー開始時点では `README.md`、`cmd/server/main.go` が変更済み、`cmd/server/main_test.go`、`docs/ai/cycles/2026-06-28-003/implementer.md` が未追跡だった。本レビュー作成により `docs/ai/cycles/2026-06-28-003/security-reviewer.md` のみを追加した。
