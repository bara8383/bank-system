# security-reviewer: 2026-06-30-001

## 入力と確認範囲

- 作業開始時に `git status --short` を確認し、未コミット変更がない状態から開始した。
- `README.md`、`AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、`docs/START_HERE.md`、`docs/*.md`、`docs/ai/output/human/*.md` を確認した。
- agent 間の直接同期は行わず、`docs/ai/cycles/2026-06-30-001/` 配下の同一 cycle 成果物と repo 差分のみを入力にした。
- 実装差分レビューを優先し、`HEAD^..HEAD` の `README.md`、`internal/domain/money.go`、`internal/domain/money_test.go`、`docs/ai/cycles/2026-06-30-001/implementer.md` を確認した。
- 観点は認証、認可、入力検証、ログ、秘密情報、SQL injection、権限境界、監査証跡。今回差分は domain helper と test / README / implementer artifact に限定され、HTTP / DB / 認証 / 監査ログ永続化を追加していないため、入力検証と将来境界利用リスクを重点確認した。

## Finding 1: Blocking 指摘なし。開始 `Balance` の fail closed が helper に接続された

- 重大度: なし
- 種別: 入力検証 / domain invariant 防御

### 根拠

- `AddBalance` は演算前に `balance.Validate()` を呼び、負の開始残高なら元 `balance` と `ErrBalanceMustBeNonNegative` を返す実装になっている。
- `SubtractBalance` も同様に、取引金額検証や残高不足判定より前に `balance.Validate()` を呼ぶ実装になっている。
- `Amount.Validate()` は 0 以下を `ErrAmountMustBePositive` で拒否し、`Balance.Validate()` は負の残高を `ErrBalanceMustBeNonNegative` で拒否する既存の domain invariant を維持している。
- unit test は、`AddBalance` / `SubtractBalance` が負の開始 `Balance` を拒否して元 `balance` を返すこと、および負の開始 `Balance` と invalid `Amount` が同時に渡った場合に `ErrBalanceMustBeNonNegative` を優先することを確認している。
- README は、残高加算・減算 helper が開始残高と取引金額を再検証すること、および外部ライブラリ、DB 接続、認証、業務 API が未導入であることを明示している。

### 影響

- 直近の security-reviewer / code-reviewer / banking-reviewer が指摘していた「開始残高の再検証責務が helper か境界か曖昧」というリスクは、今回差分で helper 側 fail closed に寄せられた。
- 将来の repository mapper、DB scan、test fixture、migration helper などが constructor を経由せず `Balance` を生成した場合でも、残高演算 helper の入口で破損した負残高を検出しやすくなった。
- 現時点の差分は HTTP route、SQL、認証、認可、Cookie / CSRF、監査ログ永続化、秘密情報処理を追加していないため、新規の認証 bypass、認可 bypass、SQL injection、秘密情報漏えい、監査ログへの機微情報混入を直接発生させる変更は確認していない。

### 推奨修正

- この cycle の実装差分に対する blocking 修正は不要。
- 後続で service / repository / DB insert を追加する際は、今回の helper 側 validation に加えて、外部入力からの `NewAmount` / `NewBalance` 生成、DB から復元した値の `Validate()`、永続化直前の DB 制約を test で接続する。

### 次サイクル planner への入力

- DB schema / repository 着手前に、Go 側 `Amount.Validate()` / `Balance.Validate()` / `AddBalance` / `SubtractBalance` と PostgreSQL `CHECK` 制約の対応関係を docs 化する scope を候補化する。
- 業務 API 実装前に、request validation、domain error から安全な HTTP error への mapping、失敗監査ログの `failure_reason` 分類を docs または小さな skeleton として採択候補にする。

## Finding 2: 業務上限と監査 failure_reason は未実装で、API 接続前に設計固定が必要

- 重大度: Medium（業務 API 公開前の設計課題）
- 種別: 入力検証 / 監査証跡

### 根拠

- `Amount.Validate()` は 0 以下を拒否するが、1 回あたり取引金額上限、口座残高上限、日次上限などの業務上限は今回の accepted scope 外として実装されていない。
- `Balance.Validate()` と `AddBalance` の overflow 検出により負残高や `int64` overflow は拒否できるが、業務上不自然な高額取引や高額残高を拒否する rule はまだない。
- `implementer.md` でも、取引金額上限、残高上限、日次上限、監査ログ分類は実装しなかったこととして明記されている。
- `docs/security-notes.md` は監査ログの `failure_reason` を安全な分類または短い理由にする方針を持つが、現時点では enum / 分類表 / error mapping までは固定されていない。

### 影響

- 現時点では業務 API がないため、外部利用者が直接大きな金額を投入する攻撃 surface はない。
- 将来 API を追加する際に上限と監査分類が未定義のままだと、認証済み顧客または admin 代行操作が極端に大きい金額を投入できる設計になり、overflow 以外の誤操作・不正操作・監査集計不能リスクが残る。
- 失敗理由が文字列の自由入力や内部 error そのままになると、利用者向け error と運用調査向け分類が分離できず、存在確認結果や内部状態を過剰に露出するリスクがある。

### 推奨修正

- 業務 API 実装前に、MVP の暫定値として 1 回あたり取引金額上限、口座残高上限、必要なら日次上限を docs に記録する。
- 上限違反、0 / 負の金額、残高不足、認証失敗、認可失敗、CSRF 不正、冪等性キー衝突、口座状態不正、対象不存在などを安全な `failure_reason` enum として設計する。
- 利用者向け error message は安全で曖昧な表現にし、監査ログには検索・集計に必要な分類情報のみを残す方針を維持する。

### 次サイクル planner への入力

- 次 cycle の候補として「金額上限・残高上限・domain error mapping・監査 `failure_reason` 分類の docs 化」を優先度高で扱う。
- API / DB 実装 scope では、認証済みユーザー本人または admin 代行の認可、Cookie session + CSRF token、冪等性キー、監査ログを同時に接続する前提にする。

## Finding 3: DB 制約・行ロック・監査ログ永続化は未実装のままなので、domain helper だけを最終防衛線にしない

- 重大度: Medium（DB / repository 着手時）
- 種別: DB 境界 / 監査証跡 / 権限境界

### 根拠

- 今回差分は `internal/domain` の helper と unit test に限定され、PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager は追加していない。
- `docs/design-principles.md` と `docs/test-strategy.md` は、残高変更、取引履歴、成功監査ログを同じ PostgreSQL transaction に含め、失敗監査ログを独立 transaction で残す方針を示している。
- `docs/ai/output/human/001-human-review.md` では PostgreSQL 行ロックでの悲観ロック、冪等性キーに操作種別・送信元口座・ログインユーザー・request body hash を含めること、Cookie + CSRF token が人間判断として示されている。
- README は、PostgreSQL 接続、DB schema、migration、transaction 処理、認証、認可、監査ログ、冪等性キー処理がまだ未実装であることを明示している。

### 影響

- domain helper の fail closed は有効な防御層だが、DB 直書き、migration 不備、repository bug、並行更新、transaction 途中失敗までは防げない。
- DB 制約、行ロック、transaction 境界、成功 / 失敗監査ログ保存境界が未接続のまま業務 API を先行すると、二重実行、片側更新、履歴欠落、監査ログ欠落、権限不足操作の調査不能につながる。
- 認証・認可・CSRF が未接続のまま残高変更 endpoint を作ると、Cookie session 前提の CSRF や本人以外口座操作の risk が API surface として顕在化する。

### 推奨修正

- DB schema では application validation に加えて、`transactions.amount > 0`、`accounts.balance_amount >= 0`、`transactions.balance_after >= 0`、`transfer_requests.amount > 0` などの `CHECK` 制約を最終防衛線にする。
- 入金・出金・振込の service / repository 実装では、残高更新、取引履歴、成功監査ログを同一 DB transaction に含め、失敗監査ログは rollback 後に独立 transaction で残す test を先に用意する。
- Cookie session を採用する前提では、残高変更・振込・顧客登録・口座作成などの state-changing API に CSRF token 検証を必須化する。
- 振込実装前に、冪等性キーの構成要素、同一キー衝突時の MVP 挙動、request body hash の算出対象、監査 failure_reason を固定する。

### 次サイクル planner への入力

- 優先候補 1: PostgreSQL schema / constraint / transaction 境界 / 行ロック順序 / rollback test 方針を docs 化する。
- 優先候補 2: 認証・認可・CSRF・安全な error mapping・監査 `failure_reason` の最小設計を docs 化する。
- 優先候補 3: 実装に進む場合は、業務 API より先に DB 制約と repository / transaction skeleton を小さく作り、domain validation と DB validation の二重防衛を test で確認する。

## 実行した確認

- ✅ `git status --short`
- ✅ `go test ./...`
- ✅ `rg -n "float32|float64" internal/domain`（一致なし。浮動小数点利用なし）
- ✅ `git diff HEAD^ HEAD -- README.md internal/domain/money.go internal/domain/money_test.go docs/ai/cycles/2026-06-30-001/implementer.md`
