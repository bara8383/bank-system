# code-reviewer: 2026-07-06-001

## レビュー範囲

- 依頼どおり `code-reviewer` として、Go / 設計 / 保守性 / テスト観点で実装差分を優先レビューした。
- 直接同期は行わず、同一 cycle の成果物 `planner.md` / `implementer.md` と repo 内ファイルだけを根拠にした。
- レビュー対象差分は最新実装 commit `95e2fd9 Add domain failure reason mapping`（`HEAD^..HEAD`）とした。差分は `README.md`、`docs/security-notes.md`、`internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`docs/ai/cycles/2026-07-06-001/implementer.md` に限定されている。

## Finding

### Finding 1: Blocking なし — safe failure category helper は accepted scope に沿っている

- 重大度: なし。
- `FailureReason` は string-based type として定義され、MVP 初期の固定分類 `invalid_amount`、`invalid_balance_state`、`insufficient_balance`、`balance_overflow`、`invalid_account_status`、`account_not_active`、`invalid_transaction_type` だけを valid としている。
- `FailureReasonFromError` は `errors.Is` を使って既存 domain sentinel error と wrapped error を分類し、`nil`、未知 error、対象外 error は `"", false` として未分類にしている。
- raw error message や自由入力をそのまま category として返す処理は確認しなかった。
- 実装は pure Go の `internal/domain` helper に留まり、HTTP route、DB 接続、SQL、repository、transaction manager、認証、認可、監査ログ永続化、冪等性キー処理を追加していない。

### Finding 2: Blocking なし — テストは今回 scope の主要な保守性リスクを固定している

- 重大度: なし。
- 新規テストは valid category の whitelist、空文字・未知値・raw sentinel error 文字列・secret 風値の拒否、各 sentinel error の mapping、wrapped error の mapping、`nil` / 未知 error の未分類、mapping 後の `Validate()` 成功を確認している。
- これにより、後続 API / audit 実装が `Error()` 文字列や自由入力を failure category として誤用するリスクを domain unit test で検知しやすくなった。
- `go test ./...`、`gofmt -l`、`git diff --check HEAD^..HEAD` は問題なしだった。

### Finding 3: 注意点 — 複数 sentinel error を含む joined / wrapped error の優先順位は未設計

- 重大度: 低。今回 scope の blocking ではない。
- 現在の `FailureReasonFromError` は `switch` の上から順に `errors.Is` を評価するため、将来 `errors.Join` などで複数の domain sentinel error を 1 つの error に含めた場合は、呼び出し側が意図した主原因ではなく実装順に基づく分類が返る可能性がある。
- 現時点の domain helper 群は単一 sentinel error を返しており、今回追加されたテストも単一 error / wrapped 単一 error を前提としているため、今すぐ修正必須とは判断しない。
- ただし、後続の service / repository / audit 実装で複数原因を join する設計を採る場合は、primary failure reason を明示する wrapper、優先順位表、または複数分類を扱わない方針を先に決めると保守しやすい。

## 根拠

- `docs/ai/cycles/2026-07-06-001/planner.md` の accepted scope は、既存 domain sentinel error を安全な固定 category へ写像する pure Go helper、対応 unit test、README / security docs 更新に限定していた。
- `docs/ai/cycles/2026-07-06-001/implementer.md` は、変更対象を `internal/domain` の failure reason helper とテスト、README、security docs、同一 cycle artifact に限定したと記録している。
- `internal/domain/failure_reason.go` では `FailureReason` constants、`ErrInvalidFailureReason`、whitelist 型の `Validate()`、`errors.Is` ベースの `FailureReasonFromError` が実装されている。
- `internal/domain/failure_reason_test.go` では、accepted scope が求めた valid / invalid / raw error string / secret-like value / sentinel mapping / wrapped mapping / unknown error のケースがテストされている。
- `README.md` は helper を現在の実装範囲に追記しつつ、HTTP error response、監査ログ永続化、DB schema は未実装と明記している。
- `docs/security-notes.md` は domain error 由来の監査 `failure_reason` に固定分類値を使い、raw request body、password、token、secret、CSRF token、セッションID、未加工の自由入力値を保存・返却する用途ではないと明記している。
- 実行した確認:
  - `git status --short`
  - `git show --stat --oneline --decorate --name-status HEAD`
  - `git show --find-renames --find-copies --stat --patch --unified=80 --format=fuller HEAD -- README.md docs/security-notes.md internal/domain/failure_reason.go internal/domain/failure_reason_test.go docs/ai/cycles/2026-07-06-001/implementer.md`
  - `gofmt -l internal/domain/failure_reason.go internal/domain/failure_reason_test.go`
  - `git diff --check HEAD^..HEAD`
  - `go test ./...`

## 影響

- 今回の helper により、後続の HTTP error response、監査ログ、構造化ログが domain sentinel error の raw `Error()` 文字列や自由入力値へ直接依存しにくくなる。
- `errors.Is` を使っているため、service 層が domain error を文脈付きで wrap しても、既知の sentinel error を分類できる。
- whitelist 型の `Validate()` により、DB insert 境界や audit writer 境界で failure category を再検証する余地ができた。
- 一方で、この helper はあくまで分類候補であり、HTTP status code、API response schema、監査ログ table、成功 / 失敗監査ログの transaction 境界、DB constraint、認証・認可、冪等性はまだ保証しない。後続 cycle で helper の存在をもって監査ログ実装済み、または API error 設計完了と誤解しないことが重要である。

## 推奨修正

- 今回差分への修正必須事項はない。
- 次に service / HTTP / audit へ進む前に、`FailureReason` をどの境界で `Validate()` するかを設計することを推奨する。特に audit repository / DB insert 境界では、固定分類以外を保存しない最終防衛線として使うとよい。
- 複数原因 error を扱う必要が出た場合は、`FailureReasonFromError` の優先順位を暗黙の `switch` 順に任せず、primary error を明示する設計または join しない方針を docs / tests に残すことを推奨する。
- API response を追加する cycle では、domain `FailureReason` をそのまま利用者向け message と混同せず、利用者向け汎用 message、HTTP status、audit `failure_reason`、internal log の責務を分けることを推奨する。
- DB schema / migration を追加する cycle では、将来の `audit_logs.failure_reason` に CHECK constraint を置くか、Go 側 whitelist のみで扱うかを planner で決めることを推奨する。

## 次サイクル planner への入力

1. **service gate 順序の docs 化**
   - 入金・出金・振込 service 前に、認証、CSRF、owner / role authorization、口座存在、口座状態、金額、取引種別、冪等性キー、PostgreSQL row lock、残高計算、取引履歴、成功監査ログ、失敗監査ログの順序を固定する。
2. **API response / audit mapping の境界設計**
   - 今回追加された `FailureReason` は audit / safe log の候補として使いつつ、利用者向け message や HTTP status code とは別に設計する。
   - domain error が複数ある場合の primary reason 方針も合わせて決める。
3. **audit_logs の failure_reason 制約**
   - DB schema に進む場合、`audit_logs.failure_reason` を nullable にするか、MVP 固定 category の CHECK constraint を持たせるか、unknown / unclassified を保存するかを検討する。
4. **DB transaction / row lock とテスト計画**
   - PostgreSQL 実装へ進む場合、成功時は残高更新・取引履歴・成功監査ログを同一 transaction、失敗監査ログは独立 transaction とする方針を test strategy とセットで accepted scope に入れる。
5. **冪等性キー設計**
   - human notes の「操作種別、送信元口座、ログインユーザー、request body hash」を含める案を、DB UNIQUE constraint、重複時拒否 response、監査 `failure_reason` と接続して具体化する。

## 作業仮定

- 本レビューは学習用ミニバンキングシステムとしての品質確認であり、本番金融システムとしての十分性は断定しない。
- 現時点では helper の直接利用箇所がないため、複数 sentinel error の優先順位は将来 service 実装時の設計課題として扱った。
- reviewer としてソースコードや設計文書は変更せず、この Markdown 成果物のみを作成した。
