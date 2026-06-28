# code-reviewer: 2026-06-28-004

## レビュー対象

- cycle-id: `2026-06-28-004`
- 役割: `code-reviewer`
- レビュー種別: repo-wide review
- 理由:
  - 作業開始時の `git status --short` と `git diff --stat` では、実装差分は確認できなかった。
  - その後、同一 cycle の `docs/ai/cycles/2026-06-28-004/implementer.md` が追加されていることを確認したが、内容は `blocked: accepted scope not found` であり、ソースコード、README、設計文書、DB schema、migration の変更はない。
  - そのため、実装差分レビューではなく repo 全体レビューを行った。

## 確認した入力

- `AGENTS.md`
- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `.agents/skills/banking-code-review/SKILL.md`
- `.agents/skills/banking-code-review/references/banking-quality-rubric.md`
- `README.md`
- `docs/START_HERE.md`
- `docs/mvp.md`
- `docs/domain-model.md`
- `docs/data-model.md`
- `docs/use-cases.md`
- `docs/design-principles.md`
- `docs/security-notes.md`
- `docs/test-strategy.md`
- `docs/ai/cycles/2026-06-28-001/` から `2026-06-28-003/` の cycle 成果物
- `docs/ai/cycles/2026-06-28-004/implementer.md`
- `cmd/server/main.go`
- `cmd/server/main_test.go`
- `internal/httpapi/router.go`
- `internal/httpapi/router_test.go`
- `go.mod`

## 実行した確認

- `git status --short`
- `git diff --stat`
- `git log --oneline --decorate -5`
- `rg` による TODO/FIXME、PostgreSQL、transaction、冪等性関連語の確認
- `go test ./...`

`go test ./...` は、この実行環境で `go` コマンドが見つからず実行できなかった。前 cycle の implementer / code-reviewer では成功が記録されているが、今回の環境では再検証できていない。

## Finding

### Finding 1: 現行の Go skeleton に修正必須のコード不具合は見つからない

- 重大度: Info

#### 根拠

- `cmd/server/main.go` は `serverConfigFromEnv` と `newServer` により、listen address と timeout 設定を小さく分離している。
- 既定 listen address は `127.0.0.1:8080` で、環境変数 `BANK_SYSTEM_HTTP_ADDR` により明示的に変更できる。
- `http.Server` には `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` が設定されている。
- `internal/httpapi/router.go` は `/healthz` のみを提供し、`GET` 以外を `405 Method Not Allowed` と `Allow: GET` で拒否している。
- `/healthz` は固定 JSON のみを返し、DB 接続情報、環境変数、秘密情報、内部 path、stack trace を返していない。
- `cmd/server/main_test.go` と `internal/httpapi/router_test.go` は、設定値、handler 設定、timeout 非 0、health check の成功、method 拒否、不要情報非露出を検証している。

#### 影響

- 現在の実装範囲は最小 HTTP server と health check に限定されており、Go の責務分離、標準ライブラリ中心の構成、README との整合性は保たれている。
- 現時点では PostgreSQL、残高更新、取引履歴、監査ログ、冪等性、認証認可が未実装のため、それらのコードレベルの金融事故バグは発火しない。

#### 推奨修正

- 現 cycle でこの finding に対する修正は不要。
- 次に endpoint が増えるまでは、`cmd/server` は起動設定、`internal/httpapi` は routing / handler という現在の分離を維持する。

#### 次サイクル planner への入力

- Go skeleton / health check / HTTP server hardening は一旦完了扱いでよい。
- 次 cycle は、業務 API を直接増やす前に、DB transaction 境界、認証認可、エラー分類、監査ログ境界のいずれかを小さな accepted scope にする。

### Finding 2: PostgreSQL schema と transaction 境界が未具体化のままで、業務 API 実装に進むと整合性リスクが高い

- 重大度: High

#### 根拠

- `docs/data-model.md` は `accounts`、`transactions`、`transfer_requests`、`audit_logs` の候補と、`accounts.balance_amount >= 0`、`transactions.amount > 0`、冪等性キー一意制約案を示している。
- `docs/design-principles.md` は、残高非負、金額整数、残高変更と取引履歴、振込の原子性、二重実行防止を原則としている。
- 一方で、実装上の PostgreSQL migration、具体的な `CHECK` / `UNIQUE` / `FOREIGN KEY`、index、transaction manager、repository への transaction 受け渡し、残高更新時の行ロックまたは条件付き更新方針はまだ存在しない。
- 現在の Go 実装には `database/sql` や PostgreSQL adapter がなく、DB transaction 境界を表す package / interface もない。

#### 影響

- 業務 API だけを先に追加すると、handler または service ごとに DB 更新方針がばらつき、残高更新と取引履歴作成が同じ整合性境界に入らない実装になりやすい。
- 並行出金・並行振込で、残高不足判定と残高更新の間に競合が入り、残高マイナスや二重処理が起きるリスクがある。
- DB 制約が後付けになると、既存データの修正、migration の破壊的変更、テストの作り直しが必要になりやすい。

#### 推奨修正

- 業務 API 実装前に、docs scope として PostgreSQL transaction 方針を先に具体化する。
- 最低限、次を決める。
  - `accounts.balance_amount >= 0`、金額正数、口座番号一意、外部キー、冪等性キー一意制約をどの migration に入れるか。
  - 出金・振込で `SELECT ... FOR UPDATE` による行ロックを使うか、条件付き `UPDATE ... WHERE balance_amount >= amount` を使うか。
  - 振込時に複数口座を更新する場合のロック順序とデッドロック回避方針。
  - Go の service 層が transaction を開始し、repository に transaction context を渡す境界。

