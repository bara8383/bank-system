# security-reviewer: 2026-06-29-002

## レビュー前提

- 役割: `security-reviewer` として、認証、認可、入力検証、ログ、秘密情報、SQL injection、権限境界、監査証跡の観点でレビューした。
- 連携方法: agent 間の直接同期は行わず、`docs/ai/cycles/2026-06-29-002/` の `planner.md` と `implementer.md` を入力 artifact として参照した。
- 優先範囲: 実装差分レビューを優先した。今回の実装差分は `HEAD^..HEAD` の `README.md`, `docs/ai/cycles/2026-06-29-002/implementer.md`, `internal/domain/money.go`, `internal/domain/money_test.go`。
- 作業仮定: 今回の accepted scope は金額・残高の domain helper に限定され、業務 API、DB、認証、認可、監査ログ永続化、冪等性キー、行ロックは実装しない。

## 確認した資料・コマンド

- `AGENTS.md`
- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `README.md`
- `docs/START_HERE.md`
- `docs/design-principles.md`
- `docs/security-notes.md`
- `docs/data-model.md`
- `docs/domain-model.md`
- `docs/mvp.md`
- `docs/use-cases.md`
- `docs/test-strategy.md`
- `docs/ai/output/human/001-human-review.md`
- `docs/ai/cycles/2026-06-29-002/planner.md`
- `docs/ai/cycles/2026-06-29-002/implementer.md`
- `internal/domain/money.go`
- `internal/domain/money_test.go`
- `git status --short`
- `git diff --name-only HEAD^..HEAD`
- `go test ./...`
- `rg -n "float32|float64" internal/domain`
- `rg -n "database/sql|Exec\(|Query\(|QueryRow\(|fmt\.Sprintf|SELECT|INSERT|UPDATE|DELETE" . --glob '*.go'`

## Finding

### S-001: 今回差分では認証・認可境界を広げる実装は追加されていない（重大度: 情報）

#### 根拠

- `planner.md` の accepted scope は、金額・残高 validation の domain 実装、unit test、README、implementer 成果物に限定している。
- `implementer.md` は、HTTP route、handler、request / response schema、業務 API、認証、認可、ユーザー登録、Cookie session、CSRF token、監査ログ永続化を実装していないと明記している。
- 実装差分 `HEAD^..HEAD` は `internal/domain/money.go` と `internal/domain/money_test.go` の追加、README と implementer 成果物の更新に限定されている。
- `internal/domain/money.go` は pure domain helper であり、HTTP request、session、user、role、account owner、DB 接続、外部入力 source を直接扱っていない。

#### 影響

- 今回差分単体では、未認証アクセス、水平権限昇格、管理者権限の濫用、CSRF、セッション固定などの直接的な攻撃面は増えていない。
- 一方で、将来の入金・出金・振込 API がこの helper を使う段階では、金額 validation が通ったことを認可済みと誤解しない設計が必要になる。金額が妥当でも、対象口座の所有者確認、口座状態確認、ロール確認、監査ログ記録は別に必須である。

#### 推奨修正

- 今回差分への修正は不要。
- 将来の API / use case 層では、`NewAmount` / `NewBalance` / `AddBalance` / `SubtractBalance` を「入力値・残高不変条件の helper」としてのみ扱い、認証・認可・口座状態・冪等性・監査ログの通過条件とは分離する。

#### 次サイクル planner への入力

- 業務 API 実装前に、入金・出金・振込それぞれの handler / use case で必ず実行する security gate を docs または skeleton に落とす候補を検討する。
  - 認証済みユーザーの確認。
  - 顧客本人または `admin` の権限確認。
  - MVP では `operator` を対象外にする確認。
  - 口座状態の確認。
  - 金額 validation。
  - 成功・失敗監査ログの記録境界。

### S-002: 金額・残高 validation は早期拒否として有効だが、取引上限・残高上限の業務制限は未定義（重大度: Low / 次工程リスク）

#### 根拠

- `internal/domain/money.go` の `NewAmount` は 0 以下を拒否し、正の `int64` を受け付ける。
- `NewBalance` は負の残高を拒否する。
- `AddBalance` は `int64` overflow を検出して元残高と `ErrBalanceOverflow` を返す。
- `SubtractBalance` は 0 以下の amount と残高不足を拒否し、元残高を返す。
- `docs/security-notes.md` と `docs/design-principles.md` は「金額は正の整数」「残高はマイナスにしない」を要求しているが、1 回あたりの取引金額上限、口座残高上限、日次上限、管理者操作上限はまだ定義していない。

#### 影響

- 現時点では外部入力を受ける API がないため、直ちに攻撃可能な脆弱性ではない。
- 将来 API へ接続した場合、`math.MaxInt64` 近い金額など、技術的には valid だが業務上不自然な金額を validation 通過させる余地がある。
- 入金 API や管理者操作が先に実装されると、誤操作・権限侵害・テストデータ汚染・監査調査困難につながる可能性がある。

#### 推奨修正

- 今回差分では accepted scope 外なので修正不要。
- 次に業務 API へ進む前に、少なくとも MVP 用の暫定上限を docs に明記する。
  - 1 回あたりの入金・出金・振込上限。
  - 口座残高上限。
  - 管理者操作で上限を超えられるかどうか。
  - 上限超過時の error 分類と失敗監査ログ分類。
