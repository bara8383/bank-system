# security-reviewer: 2026-06-30-001

## 入力と確認範囲

- 作業開始時に `git status --short`、`README.md`、`AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、docs 配下の設計文書、同一 cycle の `planner.md` / `implementer.md` を確認した。
- agent 間の直接同期は行わず、`docs/ai/cycles/2026-06-30-001/planner.md`、`docs/ai/cycles/2026-06-30-001/implementer.md`、および `HEAD~1..HEAD` の実装差分を入力にした。
- レビュー対象差分は `Amount.Validate()` / `Balance.Validate()` の追加、既存 constructor / `AddBalance` / `SubtractBalance` の validation 共有、unit test 追加、README 更新。
- 観点は認証、認可、入力検証、ログ、秘密情報、SQL injection、権限境界、監査証跡。今回差分は domain helper のみで HTTP / DB / 認証 / ログ永続化を追加していないため、入力検証と将来境界利用リスクを優先した。

## Finding 1: Blocking 指摘なし

- 重大度: なし
- 種別: 差分レビュー結果

### 根拠

- 今回差分は `internal/domain/money.go` の値 object validation と unit test、README / cycle artifact 更新に限定されている。
- HTTP route、handler、request / response schema、PostgreSQL 接続、SQL、repository、認証、認可、Cookie / CSRF、監査ログ永続化、秘密情報処理は追加されていない。
- `Amount.Validate()` は 0 以下を `ErrAmountMustBePositive` で拒否し、constructor bypass による 0 円または負の取引金額混入を境界で検出できる。
- `Balance.Validate()` は負の残高を `ErrBalanceMustBeNonNegative` で拒否し、DB / repository / service 境界で残高不変条件を再確認できる。
- unit test で `Amount{}` の拒否、正の `Amount` の許可、0 / 正の `Balance` の許可、負の `Balance` の拒否が確認されている。

### 影響

- 新規の認証 bypass、認可 bypass、SQL injection、秘密情報漏えい、ログへの機微情報混入、監査ログ欠落を直接発生させる差分は確認していない。
- `Amount{}` を invalid として扱えるため、将来の入金・出金・振込 API で 0 円取引や負の取引金額を拒否するための最小 domain 防御が強化された。

### 推奨修正

- この cycle の差分に対する blocking 修正は不要。
- 次に業務 API / repository / DB insert を追加する際は、外部入力を `NewAmount` / `NewBalance` で生成し、永続化直前または DB から復元した値に対して `Validate()` を呼ぶ規約を service / repository 実装と test に含める。

### 次サイクル planner への入力

- 業務 API 実装前に、request validation、domain error から安全な HTTP error への mapping、失敗監査ログの `failure_reason` 分類を docs または小さな skeleton として採択候補にする。
- DB schema 実装前に、`transactions.amount > 0`、`accounts.balance_amount >= 0`、`transactions.balance_after >= 0`、`transfer_requests.amount > 0` の PostgreSQL `CHECK` 制約を domain validation と対応付ける scope を検討する。

## Finding 2: `AddBalance` / `SubtractBalance` は操作対象の `Balance` 自体を再検証していない

- 重大度: Medium（将来の repository / service 接続時）
- 種別: 入力検証 / 権限境界前の domain invariant 防御

### 根拠

- 今回追加された `Balance.Validate()` は負の残高を拒否できるが、`AddBalance` と `SubtractBalance` は `amount.Validate()` のみを呼び、引数 `balance` の `Validate()` は呼んでいない。
- `Balance` の field は package 外から直接設定できないが、同一 package 内の test や将来追加される mapper / scanner / repository helper では `Balance{value: ...}` を構築できる。
- 将来 DB から復元した残高や migration / test fixture / internal mapper が constructor を経由しない場合、負の `Balance` が `AddBalance` / `SubtractBalance` に渡されても、現状の helper 自体は負の開始残高を明示的には拒否しない。

### 影響

- 現時点では repository / DB / 業務 API が未実装のため、外部攻撃者がこの経路を直接悪用する surface はない。
- ただし将来、DB 復元値や internal mapper の boundary validation が漏れると、負の開始残高を前提にした加算・減算が発生し、金融整合性と監査調査性を損なう可能性がある。
- 認可済み操作であっても、不正な既存残高を検出せず処理を続けると、障害復旧や不正検知で「どの境界で破損値を拒否すべきだったか」が曖昧になる。

### 推奨修正

- 次 cycle 以降で、`AddBalance` / `SubtractBalance` の先頭で `balance.Validate()` も呼び、負の開始残高なら元 balance と `ErrBalanceMustBeNonNegative` を返すかを検討する。
- 互換性や責務分担の理由で helper 内 validation を増やさない場合は、repository から service へ渡す直前、または service の use case entrypoint で `Balance.Validate()` を必須にする規約と unit test を追加する。
- DB 実装時は application validation だけに依存せず、`accounts.balance_amount >= 0` と `transactions.balance_after >= 0` の `CHECK` 制約を最終防衛線にする。

### 次サイクル planner への入力

- 「残高変更 helper が開始残高も検証するか」「service / repository 境界でのみ検証するか」を小さな design decision として `docs/` に記録する scope を候補化する。
- DB / repository 着手前に、DB から復元した `Amount` / `Balance` の validation policy、破損データ検出時の監査ログ分類、業務処理の停止方法を整理する。

## Finding 3: 金額上限・残高上限・監査 failure_reason は未実装のまま

- 重大度: Medium（業務 API 公開前の設計課題）
- 種別: 入力検証 / 監査証跡

### 根拠

- `Amount.Validate()` は 0 以下を拒否するが、1 回あたり取引金額上限、口座残高上限、日次上限などの業務上限は accepted scope 外として実装されていない。
- `Balance.Validate()` は負の残高を拒否するが、異常に大きな残高や `math.MaxInt64` 近傍の値を業務上拒否する rule はない。
- `implementer.md` でも、取引金額上限、残高上限、日次上限、監査ログ分類は未実装として明記されている。

### 影響

- 現時点では業務 API がないため直ちに外部入力リスクにはならない。
- 将来 API を追加する際に上限と監査分類が未定義のままだと、認証済み利用者または admin 代行操作が極端に大きい金額を投入できる設計になり、overflow 以外の業務不正・誤操作・監査不能リスクが残る。
- failure_reason が正規化されないと、監査ログの検索・集計・不正検知・利用者向け error との分離が難しくなる。

### 推奨修正

- 業務 API 実装前に、MVP の暫定値として 1 回あたり取引金額上限、口座残高上限、必要なら日次上限を docs に記録し、domain / service のどちらで検証するかを分ける。
- 上限違反、0 / 負の金額、残高不足、認証失敗、認可失敗、冪等性キー衝突、口座状態不正などを安全な `failure_reason` enum として設計する。
- 利用者向け error message には内部状態や存在確認結果を過剰に出さず、監査ログには調査に必要な分類情報だけを残す方針を維持する。

### 次サイクル planner への入力

- 次 cycle の候補として「金額上限・残高上限・domain error mapping・監査 failure_reason 分類の docs 化」を優先度高で扱う。
- その後の API / DB 実装 scope では、認証済みユーザー本人または admin 代行の認可、CSRF、冪等性キー、監査ログを同時に接続する前提にする。
