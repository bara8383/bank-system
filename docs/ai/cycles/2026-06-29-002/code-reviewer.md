# code-reviewer: 2026-06-29-002

## レビュー対象

- cycle: `2026-06-29-002`
- role: `code-reviewer`
- 参照 artifact: `docs/ai/cycles/2026-06-29-002/planner.md`, `docs/ai/cycles/2026-06-29-002/implementer.md`
- 実装差分: `git diff 1c95703..HEAD`（planner commit 後から implementer commit まで）
- 差分ファイル:
  - `README.md`
  - `docs/ai/cycles/2026-06-29-002/implementer.md`
  - `internal/domain/money.go`
  - `internal/domain/money_test.go`

## 確認した入力

- `AGENTS.md`
- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `git status --short --branch`
- `README.md`
- `docs/START_HERE.md`
- `docs/design-principles.md`
- `docs/data-model.md`
- `docs/domain-model.md`
- `docs/mvp.md`
- `docs/security-notes.md`
- `docs/test-strategy.md`
- `docs/use-cases.md`
- `docs/ai/output/human/001-human-review.md`
- `docs/ai/cycles/2026-06-29-002/planner.md`
- `docs/ai/cycles/2026-06-29-002/implementer.md`
- 実装差分と関連コード: `internal/domain/money.go`, `internal/domain/money_test.go`, `cmd/server/*`, `internal/httpapi/*`, `go.mod`

## Finding

### Finding なし（blocking / must-fix 指摘なし）

今回の accepted scope「金額・残高の最小 domain 型と validation」を満たしており、Go / 設計 / 保守性 / テスト観点で、この cycle 内で修正必須と判断する実装不備は見つからなかった。

## 根拠

- `internal/domain/money.go` は、金額を `int64` の最小通貨単位として扱い、`NewAmount` で 0 以下を拒否している。
- `internal/domain/money.go` は、残高を `int64` の最小通貨単位として扱い、`NewBalance` で負数を拒否している。
- `AddBalance` は 0 以下の `Amount` を拒否し、`math.MaxInt64` を超える加算を `ErrBalanceOverflow` として元残高を返すため、overflow による負残高化を避けている。
- `SubtractBalance` は 0 以下の `Amount` を拒否し、残高不足時に `ErrInsufficientBalance` と元残高を返すため、呼び出し側が残高非変更を維持できる API になっている。
- `Amount` / `Balance` の内部値は unexported field であり、外部 package からは constructor を経由する形に寄せられている。
- `internal/domain/money_test.go` は、正の取引金額、0 円・負の取引金額拒否、0 円・正の残高許可、負の残高拒否、加算、残高内減算、残高不足時の元残高維持、invalid amount、overflow を確認している。
- `README.md` は、現在の実装範囲に domain helper を追加しつつ、DB 接続・認証・業務 API が未実装である点を維持している。
- 差分は planner の accepted scope に含まれる domain package、unit test、README、implementer artifact に限定されており、HTTP route、DB、repository、認証、監査ログ、冪等性には踏み込んでいない。
- `go test ./...` は成功した。
- `gofmt -l internal/domain/money.go internal/domain/money_test.go cmd/server/main.go cmd/server/main_test.go internal/httpapi/router.go internal/httpapi/router_test.go` は出力なしで、対象 Go ファイルに未整形は見つからなかった。
- `rg -n "float32|float64" internal/domain` は該当なしで、domain 金額実装に浮動小数点型は見つからなかった。

## 影響

- 今回差分は、後続の入金・出金・振込 service / repository 実装が使える最小 domain 不変条件を追加している。
- 修正必須の問題は見つからないため、この cycle の実装差分は次の reviewer / planner 入力として受け入れ可能と判断する。
- ただし、現時点の helper はメモリ上の純粋な値計算に限定される。PostgreSQL の `CHECK` 制約、行ロック、DB transaction、監査ログ、冪等性キー、HTTP error mapping は未実装であり、将来の業務 API 実装時に別途設計・テストが必要である。

## 推奨修正

- この cycle 内の必須修正はなし。
- 将来の service / repository 実装では、`NewAmount` / `NewBalance` / `AddBalance` / `SubtractBalance` の結果を DB 制約・transaction 境界・監査ログ作成と二重化して検証すること。
- 将来の API 実装では、sentinel error を HTTP status / error response へ mapping する層を domain package 外に置き、domain package を HTTP / DB / 認証の詳細に依存させないこと。
- 将来の PostgreSQL schema では、`transactions.amount > 0`、`accounts.balance_amount >= 0`、`transactions.balance_after >= 0` の `CHECK` 制約を domain helper と対応させること。

## 次サイクル planner への入力

- DB / repository 実装に進む前に、PostgreSQL schema の `CHECK` 制約と domain helper の対応を accepted scope 化する候補を検討する。
- 残高更新を伴う入金・出金・振込の service 層を作る際は、今回の helper を使った unit test に加え、DB transaction 内で「残高更新・取引履歴・成功監査ログ」が同時に commit / rollback される結合テストを scope に含める。
- PostgreSQL 行ロック方針（`SELECT ... FOR UPDATE`、2 口座振込時の lock order、deadlock retry / fail 方針）は、残高更新実装前の高優先 docs scope として残す。
- 冪等性キー衝突時の扱い、request body hash、成功済み同一キー再送の MVP 挙動は、振込 API 実装前に docs scope として確定する。
- 監査ログ書き込み失敗時の fail closed 方針は既存 docs にあるため、transaction manager / audit repository 実装前にテスト注入点まで具体化する。

## 確認コマンド

- `git status --short --branch`
- `git log --oneline --decorate -n 12`
- `git diff --stat 1c95703..HEAD`
- `git diff --name-status 1c95703..HEAD`
- `git diff --find-renames --find-copies 1c95703..HEAD -- README.md internal/domain/money.go internal/domain/money_test.go docs/ai/cycles/2026-06-29-002/implementer.md`
- `gofmt -l internal/domain/money.go internal/domain/money_test.go cmd/server/main.go cmd/server/main_test.go internal/httpapi/router.go internal/httpapi/router_test.go`
- `go test ./...`
- `rg -n "float32|float64" internal/domain`
- `git diff --name-only 1c95703..HEAD`
