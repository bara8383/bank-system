# code-reviewer: 2026-06-28-002

## レビュー種別

- 同一 cycle の実装差分を優先してレビューした。
- 対象差分は `HEAD` の `261cc5e Add minimal Go REST API skeleton`。
- レビュー対象は、`README.md`、`go.mod`、`cmd/server/main.go`、`internal/httpapi/router.go`、`internal/httpapi/router_test.go`、`docs/ai/cycles/2026-06-28-002/implementer.md`。
- ソースコードは変更していない。

## 確認した入力

- `AGENTS.md`
- `docs/ai/cycles/README.md`
- `docs/ai/cycles/2026-06-28-002/planner.md`
- `docs/ai/cycles/2026-06-28-002/implementer.md`
- `docs/START_HERE.md`
- `docs/mvp.md`
- `docs/domain-model.md`
- `docs/data-model.md`
- `docs/use-cases.md`
- `docs/design-principles.md`
- `docs/security-notes.md`
- `docs/test-strategy.md`
- 実装ファイル一式
- `git status --short`
- `git log --oneline --decorate -5`
- `git diff --stat`
- `git diff`
- `git show --stat --oneline --name-only HEAD`
- `git show --find-renames --find-copies --format=fuller --stat --patch HEAD`

## 結論

- 修正必須の Finding は見つからなかった。
- accepted scope に対して、実装は小さく、標準ライブラリのみで構成されており、`main` と handler も分離されている。
- `/healthz` は固定 JSON のみを返し、今回の scope で避けるべき DB、認証、金融ドメイン処理、監査ログ、冪等性、残高更新には踏み込んでいない。
- README は現状、起動方法、テスト方法、未実装範囲、学習用であることを明記しており、今回の実装と大きな不一致はない。

## Findings

### Finding なし: 最小 Go REST API skeleton は accepted scope に適合している

#### 根拠

- `go.mod` は `module bank-system` と Go version のみで、外部依存を追加していない。
- `cmd/server/main.go` は `http.Server` の `Handler` に `httpapi.NewRouter()` を渡し、handler を `main` から分離している。
- `internal/httpapi/router.go` は `http.NewServeMux()` で `/healthz` のみを登録している。
- `/healthz` は `GET` のみを許可し、非対応 method には `Allow: GET` と `405 Method Not Allowed` を返している。
- 成功レスポンスは `Content-Type: application/json; charset=utf-8` と固定 body `{"status":"ok"}\n` のみである。
- `internal/httpapi/router_test.go` は `httptest` で成功 status、body、Content-Type、非対応 method、固定語ベースの不要情報非露出を確認している。
- `README.md` は `go run ./cmd/server`、`go test ./...`、未実装機能、学習用であり本番金融システムではないことを記録している。
- `docs/ai/cycles/2026-06-28-002/implementer.md` は、実装したこと・実装しなかったこと・テスト結果・未確認事項を accepted scope と対応づけて記録している。

#### 影響

- 後続 cycle で業務 API、DB、認証、監査ログを追加するための最小実行単位として妥当。
- 現時点では金融データや認証情報を扱っていないため、残高整合性、DB transaction、ロック、認可漏れ、監査ログ境界の実装バグは発生していない。
- handler が分離されているため、今後も HTTP 層の unit test を追加しやすい。

#### 推奨修正

- 今 cycle での修正は不要。
- 次に設定管理を入れる cycle では、listen address を固定 `:8080` から安全な設定読み込みへ切り出す。ただし、設定値を `/healthz` やログへ過剰露出しない方針を同時に決める。
- 次に業務 API を入れる前に、認証・認可・エラー形式・監査ログ境界を docs で具体化する。

#### 次サイクル planner への入力

- Go skeleton は成立したため、次の候補は「DB/元帳/認証をいきなり同時に実装する」のではなく、次のいずれかに小さく分けるのが望ましい。
  1. 元帳・残高方向・振込 transaction 方針を docs に具体化する。
  2. 認証・RBAC・セッションまたは token 方針を docs に具体化する。
  3. PostgreSQL migration 方針と最小 schema を、残高非負・金額正数・冪等性 scope の人間確認後に作る。
  4. HTTP エラー応答形式、入力検証方針、ログ/監査ログのマスキング方針を docs に具体化する。

## 追加観点

### Go / package 構成

- 現在の `cmd/server` + `internal/httpapi` 構成は、最小 skeleton として過剰ではない。
- handler を `NewRouter()` 経由で生成しており、`httptest` で直接検証できる。
- 現時点では service/repository 層を作っていないが、業務ロジックが存在しないため妥当。

### PostgreSQL / transaction 境界

- PostgreSQL 接続、migration、repository、transaction 処理は未実装であり、今回の accepted scope どおり。
- 次に DB を扱う場合は、`accounts.balance_amount >= 0`、金額正数、外部キー、冪等性キーの一意 scope、振込時のロック順序を先に設計する必要がある。

### API 設計

- `/healthz` は固定レスポンスで、環境変数、DB 接続情報、秘密情報、内部パス、stack trace を返していない。
- 非対応 method を `405` と `Allow: GET` で返す点は、最小 REST API として明確。
- 今後 API を増やす際は、エラー JSON 形式、request id、validation error の表現、機微情報を含まないログ方針を決める必要がある。

### テスト容易性

- `go test ./...` が成功し、README のテスト手順と一致している。
- `httptest` を使った handler test があるため、今後の endpoint 追加時も同じ方針で拡張できる。
- 現時点では DB や外部サービスを必要とするテストがなく、学習用 repo の初期段階として扱いやすい。

## テスト結果

- `go test ./...`: 成功。

## 人間確認事項

- 公開 module path が必要になった場合、`module bank-system` のままでよいか確認する。
- listen address を環境変数で変更可能にする場合、設定の読み込み方、デフォルト値、ログ出力範囲を確認する。
- DB 実装前に、migration ツール、ID 型、口座番号採番、冪等性キー一意 scope、残高競合制御方式を確認する。
- 業務 API 実装前に、認証方式、認可モデル、監査ログ失敗時の扱い、エラー応答形式を確認する。
