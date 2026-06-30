# banking-reviewer: 2026-06-30-001

## 確認した入力

- 作業開始時に `git status --short` を確認し、未コミット変更はなかった。
- `README.md`、`AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md` を確認した。
- `docs/START_HERE.md`、`docs/mvp.md`、`docs/domain-model.md`、`docs/data-model.md`、`docs/design-principles.md`、`docs/security-notes.md`、`docs/test-strategy.md`、`docs/use-cases.md`、`docs/ai/output/human/001-human-review.md`、`docs/ai/output/human/002-human-review.md` を確認した。
- agent 間の直接同期は行わず、`docs/ai/cycles/2026-06-30-001/` の `planner.md`、`implementer.md`、`code-reviewer.md`、`security-reviewer.md` と repo 差分だけを入力にした。
- 実装差分レビューを優先し、`HEAD^..HEAD` の `README.md`、`internal/domain/money.go`、`internal/domain/money_test.go`、`docs/ai/cycles/2026-06-30-001/implementer.md` を確認した。
- 実行確認として `go test ./...` と `rg -n "float32|float64" internal/domain` を実行した。後者は一致なしで、金額・残高 domain helper に浮動小数点利用がないことを確認した。

## Finding 1: blocking なし。開始残高の再検証により破損残高からの演算継続リスクは低下している

### 根拠

- `AddBalance` は処理先頭で `balance.Validate()` を呼び、負の開始 `Balance` を受け取った場合は加算や overflow 判定へ進まず、元の `balance` と `ErrBalanceMustBeNonNegative` を返す。
- `SubtractBalance` も処理先頭で `balance.Validate()` を呼び、負の開始 `Balance` を受け取った場合は減算や残高不足判定へ進まず、元の `balance` と `ErrBalanceMustBeNonNegative` を返す。
- `amount.Validate()` は開始 `Balance` の検証後に実行されるため、破損した既存残高を取引金額不備より優先して検出する順序になっている。
- 追加 unit test は、`AddBalance` / `SubtractBalance` が負の開始 `Balance` を拒否して元の `balance` を返すこと、および負の開始 `Balance` と invalid `Amount` が同時に渡った場合に `ErrBalanceMustBeNonNegative` が優先されることを確認している。
- README は、残高加算・減算 helper が開始残高と取引金額を再検証する現行実装範囲を説明している。

### 影響

- 将来の repository / mapper / test fixture が constructor を経由せずに `Balance` を復元した場合でも、少なくとも domain の残高演算 helper は負の開始残高を検出して fail closed できる。
- 負の開始残高に対する出金が `ErrInsufficientBalance` と誤分類されることや、負の開始残高に対する入金で見かけ上の残高が補正されたように見えることを防ぎやすくなる。
- 元帳・取引履歴実装前の段階として、破損した既存残高を「新しい取引の成功/失敗」と混同しないための domain 境界が強化された。

### 推奨修正

- 今 cycle の実装差分に対する修正は不要。
- 次に repository / service / DB read mapper を追加するときは、DB から復元した `Balance` を `Balance.Validate()` で検証し、破損値を通常の残高不足や金額不備とは別分類で停止する test を追加する。
- 破損残高を検出した場合は、残高変更取引を作らず、調査用の失敗監査ログまたは運用アラートへ接続する方針を docs / test で明確化する。

### 次サイクル planner への入力

- DB / repository 着手前に、`accounts.balance_amount >= 0`、`transactions.amount > 0`、`transactions.balance_after >= 0` などの PostgreSQL `CHECK` 制約と Go domain validation の対応関係を accepted scope 候補にする。
- `Balance` 復元時の validation 漏れを防ぐため、DB mapper または repository 境界の test scope を検討する。

## Finding 2: 残高演算 helper の error 優先順位は元帳調査性に有利だが、将来の監査分類へ接続が必要

### 根拠

- 今回の差分では、開始 `Balance` の validation が `Amount` validation より前に実行される。
- test でも、負の開始 `Balance` と `Amount{}` が同時に渡った場合に `ErrBalanceMustBeNonNegative` が返ることを `AddBalance` / `SubtractBalance` の両方で固定している。
- `implementer.md` は、破損した既存残高をリクエスト金額不備より優先的に検出する作業仮定を明記している。

### 影響

- 既存残高が壊れている場合、利用者入力の金額エラーとして処理を続けたり再試行を促したりするのではなく、内部データ不整合として扱いやすくなる。
- これは元帳調査では望ましい一方、将来 HTTP / service 層でそのまま domain error を利用者向け response に出すと、内部残高破損の存在を過度に露出する可能性がある。
- 監査ログや失敗分類が未実装のため、現時点では `ErrBalanceMustBeNonNegative` をどの監査 `failure_reason` に対応させるかは未定義のまま残る。

### 推奨修正

