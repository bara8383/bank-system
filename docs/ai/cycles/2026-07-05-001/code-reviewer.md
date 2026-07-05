# code-reviewer: 2026-07-05-001

## Finding

### 1. Blocking finding はありません

今回の実装差分に対して、Go / PostgreSQL 境界 / 設計 / 保守性 / テスト観点で、次工程へ進む前に修正必須と判断する finding はありません。

## 根拠

- レビュー対象差分は `README.md`、`docs/ai/cycles/2026-07-05-001/implementer.md`、`internal/domain/transaction.go`、`internal/domain/transaction_test.go` に限定されており、planner の accepted scope に含まれる領域と一致している。
- `TransactionType` は `deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` の 4 種のみを valid としており、空文字・未知値・`reversal` を `ErrInvalidTransactionType` で拒否している。これは human notes と planner scope の「reversal は MVP 初期 code では valid type に含めない」という前提に合っている。
- `ApplyTransaction` は accepted scope どおり `balance.Validate()`、`amount.Validate()`、`transactionType.Validate()` の順で検証し、検証失敗時に元の `Balance` を返している。
- `deposit` / `transfer_credit` は `AddBalance` に委譲し、`withdrawal` / `transfer_debit` は `SubtractBalance` に委譲しているため、既存の残高非負、正の金額、残高不足、overflow の sentinel error を再利用できている。
- 新規 helper は pure domain code に留まっており、HTTP route、DB 接続、SQL、repository、transaction manager、認証、認可、監査ログ、冪等性キーを追加していない。今回の cycle で未確定の境界を暗黙実装していない点は保守性上妥当である。
- README は取引種別 helper と `ApplyTransaction` の追加を現在の実装範囲へ反映しつつ、取引履歴永続化、transaction row 作成、`balance_after` の DB 保存、業務 API、PostgreSQL、監査ログ、冪等性キーが未実装であることを維持している。
- `gofmt -l internal/domain/transaction.go internal/domain/transaction_test.go` は出力なしで、Go formatting 上の問題は見つからなかった。
- `go test ./...` は成功した。
- `git diff --check HEAD^..HEAD` は出力なしで、差分上の whitespace error は見つからなかった。

## 影響

- `ApplyTransaction` により、将来の入金・出金・振込 service が transaction type ごとの残高増減方向を重複実装せずに済むため、方向ミスを減らせる。
- `transfer_debit` と `transfer_credit` の方向が unit test で固定されたため、振込実装時に片側だけ逆方向へ反映するリスクを早期検出しやすくなった。
- `reversal` を invalid として扱うことで、取消・組戻し・訂正の未確定仕様が MVP 初期実装へ混入するリスクを抑えている。
- ただし、この helper は transaction row 作成、PostgreSQL transaction、行ロック、監査ログ、冪等性、認証認可をまだ保証しない。後続 cycle で service / repository / DB schema へ進む際に、この helper の成功をもって金融整合性全体が満たされたと誤解しないことが重要である。

## 推奨修正

- 今回差分に対する修正必須事項はない。
- 任意の保守性改善として、将来 `ApplyTransaction` の呼び出し箇所が増える前に、domain error から API response / audit `failure_reason` への mapping 表を docs 化するとよい。現時点では `ErrInvalidTransactionType` が追加されたため、既存の `ErrAmountMustBePositive`、`ErrInsufficientBalance`、`ErrInvalidAccountStatus`、`ErrAccountNotActive` と合わせて外部応答・監査分類を決める必要がある。
- DB 実装へ進む cycle では、`transactions.transaction_type` の CHECK constraint、`transactions.amount > 0`、`transactions.balance_after >= 0`、`accounts.balance_amount >= 0`、および `accounts.status` の CHECK constraint を Go domain helper と対応付けて設計することを推奨する。
- 入出金・振込 service を実装する前に、`EnsureAccountCanTransact` と `ApplyTransaction` をどの順序で呼び、失敗時監査ログをどの transaction 境界で残すかを明示するとよい。

## 次サイクル planner への入力

- 次 cycle の有力候補は、domain error / API response / audit `failure_reason` mapping の docs 化である。今回 `ErrInvalidTransactionType` が増えたため、外部レスポンスへ過剰な内部情報を出さず、監査ログでは調査可能な分類を残す表を作ると後続 API 実装が進めやすい。
- DB schema / migration に進む場合は、Go domain の `TransactionType` 4 種と `reversal` 除外をそのまま PostgreSQL の CHECK constraint 案へ反映する。ただし `reversal` は docs/data-model では将来候補として残っているため、MVP 初期 schema で許可するかどうかを planner で明示すること。
- 入出金 service に進む場合は、残高更新、取引履歴作成、成功監査ログを同じ PostgreSQL transaction に含める設計、失敗監査ログを独立 transaction で残す設計、行ロックの取得位置を同じ scope で扱うこと。
- 振込 service に進む場合は、`transfer_debit` / `transfer_credit` の 2 行の取引履歴、2 口座の lock 順序、振込依頼状態、冪等性キー重複時拒否、片側だけ成功しない rollback test を accepted scope に含めること。
- 今回の review では blocking finding がないため、次 planner は reviewer 修正対応よりも、API / DB 実装前の error mapping、DB constraint、transaction boundary、lock 順序の具体化を優先してよい。
