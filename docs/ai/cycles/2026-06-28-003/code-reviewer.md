# code-reviewer: 2026-06-28-003

## レビュー対象

- cycle-id: `2026-06-28-003`
- 役割: `code-reviewer`
- レビュー種別: 現在の未コミット実装差分レビュー
- 主な対象差分:
  - `cmd/server/main.go`
  - `cmd/server/main_test.go`
  - `README.md`
  - `docs/ai/cycles/2026-06-28-003/implementer.md` は実装内容確認のため参照のみ

## 確認した入力

- `AGENTS.md`
- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `docs/START_HERE.md`
- `docs/domain-model.md`
- `docs/mvp.md`
- `docs/design-principles.md`
- `docs/data-model.md`
- `docs/use-cases.md`
- `docs/security-notes.md`
- `docs/test-strategy.md`
- `docs/ai/cycles/2026-06-28-003/planner.md`
- `docs/ai/cycles/2026-06-28-003/implementer.md`
- 現在の未コミット差分

## 実行した確認

- `git status --short`
- `git diff --stat`
- `git diff -- README.md cmd/server/main.go cmd/server/main_test.go`
- `go test ./...`

## Finding

### Finding 1: 修正必須の code-review finding はありません

#### 根拠

- planner の accepted scope は、業務 API 追加前に HTTP listen address を安全側の既定値へ変更し、環境変数で明示的に変更可能にし、`http.Server` timeout を設定し、README と unit test を更新する内容でした。
- `cmd/server/main.go` は、既定値 `127.0.0.1:8080` と環境変数 `BANK_SYSTEM_HTTP_ADDR` を `serverConfigFromEnv` に閉じ込め、`newServer` で `http.Server` を構築しています。設定読み込みと server 構築は小さく分離されており、外部 config framework や外部ライブラリは追加されていません。
- `ReadHeaderTimeout: 5 * time.Second` に加え、`ReadTimeout: 10 * time.Second`、`WriteTimeout: 10 * time.Second`、`IdleTimeout: 60 * time.Second` が設定され、timeout が 0 のまま残っていません。
- `cmd/server/main_test.go` は、環境変数未設定時の既定値、環境変数設定時の上書き、`newServer` への address / handler / timeout 反映、既定 timeout の非 0 を確認しています。
- `README.md` は、既定 listen address、`BANK_SYSTEM_HTTP_ADDR=:8080 go run ./cmd/server` の例、外部 interface 待ち受け時の認証・認可・公開範囲確認の注意、`/healthz` が固定レスポンスのみを返すことを説明しています。
- `go test ./...` は成功しました。
- 差分は server 起動設定、timeout、README、cycle implementer artifact に限定され、PostgreSQL schema、DB transaction、認証、認可、金融ドメイン仕様、監査ログ、冪等性には触れていません。

#### 影響

- cycle 002 security-reviewer が指摘していた「全 interface 待ち受けの可能性」と「HTTP server timeout 未設定」は、今回の accepted scope の範囲では解消されています。
- `cmd/server` と `internal/httpapi` の責務分離は維持されています。router / handler 側に server 起動設定が混入していないため、現時点の規模では保守しやすい構成です。
- 業務 API、DB、金融取引の仕様を暗黙に決めていないため、今後の PostgreSQL / transaction / 元帳設計を妨げる変更ではありません。

#### 推奨修正

- 今回の cycle 内で必須の修正はありません。
- 任意の改善として、次に設定項目が増える場合は、`serverConfig` の field 名を外部公開しない現状のまま維持しつつ、設定値の妥当性検証方針を小さく決めるとよいです。たとえば空文字は既定値に戻す現在の挙動で十分ですが、空白のみの `BANK_SYSTEM_HTTP_ADDR`、不正な address、timeout の将来的な環境変数化をどう扱うかは、設定項目追加時に整理する余地があります。
- `newServer` が常に `httpapi.NewRouter()` を内部生成する現状は accepted scope には適合しています。将来 middleware、認証、request logging、test double が必要になった段階で、handler を引数に取る形へ広げるかを検討すれば十分です。現時点での先回り修正は不要です。

#### 次サイクル planner への入力

- この hardening は小さく完了しているため、次 cycle ではより優先度の高い業務 API 前提の設計に戻れます。
- 次候補としては、過去 reviewer と今回 planner の保留事項に沿い、以下のいずれかを小さく採択するのが妥当です。
  1. 元帳・残高方向・transaction 境界・冪等性キー一意スコープ・監査ログ境界を docs に具体化する。
  2. 認証方式、認可/RBAC、CSRF または bearer token、パスワードハッシュ方針を docs に具体化する。
  3. PostgreSQL migration / schema に進む前に、DB transaction manager と repository 境界の設計案を docs に分離する。
- 業務 API 実装に進む前に、認証・認可、入力検証、エラー応答形式、監査ログ、DB transaction 境界の最低限の方針を先に cycle scope として扱うことを推奨します。

## 補足レビュー観点

### Go / REST

- 標準ライブラリのみで実装されており、現時点の学習用 skeleton として依存関係が増えていません。
- `main` は `newServer(serverConfigFromEnv())` と起動処理に集中しており、過度な抽象化はありません。
- `http.ErrServerClosed` を正常停止相当として扱う既存のエラーハンドリングは維持されています。

### PostgreSQL / transaction

- 今回差分では PostgreSQL 接続、migration、repository、transaction 処理は追加されていません。
- そのため、DB 制約、残高更新、取引履歴、監査ログ、冪等性の correctness finding は今回差分からは発生しません。

### 設計 / 保守性

- server 設定を `serverConfig` にまとめたことで、今後設定項目が増えた場合も `main` に直書きが増えにくい構成です。
- 一方で、現時点では環境変数は address のみで、timeout は固定値です。accepted scope は「過度に複雑な設定項目化はしない」方針だったため妥当です。

### テスト

- 追加 test は accepted scope の主要条件を満たしています。
- `go test ./...` が成功しており、既存 `/healthz` の `httptest` ベース test も維持されています。