- 実装時は domain helper に上限を含めるか、use case 層で operation-specific limit として扱うかを明確に分ける。

#### 次サイクル planner への入力

- 「MVP 金額上限・残高上限・上限超過時の監査分類」を docs-only scope として採択候補にする。
- 特に振込 API の前に、冪等性キー、request body hash、失敗監査ログの failure reason と合わせて整理する。

### S-003: SQL injection と秘密情報漏えいの新規リスクは確認されなかった（重大度: 情報）

#### 根拠

- 実装差分は Go 標準ライブラリの `errors` と `math` のみを使う domain helper で、SQL、DB driver、repository、migration は追加していない。
- `rg -n "database/sql|Exec\(|Query\(|QueryRow\(|fmt\.Sprintf|SELECT|INSERT|UPDATE|DELETE" . --glob '*.go'` では Go ファイル内に SQL 構築・実行パターンは見つからなかった。
- `internal/domain/money.go` の error message は固定文字列で、password、token、secret、CSRF token、session ID、raw request body、個人情報、口座番号、残高詳細を含まない。
- `/healthz` は既存の固定レスポンスで、今回差分では変更されていない。

#### 影響

- 今回差分によって SQL injection 攻撃面は増えていない。
- 今回差分によってログ・レスポンスに秘密情報や個人情報を出す経路も増えていない。
- ただし、将来 HTTP error mapping で sentinel error をそのまま外部レスポンスへ出す場合は、エラー文言の安定性・情報量・監査ログ分類を別途設計する必要がある。

#### 推奨修正

- 今回差分への修正は不要。
- 将来の repository 実装では、SQL 文字列連結や `fmt.Sprintf` に外部入力を混ぜず、placeholder / parameter binding を使う。
- 将来の handler 実装では、domain error をそのままログやレスポンスに流すのではなく、安全な API error code と監査ログ用 failure reason に map する。

#### 次サイクル planner への入力

- DB schema / repository の実装前に、SQL parameter binding、transaction 境界、監査ログ failure reason の安全な分類を accepted scope 候補にする。
- HTTP API 実装前に、domain error から response code / response body / audit failure reason への mapping 方針を docs 化する。

### S-004: 監査証跡そのものは未実装だが、今回 helper は監査ログの信頼境界を汚染していない（重大度: 情報）

#### 根拠

- `implementer.md` は、監査ログ永続化、監査ログ正規化、監査ログ照会、outbox、非同期補償を実装していないと明記している。
- `internal/domain/money.go` は、`ip_address`、`user_agent`、request body、actor、target、action type、failure reason を扱わない。
- 既存 docs は、監査ログに raw request body、パスワード、token、secret、CSRF token、セッション ID、過剰な個人情報を保存しない方針を持つ。

#### 影響

- 今回差分では監査ログ注入、改ざん、秘密情報混入、proxy header 偽装のリスクは増えていない。
- 一方で、将来の入出金・振込 API 実装時には、金額 validation 失敗、残高不足、overflow、認可失敗を安全な `failure_reason` に分類し、業務データ更新なしで失敗監査ログを独立して残す必要がある。

#### 推奨修正

- 今回差分への修正は不要。
- 監査ログ実装前に、`ErrAmountMustBePositive`, `ErrBalanceMustBeNonNegative`, `ErrInsufficientBalance`, `ErrBalanceOverflow` をどの監査分類に map するか定義する。
- `ip_address` / `user_agent` の最大長、制御文字除去、信頼する reverse proxy header の有無を docs 化してから永続化する。

#### 次サイクル planner への入力

- 直近 planner の保留候補 D「監査ログ属性の正規化・信頼境界を docs に具体化する」は security 優先度が高い。業務 API / 監査ログ永続化の前に採択候補として維持する。
- 失敗監査ログ分類に、入力不備、残高不足、overflow、権限不足、対象不存在、口座状態不正を含める。

## 総合判断

- 今回の money domain helper 差分に、認証・認可 bypass、SQL injection、秘密情報漏えい、ログ注入、監査ログ汚染につながる直接的なブロッカーは確認しなかった。
- セキュリティ観点の主な残リスクは、今回差分ではなく、次工程で domain helper を HTTP / DB / 認証認可 / 監査ログに接続する際の境界設計にある。
- 次サイクルでは、業務 API に進む前の docs scope として、金額上限、失敗監査分類、監査ログ属性正規化、domain error mapping を優先候補にすることを推奨する。

## 確認結果

- `git status --short`: レビュー開始時点では未コミット変更なし。
- `git diff --name-only HEAD^..HEAD`: `README.md`, `docs/ai/cycles/2026-06-29-002/implementer.md`, `internal/domain/money.go`, `internal/domain/money_test.go`。
- `go test ./...`: 成功。
- `rg -n "float32|float64" internal/domain`: 該当なし。金額・残高 domain helper に浮動小数点型は確認されなかった。
- `rg -n "database/sql|Exec\(|Query\(|QueryRow\(|fmt\.Sprintf|SELECT|INSERT|UPDATE|DELETE" . --glob '*.go'`: 該当なし。Go ファイル内に SQL 構築・実行パターンは確認されなかった。