- 今 cycle の修正は不要。
- API / service 実装時は、`ErrBalanceMustBeNonNegative` を利用者向けには汎用的な内部処理失敗として返し、監査・運用ログでは `corrupt_balance_detected` のような調査用分類へ正規化する設計を検討する。
- `ErrAmountMustBePositive`、`ErrInsufficientBalance`、`ErrBalanceOverflow`、`ErrBalanceMustBeNonNegative` を、利用者向け error、監査 `failure_reason`、運用アラート要否に分ける mapping 表を docs に追加する。

### 次サイクル planner への入力

- 金額・残高 domain error mapping と監査 `failure_reason` 分類を docs-only scope として採択候補にする。
- 特に `ErrBalanceMustBeNonNegative` は通常の利用者操作エラーではなく、DB 破損・mapper bug・migration bug の可能性があるため、通常の残高不足とは別扱いにする。

## Finding 3: 現行差分は scope 内だが、元帳・残高整合性の本丸である DB transaction / 取引履歴 / 行ロックは未着手

### 根拠

- 今回の実装差分は `internal/domain` の helper validation と unit test、README、cycle artifact に限定されている。
- `implementer.md` は、HTTP route、業務 API、PostgreSQL 接続、migration、DB schema、repository、transaction manager、監査ログ永続化、冪等性キー、行ロック、取消 / reversal を実装していないと明記している。
- `go test ./...` は成功したが、現在の test は domain helper、server、router の範囲であり、DB transaction による残高更新と取引履歴の同時 commit / rollback はまだ検証対象ではない。

### 影響

- 今回の helper 強化だけでは、入金・出金・振込における「残高変更と取引履歴が必ず同時に残る」「振込の片側だけ成功しない」「同一口座の並行出金で二重に残高を消費しない」という金融事故リスクはまだ解消されない。
- 業務 API を先に公開すると、DB transaction、行ロック、監査ログ、冪等性の未整備により、残高不整合や取引履歴欠落のある endpoint になり得る。
- 現時点では業務 API が未実装のため直接の金融事故は発生しないが、次に進む順序を誤ると設計負債が大きくなる。

### 推奨修正

- 次 cycle 以降は、業務 API 実装より前に DB schema / transaction 方針を docs または小さな migration scope として固める。
- 入出金・振込を実装する前に、少なくとも次を設計・test 方針へ落とす。
  - `accounts` の残高非負制約。
  - `transactions` の追記専用方針、正の取引金額、`balance_after` 非負制約。
  - 残高変更と取引履歴を同一 DB transaction に含める方針。
  - 失敗監査ログを業務 transaction とは独立して残す方針。
  - 同一口座 / 振込元・振込先口座の行ロック順序。
  - 残高不足時に残高・取引履歴が変わらない rollback test。

### 次サイクル planner への入力

- 次の優先候補は「PostgreSQL schema / transaction 前の元帳整合性設計 docs」または「最小 migration と DB 制約 test」にする。
- 具体的には、`accounts.balance_amount`、`transactions.amount`、`transactions.balance_after`、`transfer_requests.amount` の制約、取引履歴の追記専用方針、行ロック順序、deadlock 時の fail / retry 方針を扱う。

## Finding 4: 金額上限・残高上限は未確定であり、API 接続前に業務上の事故シナリオを潰す必要がある

### 根拠

- `Amount.Validate()` は 0 以下を拒否し、`Balance.Validate()` は負の残高を拒否するが、1 回あたり取引金額上限、口座残高上限、日次上限は実装されていない。
- `AddBalance` は `math.MaxInt64` overflow を検出するが、業務上不自然な巨大入金・巨大残高を拒否する仕様ではない。
- planner / implementer は、取引金額上限、残高上限、日次上限を今回の scope 外として明記している。

### 影響

- 現時点では外部入力 API がないため、直ちに利用者が巨大金額を投入する経路はない。
- ただし API / admin 代行操作 / migration tool を追加する前に上限が未定義だと、overflow しないが業務上は異常な金額の入金・出金・振込を許す設計になり得る。
- 上限超過が監査分類に接続されないと、誤操作、不正操作、test data 混入の調査が難しくなる。

### 推奨修正

- API 実装前に、MVP の暫定値として 1 回あたり入金・出金・振込上限、口座残高上限、必要なら日次上限を docs に作業仮定として記録する。
- domain helper に入れる不変条件と、service / policy 層で扱う業務制限を分ける。例えば「0 以下禁止」は domain、「1 回あたり上限・日次上限」は service policy とする案を検討する。
- 上限超過時の監査 `failure_reason` と利用者向け error を、残高不足や内部破損とは別分類にする。

### 次サイクル planner への入力

- 「金額上限・残高上限・日次上限・監査 failure_reason の docs 化」を DB / API 実装前の候補として扱う。
- 人間レビューで調整しやすいよう、上限値は本番金融機関相当の断定ではなく、学習用 MVP の暫定値として提示する。

## 実行した確認

- ✅ `git status --short`
- ✅ `go test ./...`
- ✅ `rg -n "float32|float64" internal/domain`（一致なし。浮動小数点利用なし）
- ✅ `git diff --find-renames HEAD^ HEAD -- README.md internal/domain/money.go internal/domain/money_test.go docs/ai/cycles/2026-06-30-001/implementer.md`
