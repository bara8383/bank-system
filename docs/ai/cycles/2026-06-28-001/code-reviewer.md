# code-reviewer review: 2026-06-28-001

## レビュー種別

- 種別: repo-wide review
- 理由: `git status --short` と `git diff --stat` に実装差分がなく、同一 cycle の `implementer.md` も `blocked: accepted scope not found` と記録しているため。
- 対象: 現在の repo 全体。実装済みコードは Go 標準ライブラリのみの最小 HTTP server と `/healthz` handler。PostgreSQL、migration、業務 API、認証認可、残高更新、取引履歴、監査ログは未実装。
- 確認した入力: `.codex/agents/README.md`、`docs/ai/cycles/README.md`、`.agents/skills/banking-code-review/SKILL.md`、`README.md`、`AGENTS.md`、`docs/START_HERE.md`、主要 `docs/*.md`、同一 cycle の planner / implementer / security-reviewer / banking-reviewer 成果物。
- テスト: `go test ./...` を試行したが、この環境では `/bin/bash: line 1: go: command not found` となり未実行。

## 全体所見

現在の Go 実装は、`cmd/server/main.go` と `internal/httpapi/router.go` に責務が分かれており、`main` へ handler を直書きしていない。`/healthz` は固定 JSON のみを返し、DB 接続情報や秘密情報を返さない。現時点の実装範囲に対して、Go の構成・依存最小化・テスト容易性に大きなコード品質問題は見つからなかった。

一方で、次に DB や業務 API へ進む前に解消すべき設計・運用上のリスクがある。特に cycle 成果物と現在 repo の実態が食い違っている点、Go テストをこの環境で実行できない点、PostgreSQL トランザクション境界とロック方針がまだ実装可能な粒度に落ちていない点は、次サイクル planner の入力にするべきである。

## Finding 1: 同一 cycle 成果物が現在 repo の実態と食い違っている

- 重大度: Medium
- 人間確認要否: なし

### 根拠

- `docs/ai/cycles/2026-06-28-001/implementer.md:5` は `planner.md` が存在せず accepted scope を確認できなかったとしている。
- 同ファイル `docs/ai/cycles/2026-06-28-001/implementer.md:13` は、コード・設計文書・README の実装変更を行わなかったと記録している。
- 一方で現在の repo には `go.mod`、`cmd/server/main.go`、`internal/httpapi/router.go`、各テスト、`README.md` が存在する。
- `docs/ai/cycles/2026-06-28-001/planner.md:21` と `docs/ai/cycles/2026-06-28-001/planner.md:26` も、Go ソースや `go.mod` が存在しない前提で記録されている。

### 影響

次サイクル planner が同一 cycle の Markdown 成果物をそのまま信じると、「実装なし」「README なし」と誤認し、既に存在する最小 HTTP server を再作成する scope や、現状と合わないレビュー入力を作る可能性がある。並列 cycle の連携を Markdown 成果物だけに限定する運用では、成果物の鮮度不一致は実装重複や不要な差分の原因になる。

### 推奨修正

次サイクル planner は、過去 cycle 成果物を読む際に必ず現在の `git status`、`rg --files`、`README.md`、Go ソースの存在を再確認し、古い cycle 成果物の「実装なし」記録を現在状態として扱わない。必要なら次 cycle の planner 出力で「2026-06-28-001 の implementer 成果物は現在 repo と不一致」と明記する。

### 次サイクル planner への入力

- accepted scope を作る前に、現在 repo の実装済み範囲を再棚卸しする。
- 「最小 Go REST API 土台」は既に存在する前提で、重複実装ではなく次の小さな改善に進む。
- 古い cycle 成果物と現 repo の不一致を human notes または planner の入力レビューに記録する。

## Finding 2: Go テスト実行環境が不足しており、`go test ./...` を確認できない

- 重大度: Medium
- 人間確認要否: なし

### 根拠

- `README.md:43` から `README.md:47` は `go test ./...` をテスト方法として案内している。
- `go.mod:3` は `go 1.24` を指定している。
- 本レビュー環境で `go test ./...` を実行したところ、`go` コマンドが存在せず未実行だった。

### 影響

現時点のコードは小さいが、今後 PostgreSQL、残高更新、振込、冪等性、監査ログが入ると、テスト実行不能は金融品質上の大きな盲点になる。特に DB トランザクションや並行更新はレビューだけで担保できず、自動テストが実行できる環境が必要になる。また Go バージョンの導入方法が repo 内にないため、開発者や subagent ごとに検証可否がばらつく。

### 推奨修正

次サイクルで、実装変更に入る前または同時に Go ツールチェーン前提を明文化する。最低限、README または docs に必要な Go version、インストール前提、`go test ./...` が通る環境を記録する。CI を導入できるなら、最小の `go test ./...` を実行する workflow を追加する。

### 次サイクル planner への入力

- accepted scope 候補: 「Go テスト実行環境の前提を README/docs に明記し、可能なら CI で `go test ./...` を実行する」
- 人間確認事項: CI を導入してよいか。導入する場合、GitHub Actions 等の利用可否を確認する。

## Finding 3: 次の業務 API 実装前に、PostgreSQL トランザクション境界とロック方針が未確定

- 重大度: High
- 人間確認要否: 一部あり

