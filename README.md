# bank-system

`bank-system` は、銀行・金融システムの設計、セキュリティ、保守性を学習するためのミニバンキングシステムです。

> [!WARNING]
> このリポジトリは学習用です。実際の金融機関向け本番システムではありません。

## 現在の実装範囲

現時点で実装済みなのは、Go 標準ライブラリだけを使った最小 REST API サーバー、ヘルスチェック、金額・残高 validation の domain 土台です。

- `GET /healthz`: サーバーの最小ヘルスチェックとして `{"status":"ok"}` を返します。
- `internal/domain`: JPY の整数最小通貨単位を `int64` で扱う金額・残高 helper を提供します。正の取引金額、0 以上の残高、残高加算、残高不足を拒否する減算に加え、constructor を経由しない値を service / repository / DB insert 境界で再検証する `Validate()` method を利用できます。
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
- 入金、出金、振込
- 残高照会、取引履歴照会
- PostgreSQL 接続、DB schema、migration、transaction 処理
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウト
- 監査ログ
- 冪等性キー処理

金融ドメイン仕様、DB schema、認証方式、監査ログ方式は今後の設計・実装 cycle で扱います。
