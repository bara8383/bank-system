# banking-reviewer: 2026-07-05-001

## Finding

### Finding 1: Blocking なし — 取引種別ごとの残高増減方向は設計と一致している

- `deposit` / `transfer_credit` を残高増加、`withdrawal` / `transfer_debit` を残高減少として扱っており、元帳・取引履歴で最も事故につながりやすい「振込の片側方向ミス」は今回差分では確認しなかった。
- `reversal` は valid type に含まれておらず、取消・組戻し・訂正の未確定仕様を MVP 初期の通常取引 helper に混入させていない。

### Finding 2: Blocking なし — 異常系で元の残高を返す方針は残高不整合を増やしていない

- `ApplyTransaction` は `balance`、`amount`、`transactionType` の順に validation し、validation 失敗時は元の `balance` を返している。
- 出金系の残高不足と入金系の overflow でも元の `balance` と sentinel error を返すテストがあり、呼び出し側が error を無視しない限り、helper 単体では残高を中途半端に進めない。

### Finding 3: 注意点 — 今回 helper は `balance_after` の計算候補であり、元帳完成ではない

- 今回の実装は pure domain helper に限定され、取引履歴 row 作成、`balance_after` の DB 保存、監査ログ、DB transaction、行ロック、冪等性、口座状態 gate、認証認可は実装していない。
- したがって、後続の入金・出金・振込 service で `ApplyTransaction` だけを呼んで口座残高を更新すると、「残高は変わったが取引履歴・監査ログがない」「振込元だけ成功した」「停止中口座へ入金した」などの事故シナリオが残る。

## 根拠

- `docs/data-model.md` は `transaction_type` ごとの方向として、`deposit` と `transfer_credit` は残高増加、`withdrawal` と `transfer_debit` は残高減少、`reversal` は MVP 初期では方向と利用条件を未確定と定義している。
- `docs/design-principles.md` は、成功した残高変更では口座残高、取引履歴、成功監査ログを同じ PostgreSQL transaction に含め、残高変更だけ・取引履歴だけの状態を禁止している。
- planner accepted scope は、今回の対象を取引種別 validation と残高反映 helper に限定し、`reversal`、DB、取引履歴永続化、監査ログ、冪等性、業務 API を実装しないことを明示している。
- `internal/domain/transaction.go` では MVP 4 種だけを valid とし、`ApplyTransaction` で `deposit` / `transfer_credit` を `AddBalance`、`withdrawal` / `transfer_debit` を `SubtractBalance` に委譲している。
- `internal/domain/transaction_test.go` では、MVP 4 種の validation、空文字・未知値・`reversal` の拒否、増加方向、減少方向、残高不足、invalid starting balance、invalid amount、invalid transaction type、overflow、validation 順序が確認されている。
- README は、取引種別 helper を現在の実装範囲に追加する一方、取引履歴の永続化、transaction row 作成、`balance_after` の DB 保存、業務 API は未実装のままと説明している。

## 影響

- 今回差分により、後続 service が取引種別ごとの残高方向を個別実装する必要が減り、入金・出金・振込入金・振込出金の方向ミスを unit test で検出しやすくなった。
- `reversal` を通常取引 helper から除外したことで、取消・組戻し・訂正の未確定仕様が残高計算へ暗黙に入り込むリスクは抑えられている。
- 一方で、この helper は取引履歴を作らないため、金融システムとしての「残高変更の根拠を必ず残す」保証はまだない。DB 永続化時には、残高更新、取引履歴、監査ログ、振込依頼状態を同じ整合性境界で扱う設計が必須である。
- `ApplyTransaction` は口座ステータスや認可を知らないため、後続 service では `EnsureAccountCanTransact`、owner / role authorization、冪等性キー検証、監査ログ分類と組み合わせる必要がある。

## 推奨修正

- 今回差分への即時修正は不要。
- 次に入金・出金・振込 service を実装する前に、`ApplyTransaction` を直接 DB update の便利関数として使うのではなく、次の順序を service 境界の設計として固定することを推奨する。
  1. 認証・認可を確認する。
  2. 口座存在、口座ステータス、金額、取引種別を検証する。
  3. PostgreSQL transaction 内で対象口座を行ロックする。
  4. `ApplyTransaction` で取引後残高を計算する。
  5. `accounts.balance_amount` 更新、`transactions.balance_after` を含む取引履歴作成、成功監査ログ作成を同じ transaction で commit する。
  6. 業務拒否や transaction 途中失敗では業務データを更新せず、失敗監査ログを独立 transaction で残す。
- `ErrInvalidTransactionType`、`ErrAmountMustBePositive`、`ErrInsufficientBalance`、`ErrBalanceMustBeNonNegative`、`ErrBalanceOverflow`、`ErrInvalidAccountStatus`、`ErrAccountNotActive` を API response と audit `failure_reason` にどう写像するかを、業務 API 追加前に docs 化することを推奨する。
- 振込実装前には、2 口座の lock 順序、同一口座間振込の扱い、冪等性キーの一意範囲、重複時拒否の監査分類を設計してから実装することを推奨する。

## 次サイクル planner への入力

- 優先候補 1: domain error / API response / audit `failure_reason` mapping を docs 化する。今回追加された `ErrInvalidTransactionType` も mapping 対象に含める。
- 優先候補 2: 入金・出金・振込 service の DB transaction 境界を設計する。成功時は残高更新、取引履歴、成功監査ログを同一 transaction に含め、失敗監査ログは独立 transaction とする。
- 優先候補 3: PostgreSQL schema / constraint の最小実装に進む場合は、`accounts.balance_amount >= 0`、`transactions.amount > 0`、`transactions.balance_after >= 0`、`transaction_type` CHECK、口座 status CHECK、冪等性キー UNIQUE の範囲を同時に検討する。
- 優先候補 4: `ApplyTransaction` と `EnsureAccountCanTransact` の併用を service 設計に明記し、停止中・解約済み口座で残高変更 helper が単独利用されないようにする。
- 優先候補 5: `reversal` は引き続き MVP 通常取引から除外し、通常の入金・出金・振込、取引履歴永続化、監査ログ、冪等性が固まるまで実装しない。
