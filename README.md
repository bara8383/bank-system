# bank-system

`bank-system` は、銀行・金融システムの設計、セキュリティ、保守性を学習するためのミニバンキングシステムです。

> [!WARNING]
> このリポジトリは学習用です。実際の金融機関向け本番システムではありません。

## 現在の実装範囲

現時点で実装済みなのは、Go 標準ライブラリだけを使った最小 REST API サーバー、ヘルスチェック、金額・残高 validation、口座ステータス validation、取引種別 validation、取引種別に応じた残高反映 helper、domain error を安全な failure category へ写像する helper の domain 土台です。

- `GET /healthz`: サーバーの最小ヘルスチェックとして `{"status":"ok"}` を返します。
- `internal/domain`: JPY の整数最小通貨単位を `int64` で扱う金額・残高 helper、口座ステータス helper、取引種別 helper、safe failure category helper を提供します。金額・残高では、正の取引金額、0 以上の残高、開始残高と取引金額を再検証する残高加算、開始残高と取引金額を再検証して残高不足を拒否する減算に加え、constructor を経由しない値を service / repository / DB insert 境界で再検証する `Validate()` method を利用できます。口座ステータスでは、MVP の `active` / `suspended` / `closed` を検証し、残高変更系操作へ進めるのは `active` のみであることを `EnsureAccountCanTransact` で確認できます。取引種別では、MVP の `deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` を検証し、`ApplyTransaction` で取引種別に応じた取引後残高を計算できます。`deposit` と `transfer_credit` は残高を増やし、`withdrawal` と `transfer_debit` は残高を減らします。`reversal` は取消仕様が未確定のため valid type には含めていません。`FailureReasonFromError` は既知の domain sentinel error を `invalid_amount` などの固定分類へ写像し、未知 error や `nil` は未分類として扱います。`SafeFailureReasonFromError` は監査ログ / safe structured log 用の fallback helper として、既知 domain error は同じ固定分類へ、未知の non-nil error は raw message ではなく `internal_error` へ寄せます。これらの helper は将来の監査ログ・安全な構造化ログで使う分類候補であり、利用者向け HTTP error response / status code の最終仕様、監査ログ永続化、DB schema はまだ実装していません。
- 外部ライブラリ、DB 接続、認証、業務 API はまだ導入していません。

## 実行方法

```bash
go run ./cmd/server
```

既定では `127.0.0.1:8080` で待ち受けます。サーバー起動後、別のターミナルから次を実行できます。

```bash
curl http://localhost:8080/healthz
```

期待されるレスポンスは次のとおりです。

```json
{"status":"ok"}
```

## listen address の変更

既定の `127.0.0.1:8080` はローカル開発で意図せず外部公開しないための設定です。コンテナ実行などで外部 interface から到達させたい場合は、明示的に環境変数を指定します。

```bash
BANK_SYSTEM_HTTP_ADDR=:8080 go run ./cmd/server
```

外部 interface で待ち受ける場合は、将来の業務 API 追加前に認証・認可・公開範囲を確認してください。`/healthz` は固定レスポンスのみを返し、環境変数、DB 接続情報、秘密情報、内部パスは返しません。

## テスト方法

```bash
go test ./...
```

## 未実装の機能

次の機能はまだ実装していません。

- 顧客登録
- 口座作成
- 入金、出金、振込の業務 API
- 残高照会、取引履歴照会
- 取引履歴の永続化、transaction row の作成、`balance_after` の DB 保存
- PostgreSQL 接続、DB schema、migration、transaction 処理
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウト
- HTTP error response / status code mapping
- 監査ログ
- 冪等性キー処理

金融ドメイン仕様、DB schema、認証方式、監査ログ方式は今後の設計・実装 cycle で扱います。
