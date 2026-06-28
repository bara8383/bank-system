# security-reviewer: 2026-06-28-002

## レビュー範囲

- 役割: `security-reviewer` として、同一 cycle の実装差分を優先して確認した。
- 主な確認対象:
  - `docs/ai/cycles/2026-06-28-002/planner.md`
  - `docs/ai/cycles/2026-06-28-002/implementer.md`
  - `README.md`
  - `go.mod`
  - `cmd/server/main.go`
  - `internal/httpapi/router.go`
  - `internal/httpapi/router_test.go`
- 確認したコマンド:
  - `git status --short`
  - `git log --oneline -5`
  - `git show --stat --oneline --decorate HEAD`
  - `git show --name-only --format=short HEAD`
  - `git diff --find-renames HEAD^..HEAD -- ...`
  - `go test ./...`

## 前提

- 今回の accepted scope は、最小 Go REST API 土台、`GET /healthz`、テスト、README に限定されている。
- DB、認証、認可、顧客、口座、入出金、振込、残高、取引履歴、監査ログ、冪等性は実装対象外である。
- 学習用ミニバンキングシステムのため、本番金融システム相当の安全性が実装済みであるとは扱わない。

## 結論

- High / Medium 相当のセキュリティ Finding はない。
- `/healthz` は固定 JSON のみを返し、環境変数、DB 接続情報、秘密情報、内部ファイルパス、stack trace、ホスト固有情報を返さない実装になっている。
- 現時点では業務データや認証情報を扱っていないため、金融情報漏えい・不正送金・権限昇格に直結する実装リスクは確認されなかった。
- ただし、HTTP server の公開範囲とタイムアウト既定値は、今後の業務 API 追加前に方針化・実装する必要がある。

## 確認済み事項

### `/healthz` の情報露出は限定されている

- 根拠:
  - `internal/httpapi/router.go` では `/healthz` のレスポンスが `{"status":"ok"}` の固定値に限定されている。
  - `GET` 以外は `405 Method Not Allowed` と `Allow: GET` で拒否している。
  - `internal/httpapi/router_test.go` では、body が固定レスポンスと一致すること、JSON content type であること、秘密情報・DB 情報・環境情報・内部パスに相当する固定語を含まないことを検証している。
- 影響:
  - 現時点の health check から秘密情報や内部構成が漏れる可能性は低い。
- 推奨修正:
  - 今回の scope 内では修正不要。
  - 将来 readiness check を追加する場合も、DB 接続文字列、環境変数、依存サービス詳細、内部パス、stack trace はレスポンスに含めない。

### README は本番金融システムと誤認されにくい

- 根拠:
  - `README.md` は学習用リポジトリであり、実際の金融機関向け本番システムではないことを警告している。
  - 未実装範囲として DB、認証、認可、業務 API、監査ログ、冪等性キー処理が明記されている。
- 影響:
  - 現状の実装を本番利用可能な金融システムと誤解するリスクは低い。
- 推奨修正:
  - 今回の scope 内では修正不要。

## Finding 1: HTTP server が全 interface で待ち受ける可能性がある

- 重大度: Low
- 人間確認要否: あり

### 根拠

- `cmd/server/main.go` の HTTP server は `Addr: ":8080"` で起動する。
- Go の `net/http` では `:8080` は通常、ローカルホスト限定ではなく全 network interface で待ち受ける指定として扱われる。
- `README.md` の確認例は `localhost:8080` だが、実装上の意図が「ローカル開発専用」なのか「コンテナや外部からの接続も許可」なのかは未確定である。

### 攻撃/事故シナリオ

- 開発者が共有ネットワーク上の端末やクラウド VM で `go run ./cmd/server` を実行した場合、意図せず `/healthz` が外部から到達可能になる。
- 現時点の `/healthz` は固定レスポンスのみで影響は限定的だが、将来、同じ server に未認証の業務 API や詳細な readiness 情報が追加されると、意図しない公開範囲がそのままリスクになる。

### 影響

- 現時点では業務データや秘密情報を返さないため直接影響は低い。
- ただし、今後の認証・業務 API 実装前に待ち受け address の方針を決めないと、開発用設定のまま広い network exposure を持つ構成になりうる。

### 推奨修正

- 次 cycle 以降で、待ち受け address の方針を明文化する。
  - ローカル開発を安全側に倒すなら、既定値を `127.0.0.1:8080` にする。
  - コンテナ実行や公開前提があるなら、環境変数で明示的に `:8080` を指定できるようにし、README に exposure の意味を書く。
- 業務 API を追加する前に、公開可能な endpoint と認証必須 endpoint を分離して設計する。

### 次サイクル planner への入力

- 「server listen address / config 方針」を accepted scope 候補に追加する。
- 人間確認事項として、開発既定値を `127.0.0.1:8080` にするか、コンテナ互換のため `:8080` を維持して README で注意喚起するかを決める。

## Finding 2: HTTP server の timeout が未設定

- 重大度: Low
- 人間確認要否: なし。ただし値の基準は後続で確認してよい。

### 根拠

- `cmd/server/main.go` の `http.Server` は `Addr` と `Handler` のみを設定しており、`ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` は未設定である。
- 現在の handler は軽量な `/healthz` のみだが、server が network に露出した場合、低速な client による connection 維持で goroutine や file descriptor が消費される可能性がある。

### 攻撃/事故シナリオ

- 攻撃者または誤設定された client が HTTP header や request body を非常に遅く送信し、接続を長時間保持する。
- 同時接続が増えると、将来の業務 API や運用確認用 `/healthz` が応答しにくくなり、可用性低下につながる。

### 影響

- 現時点では学習用かつ業務データなしのため影響は低い。
- 金融系では可用性も重要なセキュリティ要件であり、認証や業務 API を追加する前に timeout の安全な既定値を入れることが望ましい。

### 推奨修正

- 次 cycle 以降で、`http.Server` に少なくとも `ReadHeaderTimeout` を設定する。
- 業務 API 追加時には、処理特性に応じて `ReadTimeout`、`WriteTimeout`、`IdleTimeout` も設定し、README または docs に意図を記録する。
- timeout 値は過度に短くして正当な client を落とさないよう、学習用の暫定値として明示する。

### 次サイクル planner への入力

- 「HTTP server hardening」として timeout 設定を小さな改善候補にする。
- この改善は DB・認証・金融仕様を確定せずに実装できるため、次の小規模 hardening scope に適している。

## 人間確認事項

1. 開発時の server listen address は、ローカル限定の `127.0.0.1:8080` を既定にするか、コンテナ互換の `:8080` を既定にするか。
2. 将来の health / readiness endpoint は完全公開にするか、詳細情報を含む readiness は認証済みまたは内部 network 限定にするか。
3. 業務 API を追加する前に、認証・認可・CSRF または bearer token 方針を docs scope として先に固めるか。

## 次サイクル planner への入力まとめ

- High / Medium の修正必須 Finding はないため、今回の最小 skeleton は accepted scope に概ね適合している。
- 次の小規模改善候補:
  1. HTTP listen address と設定管理の方針化。
  2. `http.Server` timeout の追加。
  3. 業務 API 実装前の認証・認可方針 docs 化。
  4. readiness check を追加する場合の情報露出ルール策定。

## テスト・確認結果

- `go test ./...`: 成功。
- `git status --short`: security review 作成前は未コミット変更なし。作成後は本ファイルのみが変更対象。
