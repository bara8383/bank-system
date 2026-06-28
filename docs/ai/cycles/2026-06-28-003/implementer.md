# implementer: 2026-06-28-003

## 参照した accepted scope

- `docs/ai/cycles/2026-06-28-003/planner.md` の accepted scope「HTTP server hardening と最小設定管理」を参照した。
- 目的は、業務 API 追加前に既存の最小 Go REST API server の listen address と timeout を小さく hardening し、README とテストを同期すること。

## 変更内容

### HTTP server 設定

- `cmd/server/main.go` に最小の server 設定を追加した。
- 既定 listen address は `127.0.0.1:8080` とした。
- 環境変数 `BANK_SYSTEM_HTTP_ADDR` が設定されている場合は、その値を listen address として使う。
- `http.Server` に以下の timeout を設定した。
  - `ReadHeaderTimeout: 5 * time.Second`
  - `ReadTimeout: 10 * time.Second`
  - `WriteTimeout: 10 * time.Second`
  - `IdleTimeout: 60 * time.Second`
- server 構築を `serverConfigFromEnv` と `newServer` に分け、unit test で確認できるようにした。
- router は引き続き `httpapi.NewRouter()` を使い、`/healthz` のレスポンス内容は変更していない。

### テスト

- `cmd/server/main_test.go` を追加した。
- 環境変数未設定時に `127.0.0.1:8080` が使われることを確認する test を追加した。
- `BANK_SYSTEM_HTTP_ADDR=:8080` のような環境変数設定が反映されることを確認する test を追加した。
- `newServer` が address、handler、timeout を `http.Server` に反映することを確認する test を追加した。
- 既定 timeout が 0 ではないことを確認する test を追加した。

### README

- `README.md` の実行方法に、既定 listen address が `127.0.0.1:8080` であることを追記した。
- 外部 interface で待ち受けたい場合の例として、`BANK_SYSTEM_HTTP_ADDR=:8080 go run ./cmd/server` を追記した。
- 外部 interface で待ち受ける場合は、将来の業務 API 追加前に認証・認可・公開範囲を確認すべきことを明記した。
- `/healthz` が固定レスポンスのみを返し、環境変数、DB 接続情報、秘密情報、内部パスを返さないことを明記した。

## scope 適合性

- accepted scope の「HTTP listen address の最小設定」は実装済み。
- accepted scope の「既定値は `127.0.0.1:8080`」は実装済み。
- accepted scope の「環境変数で明示的に `:8080` を指定できる」は `BANK_SYSTEM_HTTP_ADDR` で実装済み。
- accepted scope の「少なくとも `ReadHeaderTimeout` を設定する」は実装済み。加えて `ReadTimeout`、`WriteTimeout`、`IdleTimeout` も固定の暫定値として設定した。
- accepted scope の「server 構築をテストしやすくする」は `serverConfigFromEnv` と `newServer` で実装済み。
- accepted scope の「外部ライブラリや config framework は導入しない」は遵守した。
- accepted scope の「README を更新する」は実施済み。
- accepted scope の「`go test ./...` が成功する状態を維持する」は確認済み。

## 実装しなかったこと

- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログの業務 API は実装していない。
- PostgreSQL 接続、DB schema、migration、repository、transaction 処理は実装していない。
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウトは実装していない。
- 口座番号採番、冪等性キー処理、残高更新、取引履歴、監査ログの詳細仕様は確定していない。
- `transaction_type`、`reversal`、残高競合制御、監査ログ境界は確定していない。
- Docker、CI、lint、外部フレームワーク、外部ライブラリ、OpenAPI 仕様は導入していない。
- `/healthz` に DB 状態、環境変数、listen address、hostname、内部 path、stack trace、秘密情報を含めていない。
- cycle 001 / 002 の成果物は編集していない。

## テスト結果

- `gofmt -w cmd/server/main.go cmd/server/main_test.go`: 成功。
- `go test ./...`: 成功。

## 未確認事項

- `BANK_SYSTEM_HTTP_ADDR` という環境変数名が将来の運用命名規則と一致するかは未確認。
- timeout 値は学習用の暫定値であり、本番相当の要件を学ぶ段階では見直しが必要。
- コンテナ実行を正式に採用する場合、既定値を `127.0.0.1:8080` のままにするか、起動例・compose 等で `BANK_SYSTEM_HTTP_ADDR=:8080` を明示するかは今後の確認事項。
- 業務 API 追加前の認証・認可・CSRF または bearer token 方針は未確定。
- 元帳・残高方向・冪等性キー一意スコープ・監査ログ書き込み失敗時の扱いは未確定。
