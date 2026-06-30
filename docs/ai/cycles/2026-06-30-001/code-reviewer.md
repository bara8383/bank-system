# code-reviewer: 2026-06-30-001

## 確認した入力

- 作業開始時に `git status --short` を確認し、未コミット変更がないことを確認した。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、`README.md`、`docs/START_HERE.md`、`docs/*.md`、`docs/ai/output/human/*.md` を確認した。
- agent 間の直接同期は行わず、同一 cycle の `docs/ai/cycles/2026-06-30-001/planner.md`、`docs/ai/cycles/2026-06-30-001/implementer.md`、`docs/ai/cycles/2026-06-30-001/banking-reviewer.md`、`docs/ai/cycles/2026-06-30-001/security-reviewer.md`、および repo 差分を入力にした。
- 実装差分レビューを優先し、`HEAD~1..HEAD` の `6444ea3 Validate starting balance in domain helpers` を対象に確認した。対象ファイルは `README.md`、`internal/domain/money.go`、`internal/domain/money_test.go`、`docs/ai/cycles/2026-06-30-001/implementer.md`。
- reviewer として、ソースコード・README・通常 docs は変更せず、本ファイルのみを更新した。

## Finding 1: blocking なし。残高演算 helper は accepted scope どおり開始 `Balance` を先に再検証している

### 根拠

- `AddBalance(balance, amount)` は `amount.Validate()` より前に `balance.Validate()` を呼び、負の開始残高なら元の `balance` と `ErrBalanceMustBeNonNegative` を返す。
- `SubtractBalance(balance, amount)` も同じ順序で `balance.Validate()` を先に呼び、負の開始残高では残高不足判定や減算へ進まない。
- 既存の invalid amount、overflow、残高不足の sentinel error と「エラー時に元 balance を返す」挙動は維持されている。
- `TestAddBalanceRejectsInvalidStartingBalanceAndReturnsOriginalBalance`、`TestSubtractBalanceRejectsInvalidStartingBalanceAndReturnsOriginalBalance` により、負の開始 `Balance` を拒否し元 balance を返すことを確認している。
- `TestAddBalancePrioritizesInvalidStartingBalanceOverInvalidAmount`、`TestSubtractBalancePrioritizesInvalidStartingBalanceOverInvalidAmount` により、負の開始 `Balance` と invalid `Amount` が同時に渡った場合に `ErrBalanceMustBeNonNegative` を優先する仕様が test で固定されている。
- README は実装済み範囲を「開始残高と取引金額を再検証する残高加算・減算」と説明し、今回差分と矛盾していない。

### 影響

- DB / repository / mapper / test fixture などから constructor を経由しない破損 `Balance` が渡った場合でも、domain helper が演算継続を止める防御層になっている。
- 破損した既存残高を request amount の不備や残高不足として扱わず、`ErrBalanceMustBeNonNegative` として早期検出できるため、将来の service / repository 層で障害調査しやすい。
- 今回差分は domain helper と test / README 更新に限定され、HTTP / DB / 認証 / 監査 / 冪等性など未確定の領域を先取りしていない。

### 推奨修正

- 今 cycle の差分に対する修正は不要。
- 次に service / repository / DB read mapper を追加する際は、DB から復元した `Balance` / `Amount` の `Validate()` と、演算 helper 側の fail closed 挙動の責務分担を test 名と設計 docs で明示する。

### 次サイクル planner への入力

- DB schema / repository 着手前に、Go domain validation と PostgreSQL `CHECK` 制約の対応表を docs 化する scope を検討する。
- 例: `accounts.balance_amount >= 0`、`transactions.amount > 0`、`transactions.balance_after >= 0`、`transfer_requests.amount > 0`。

## Finding 2: validation 順序は安全側だが、将来の error mapping / 監査分類で区別が必要になる

### 根拠

- 実装は「開始 `Balance` の破損」を「取引 `Amount` の不備」より優先して検出する順序を採用している。
- この順序は planner の accepted scope と implementer の作業仮定に沿っており、既存 unit test でも明示されている。
- 現在は service / handler / audit repository が存在しないため、`ErrBalanceMustBeNonNegative` を利用者向け error、運用ログ、監査 `failure_reason` にどう対応させるかは未実装。