### 根拠

- `README.md:57` は PostgreSQL 接続、DB schema、migration、transaction 処理が未実装であると明記している。
- `docs/design-principles.md` は、残高非負、残高変更ごとの取引履歴、振込の原子性、冪等性を原則としている。
- `docs/data-model.md` は `accounts.balance_amount >= 0`、`transactions.amount > 0`、`transfer_requests.idempotency_key` の一意性を制約案としている。
- しかし、PostgreSQL の `CHECK` / `UNIQUE` / `FOREIGN KEY`、出金・振込時の `SELECT ... FOR UPDATE` または条件付き `UPDATE`、複数口座ロック順序、repository へ transaction を渡す方式はまだ具体化されていない。

### 影響

次に入金・出金・振込 API を実装する際、handler や service が「残高を読む、Go 側で判定する、後で更新する」という形になると、同時出金や同時振込で lost update や過剰出金が発生し得る。DB 制約を後付けにすると、既存データとの整合、migration、テスト修正のコストも上がる。

### 推奨修正

業務 API 実装前に、PostgreSQL 前提の最小トランザクション方針を docs に固定する。出金・振込は、対象口座行を transaction 内でロックしてから残高判定・更新する方式、または `balance_amount >= amount` を条件にした単一 `UPDATE` の更新件数で成功判定する方式のどちらかを選ぶ。振込では振込元・振込先のロック順序を口座 ID 昇順などに固定し、DB 制約として残高非負、金額正数、冪等性キー一意制約を初回 migration に含める。

### 次サイクル planner への入力

- accepted scope 候補: 「PostgreSQL 残高更新・振込トランザクション方針を設計文書へ追加する」
- accepted scope 候補: 「初期 migration 方針として `CHECK`、`UNIQUE`、`FOREIGN KEY`、主要 index を具体化する」
- 人間確認事項: DB migration ツール、ID 型、冪等性キーの一意スコープ、口座番号採番方針。

## Finding 4: 業務 API 向けのエラー応答形式と handler 境界が未定義

- 重大度: Medium
- 人間確認要否: なし

### 根拠

- `internal/httpapi/router.go:14` から `internal/httpapi/router.go:24` の `/healthz` handler は現状の範囲では十分小さい。
- ただし unsupported method では `internal/httpapi/router.go:17` で `http.Error` を使い、plain text のエラーを返している。
- 業務 API で必要になる入力不正、認証失敗、認可失敗、残高不足、冪等性キー衝突、DB 障害の分類と JSON エラー形式はまだ定義されていない。

### 影響

このまま業務 handler を増やすと、handler ごとにエラー本文、HTTP status、ログ出力、監査用 failure reason の扱いがばらつきやすい。金融系では、利用者向けには安全なエラーを返しつつ、運用調査・監査には十分な分類を残す必要がある。エラー分類が遅れると、テストも status code と body の期待値を後から大きく直すことになる。

### 推奨修正

業務 API を追加する前に、最小の API エラー標準を決める。例として、JSON の共通エラー envelope、業務エラー種別、HTTP status 対応、利用者向け message と内部ログ/監査用 reason の分離を定義する。`/healthz` は固定レスポンスのままでよいが、業務 API では共通 error writer を使う境界を作る。

### 次サイクル planner への入力

- accepted scope 候補: 「REST API のエラー応答形式、エラー分類、handler/service 境界を docs に追加する」
- 次の実装 scope で業務 API を作る場合、共通 error writer とテストを最初に含める。

## Finding 5: 現在の最小 HTTP server 自体には大きなコード品質問題は見つからない

- 重大度: Info
- 人間確認要否: なし

### 根拠

- `cmd/server/main.go:26` から `cmd/server/main.go:39` で設定読み取りが分離されている。
- `cmd/server/main.go:41` から `cmd/server/main.go:49` で `http.Server` 生成が分離され、timeout が設定されている。
- `internal/httpapi/router.go:8` から `internal/httpapi/router.go:11` で router 生成が `main` から分離されている。
- `internal/httpapi/router_test.go` と `cmd/server/main_test.go` に handler と server config のテストがある。
- 外部依存はなく、現時点では DB・認証・業務 API を暗黙に確定していない。

### 影響

現在のスコープでは、過剰な抽象化や不要な外部依存は見当たらない。次の機能追加時にこの小ささを維持し、handler に SQL や金融業務ルールを直接入れないことが重要である。

### 推奨修正

現状の `main` / `httpapi` 分離は維持する。次に業務 API を追加する場合は、HTTP handler、application/service、repository/postgres の責務を分け、DB transaction を application/service 層で制御できる形にする。

### 次サイクル planner への入力

- 次の accepted scope では、現在の最小構成を壊さず、1 つの設計文書化または 1 つの小さな縦切りに限定する。
- 業務ロジックを `internal/httpapi` に寄せすぎない package 方針を先に決める。

## 人間確認事項

1. CI を導入して `go test ./...` を機械的に確認してよいか。
2. PostgreSQL migration ツールを何にするか。
3. 残高更新方式を、行ロック方式と条件付き `UPDATE` 方式のどちらで学習・実装するか。
4. 冪等性キーの一意スコープを `requested_by_user_id` 基準、`source_account_id` 基準、または別の組み合わせにするか。