#### 次サイクル planner への入力

- accepted scope 候補: `docs/transaction-boundary.md` などに、残高更新、取引履歴作成、振込、冪等性、監査ログの DB transaction 境界案を記録する。
- schema 実装に進む場合は、migration ツール、ローカル PostgreSQL 起動方法、test database 方針も同じ scope に含めるか、明示的に別 cycle へ分ける。

### Finding 3: Go の業務レイヤ、エラー分類、HTTP 応答変換の境界が未定義

- 重大度: Medium

#### 根拠

- 現在の package は `cmd/server` と `internal/httpapi` のみで、業務ロジックが存在しないため最小構成としては妥当。
- ただし、今後 MVP の顧客登録、口座作成、入金、出金、振込、照会を追加する際の `domain` / `service` / `repository` / `http handler` の責務境界はまだ決まっていない。
- `docs/use-cases.md` と `docs/test-strategy.md` は、残高不足、権限不足、入力不正、存在しない口座、処理途中エラーなどを分けて扱う必要を示しているが、Go の error type / sentinel error / HTTP status / 監査用 failure reason の対応表はない。

#### 影響

- handler が直接 SQL、認可、残高計算、監査ログ、レスポンス生成を持つ実装になると、テスト対象が肥大化し、トランザクション境界も見えにくくなる。
- エラー分類が曖昧なままだと、残高不足を `500` として返す、内部エラー詳細を利用者に返す、監査ログの失敗理由が実装ごとに異なる、などの保守性・調査性の問題につながる。

#### 推奨修正

- 最初の業務 API 実装前に、最小 package 方針とエラー分類方針を docs に残す。
- 例として、HTTP handler は入力検証、actor 抽出、service 呼び出し、HTTP 応答変換に寄せ、残高変更や認可判定は application/service 層へ置く。
- 業務エラーは、入力不正、認証不足、認可不足、対象なし、状態不正、残高不足、冪等性衝突、内部エラーを最低限区別できる形にする。

#### 次サイクル planner への入力

- accepted scope 候補: 業務 API 実装前の `internal` package layout と error mapping の設計文書を作る。
- 実装に進む場合でも、最初の API は 1 フローに限定し、handler / service / repository の境界とテストを同時に確認する。

### Finding 4: cycle 004 は accepted scope 不在のため実装が blocked で、次 cycle の入力が不足している

- 重大度: Medium

#### 根拠

- `docs/ai/cycles/2026-06-28-004/implementer.md` は、同一 cycle の `planner.md` が存在せず accepted scope を確認できないため、`blocked: accepted scope not found` と記録している。
- `.codex/agents/README.md` と `docs/ai/cycles/README.md` は、implementer が同一 cycle の accepted scope なしに実装しないことを定めている。
- 現在の `docs/ai/cycles/2026-06-28-004/` には `implementer.md` のみが存在し、planner の採択判断はない。

#### 影響

- 次に何を実装するかが cycle artifact 上で固定されていないため、業務 API、DB schema、認証、監査ログなどの重い判断が混ざった実装に進むリスクがある。
- reviewer の出力は次 cycle planner への入力として残せるが、cycle 004 内の implementer はこのままでは実装を進められない。

#### 推奨修正

- 次に implementer を動かす前に、planner が accepted scope を作る。
- accepted scope には、対象ファイル、実装すること、実装しないこと、テスト方針、人間確認事項を明記する。
- 金融ドメイン上の後戻りが難しい判断は、人間確認事項として分離する。

#### 次サイクル planner への入力

- cycle 004 の implementer は blocked として扱い、次 cycle では planner を先に実行する。
- 採択候補は、次のどれか 1 つに絞るのが望ましい。
  - DB transaction 境界と残高更新方針の docs 化。
  - 認証/RBAC/セッションまたは token 方針の docs 化。
  - API エラー形式、入力検証、ログ/監査マスキング方針の docs 化。
  - PostgreSQL migration 方針と最小 schema の設計。ただし冪等性 scope と残高競合制御の人間確認を先に行う。

## 補足

### PostgreSQL / transaction

- PostgreSQL は未実装であり、現時点で transaction bug はコード上には存在しない。
- ただし、docs 上の MVP はすでに残高変更、取引履歴、振込、冪等性、監査ログを要求しているため、次の設計 scope で DB transaction 境界を扱う優先度は高い。

### テスト

- 現在の test file は health check と server config に集中しており、現行実装範囲には合っている。
- 今回の環境では `go` コマンドがなく `go test ./...` を再実行できなかった。次 cycle では、実装環境に Go toolchain があること、または CI / 開発環境での検証結果を cycle 成果物に明記する必要がある。

### 人間確認事項

1. 業務 API 実装前に、認証基盤を先に設計するか、DB transaction / 元帳境界を先に設計するか。
2. 出金・振込の残高競合制御は、行ロック方式と条件付き更新方式のどちらを MVP の学習対象にするか。
3. 冪等性キーの一意 scope は、`requested_by_user_id + idempotency_key`、`source_account_id + idempotency_key`、または別の scope のどれにするか。
4. 監査ログ書き込み失敗時に、業務処理を失敗させるか、業務処理を優先して別途調査対象にするか。