### 影響

- 将来の業務 API で `ErrBalanceMustBeNonNegative` を単純な入力検証エラーとして利用者へ返すと、内部残高破損の存在を外部に過剰に示す可能性がある。
- 一方、内部破損を generic error として潰しすぎると、監査ログや運用調査で「DB 復元値の破損」なのか「利用者入力の不備」なのか追跡しにくくなる。
- 現時点では API / DB が未実装のため blocking ではないが、service / repository を接続する前に error mapping と監査分類を設計しておくと保守性が高い。

### 推奨修正

- 業務 API または usecase 層を追加する前に、domain error を次のように分類する設計を docs 化する。
  - 利用者入力由来: `ErrAmountMustBePositive`、`ErrInsufficientBalance` など。
  - 内部データ不整合由来: DB から復元した負の `Balance` による `ErrBalanceMustBeNonNegative` など。
  - システム制約由来: `ErrBalanceOverflow`、将来の金額上限・残高上限など。
- `ErrBalanceMustBeNonNegative` が DB 復元後に発生した場合は、利用者向けには安全な汎用エラー、監査 / 運用向けには安全な分類済み `failure_reason` を残す方針を検討する。

### 次サイクル planner への入力

- 「domain error mapping と監査 `failure_reason` 分類」を docs-only scope として採択候補にする。
- 特に `invalid_amount`、`insufficient_balance`、`invalid_account_state`、`authorization_denied`、`invalid_persisted_balance`、`balance_overflow` のような分類名を、API 実装前に仮固定する。

## Finding 3: テストは current domain scope に十分。ただし DB / transaction / lock の検証はまだ存在しない

### 根拠

- `go test ./...` は成功した。
- `rg -n "float32|float64" internal/domain` は一致なしだった。`rg` は一致なしの場合 exit code 1 になるため、これは「domain helper に浮動小数点利用なし」の期待結果として扱った。
- `git diff --name-only HEAD~1..HEAD` と `git ls-files --others --exclude-standard` で、実装差分が accepted scope 内の `README.md`、`docs/ai/cycles/2026-06-30-001/implementer.md`、`internal/domain/money.go`、`internal/domain/money_test.go` に限定されていることを確認した。
- 今回差分は PostgreSQL 接続、migration、SQL、repository、transaction manager、行ロック、業務 API を追加していない。

### 影響

- domain helper の単体品質としては、正系・invalid amount・invalid starting balance・overflow・残高不足が確認されており、今回 scope に対して妥当。
- 一方で、ミニバンキング MVP の中心である「残高更新、取引履歴、成功監査ログを同一 PostgreSQL transaction で整合させる」検証は、まだ code がないため未着手のまま残る。
- 次に API だけを先行追加すると、transaction 境界、監査境界、行ロック、冪等性の未設計部分が handler に漏れ込み、保守性が下がるリスクがある。

### 推奨修正

- 次の code-changing scope は、業務 API より前に DB schema / repository / transaction 境界、またはその docs 設計を優先する。
- 入金または出金の最小 slice を実装する場合は、handler ではなく usecase / repository test を中心にし、残高更新・取引履歴・成功監査ログ・rollback・失敗監査ログの境界を確認する。
- 行ロックを使う方針は human note で示されているため、振込より前に lock order と deadlock 時の扱いを docs 化する。

### 次サイクル planner への入力

- 優先候補 1: PostgreSQL schema / constraint / lock order / transaction 境界を docs 化する。
- 優先候補 2: domain error mapping と監査 `failure_reason` 分類を docs 化する。
- 優先候補 3: DB なしの小実装を続けるなら、domain error を usecase 層で安全に分類する薄い service skeleton と unit test を作る。ただし HTTP route はまだ追加しない。

## 実行した確認

- ✅ `git status --short`
- ✅ `go test ./...`
- ✅ `rg -n "float32|float64" internal/domain`（一致なし。浮動小数点利用なし）
- ✅ `git diff --name-only HEAD~1..HEAD && git ls-files --others --exclude-standard`
- ✅ `git diff --find-renames HEAD~1..HEAD -- README.md internal/domain/money.go internal/domain/money_test.go docs/ai/cycles/2026-06-30-001/implementer.md`
