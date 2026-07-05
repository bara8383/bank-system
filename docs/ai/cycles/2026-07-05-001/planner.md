# planner: 2026-07-05-001

## repo現状

- 作業開始時に `git status --short` を確認した。表示は空で、未コミット変更は確認しなかった。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、`.agents/skills/banking-planning/SKILL.md` を確認した。planner は実装せず、同一 cycle の `planner.md` に accepted scope を作り、agent 間同期は `docs/ai/cycles/<cycle-id>/` 配下の成果物だけで行う。
- README 上の現行実装範囲は、Go 標準ライブラリだけを使った最小 REST API server、`GET /healthz`、JPY `int64` の金額・残高 domain helper、口座ステータス validation helper である。
- 実装済み:
  - `cmd/server`: `BANK_SYSTEM_HTTP_ADDR` による listen address 設定、HTTP server timeout、`internal/httpapi.NewRouter()` 接続。
  - `internal/httpapi`: `GET /healthz`、method 制限、固定 JSON response、運用詳細を返さない test。
  - `internal/domain/money.go`: `Amount` / `Balance`、正の取引金額、0 以上の残高、残高加算、残高減算、残高不足、overflow、constructor bypass 値の `Validate()`。
  - `internal/domain/account.go`: `AccountStatus` の `active` / `suspended` / `closed` validation と、`active` のみ残高変更系操作に進める `EnsureAccountCanTransact`。
- 設計済みだが未実装:
  - 顧客、ログインユーザー、口座、取引履歴、振込依頼、監査ログの DB schema / repository / migration。
  - 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API。
  - 認証、認可、Cookie session、CSRF token、password hash、RBAC、logout。
  - PostgreSQL transaction、悲観的行ロック、冪等性キー、監査ログ永続化、失敗監査ログの独立 transaction。
  - 取引種別 `deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` の Go domain helper と、取引種別に応じた残高反映 helper。
- 未設計または未確定:
  - 1 回あたりの金額上限、口座残高上限、日次上限。
  - domain error から API response / audit `failure_reason` への mapping。
  - `ip_address` / `user_agent` / request body hash の正規化、最大長、信頼境界。
  - DB transaction manager、2 口座振込時の lock 順序、分離レベル。
  - 口座 lifecycle の状態遷移表、残高あり解約の扱い、未完了振込依頼がある口座の解約可否。
- docs / 実装不一致:
  - README は現行実装と大きく矛盾していない。
  - `docs/data-model.md` は `transactions.transaction_type` と `balance_after`、`deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` の残高方向を定義しているが、Go domain code にはまだ取引種別 helper がない。
  - `docs/data-model.md` は `reversal` を候補に含める一方で利用条件未確定とし、human notes は reversal を MVP に含めないとしている。従って次 scope で `reversal` を実装対象にしないのが妥当。
- レビュー未反映:
  - 2026-07-01-001 reviewer 群は、直近の口座ステータス helper に blocking を出していない。
  - 次工程入力として、domain error / API response / audit `failure_reason` mapping、DB constraint、transaction 内 row lock、owner / role authorization、口座状態遷移表、元帳・取引履歴・冪等性の未実装を挙げている。
- TODO / FIXME:
  - 実装コード上の明示的な TODO / FIXME は確認できなかった。未実装事項は README、docs、cycle artifact に記録されている。
- 現状確認として `go test ./...` を実行し、`cmd/server`、`internal/domain`、`internal/httpapi` のテストが成功した。

## 入力レビュー

- human notes:
  - reversal は MVP に含めない。通常取引を先に明確にする。
  - PostgreSQL は学習目的で行ロックによる悲観ロックを採用する方針。
  - 冪等性キーには操作種別、送信元口座、ログインユーザー、request body hash を含めるべきという意見。
  - MVP の冪等性キー重複は既存結果返却ではなく拒否でよい。
  - 監査ログは成功・失敗とも残す。失敗監査ログも独立 transaction で残す。
  - 認証情報は Cookie、CSRF token も別に持つ。
  - `operator` は MVP 対象外、`admin` は代行可能。
  - 直近 human review では、小さい実装の粒度を少し大きくしてもよいという入力がある。
- 2026-07-01-001 code-reviewer:
  - 口座ステータス helper は accepted scope に適合しており blocking なし。
  - 次 cycle では、`ErrInvalidAccountStatus` と `ErrAccountNotActive` を API response と audit `failure_reason` にどう写像するかを明示することを推奨。
  - DB schema に進む場合は、`accounts.status` CHECK constraint、`accounts.balance_amount >= 0`、transaction boundary、悲観ロック順序をまとめて扱うのが望ましい。
- 2026-07-01-001 security-reviewer:
  - 新規 HTTP / DB / 認証境界は追加されておらず、直接的な外部攻撃面の増加はない。
  - `EnsureAccountCanTransact` は認可ではなく業務状態 gate であり、後続実装で owner / role check と混同しないこと。
  - `ErrAccountNotActive` は suspended / closed を同一 error にまとめるため、監査分類・外部応答 mapping を別途設計する必要がある。
  - DB 制約・transaction・行ロックは未実装であり、永続化境界で再検証が必要。
- 2026-07-01-001 banking-reviewer:
  - 口座ステータス判定は残高変更 gate として accepted scope に合致しており blocking なし。
  - 口座状態遷移の許可表は未定義であり、将来の口座作成・停止・解約で事故シナリオが残る。
  - 元帳・取引履歴・冪等性の保証は今回差分では増えておらず、業務 API 追加前の優先課題として残る。
- 既存設計文書:
  - `docs/design-principles.md` は、すべての残高変更に取引履歴を残し、入金・出金・振込の成功時には口座残高、取引履歴、成功監査ログを同じ PostgreSQL transaction に含める方針を定義している。
  - `docs/data-model.md` は、`transactions.transaction_type` ごとの残高増減方向と `balance_after` の意味を定義している。
  - `docs/test-strategy.md` は、金額計算と残高更新、停止中・解約済み口座の拒否、取引履歴と監査ログ、冪等性、transaction rollback を重点テスト対象にしている。

## 改善候補

| 候補 | 根拠 | MVP に入れる理由 | reviewer 観点 | 注意点 |
| --- | --- | --- | --- | --- |
| A. 取引種別と残高反映の domain helper を追加する | `docs/data-model.md` は `transaction_type` ごとの残高方向と `balance_after` を定義済みだが code 未実装 | DB/API 前に、入金・出金・振込出金・振込入金の残高方向を pure domain code と unit test で固定できる | banking: 元帳方向、残高非負、reversal 除外。code: 小さい pure domain。security: 新規外部境界なし | 取引履歴永続化、監査ログ、DB transaction、冪等性は含めない |
| B. domain error / API response / audit `failure_reason` mapping を docs 化する | 2026-07-01-001 reviewer 群が繰り返し推奨 | 監査ログ・API 実装前に外部露出と内部分類を分けられる | security: 情報露出抑制、監査分類。code: handler/service 境界 | docs-only になりやすく、今回の code-changing scope 要件を満たしにくい |
| C. 口座状態遷移表を docs 化する | banking-reviewer が状態遷移未定義を指摘 | 将来の口座停止・再開・解約 API の事故を防ぐ | banking: closed 復活禁止、残高 0 条件。security: admin 代行監査 | 今回は口座管理 API も audit persistence も未実装で docs-only になりやすい |
| D. PostgreSQL schema / migration skeleton を追加する | data model と reviewer が DB 制約を推奨 | 残高・取引履歴の最終防衛線を作れる | code/banking/security: CHECK、FK、UNIQUE、lock、PII | driver / migration tool / transaction manager / lock 順序が未確定で今回には大きい |
| E. 認証・認可 gate の最小設計または code skeleton を追加する | 業務 API 前に必要。security-reviewer が owner / role check の混同を指摘 | active な他人口座への操作を防ぐ前提 | security: horizontal authorization、CSRF、RBAC | Cookie/CSRF/session 管理や admin 作成方法が絡み、現段階の code scope には大きい |

## 採択

### 採択 A: 取引種別と残高反映の domain helper を追加する

- 採択理由:
  - `docs/data-model.md` と `docs/design-principles.md` で既に定義された「取引種別ごとの残高増減方向」と「取引後残高」を、DB/API なしの小さな Go domain code に落とせる。
  - 直近 reviewer が指摘した「元帳・取引履歴の保証は未実装」という課題に対し、永続化へ進む前の最小部品を作れる。
  - 既存 `Amount` / `Balance` / `AccountStatus` と同じ `internal/domain` の pure helper に限定でき、新規 HTTP / DB / 認証境界を増やさない。
  - human notes の「reversal は MVP に含めない」に沿い、`reversal` を valid transaction type から除外する test を追加できる。
  - 直近 human review の「実装粒度を少し大きくしてもよい」に対し、金額・残高 helper を再利用する程度の小さな code-changing scope として適切。
- 期待する効果:
  - 後続の入金・出金・振込 service で、取引種別ごとの残高方向を handler / service ごとに重複実装しなくてよくなる。
  - `transfer_debit` と `transfer_credit` の方向を unit test で固定し、振込時の片側方向ミスを早期に検出しやすくなる。
  - `reversal` を未確定のまま実装に混ぜない境界が明確になる。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| D. PostgreSQL schema / migration skeleton | 重要だが、migration tool、driver、transaction manager、lock 順序、監査ログ保存境界、冪等性一意制約まで絡み、今回の小 scope には大きい | 取引種別・残高方向 helper、error / audit mapping、lock 順序 docs が揃った後 |
| E. 認証・認可 gate の code skeleton | Cookie + CSRF、session、RBAC、admin 代行範囲、監査 actor 設計が絡む。中途半端な認可 stub は業務 API 実装時に誤用されやすい | 認証方式と owner / role authorization の docs scope を先に採択した後 |
| 業務 API skeleton | 認証、認可、DB transaction、監査ログ、冪等性が未実装のまま endpoint を増やすと、学習用でも危険な半端な業務 API になる | 最小の DB/repository/service/audit 方針が固まった後 |

## 保留

| 候補 | 保留理由 | 次のアクション |
| --- | --- | --- |
| B. domain error / API response / audit `failure_reason` mapping | security / banking 観点では高優先だが、今回の code-changing 要件を満たすため、先に pure domain の取引種別 helper を採択する | 次 cycle で `ErrAmountMustBePositive`、`ErrInsufficientBalance`、`ErrInvalidAccountStatus`、`ErrAccountNotActive`、新規 `ErrInvalidTransactionType` を含む mapping 表を作る |
| C. 口座状態遷移表 | 口座管理 API 前に必要だが、今回の code scope は取引種別・残高方向に限定する | `active -> suspended`、`suspended -> active`、`active/suspended -> closed`、`closed -> *` 禁止案と audit 要件を docs 化する |
| 行ロック・DB transaction 方針 | 入出金・振込 service 前に必要だが、DB 実装の直前 cycle として扱う方がよい | 2 口座振込時の lock 順序、分離レベル、失敗監査ログ独立 transaction を docs 化する |
| 冪等性キー設計 | human notes に具体入力があり重要だが、transfer request / DB 一意制約と同時に扱う必要がある | 操作種別、送信元口座、ログインユーザー、request body hash の組み合わせと、重複時拒否を docs 化する |
| 金額上限・残高上限 | API 接続前に必要だが、上限値は業務判断が強い | 暫定上限を作業仮定として置けるか次 cycle 以降で判断する |

## accepted scope

### 目的

- 入金、出金、振込出金、振込入金の取引種別を Go domain 層で表現し、取引種別ごとの残高反映方向を unit test で固定する。
- `docs/data-model.md` の `transaction_type` と `balance_after` の設計を、DB / API 実装前の pure domain helper として具体化する。
- `reversal` は human notes に従って MVP から除外し、未確定の取消仕様を暗黙に実装しない。

### 対象ファイル/領域

- `internal/domain/`
  - `transaction.go` を追加する。
  - 既存 `money.go` の `Amount` / `Balance` / `AddBalance` / `SubtractBalance` を再利用する。
- `internal/domain/*_test.go`
  - `transaction_test.go` を追加する。
- `README.md`
  - 現在の実装範囲に、取引種別 validation と取引種別に応じた残高反映 helper が追加されたことを反映する。
  - 未実装機能リストと矛盾しないよう、取引履歴永続化、DB schema、監査ログ、業務 API は引き続き未実装と明記する。
- `docs/ai/cycles/2026-07-05-001/implementer.md`
  - implementer は、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

### 実装対象

1. `internal/domain/transaction.go` を追加する。
2. `TransactionType` を string-based type として定義する。
3. MVP で valid な取引種別は、次の 4 つだけにする。
   - `TransactionTypeDeposit = "deposit"`
   - `TransactionTypeWithdrawal = "withdrawal"`
   - `TransactionTypeTransferDebit = "transfer_debit"`
   - `TransactionTypeTransferCredit = "transfer_credit"`
4. `ErrInvalidTransactionType = errors.New("invalid transaction type")` を追加する。
5. `func (t TransactionType) Validate() error` を追加する。
   - `deposit`、`withdrawal`、`transfer_debit`、`transfer_credit` は valid とする。
   - 空文字、未知値、`reversal` は `ErrInvalidTransactionType` で拒否する。
6. `func ApplyTransaction(balance Balance, amount Amount, transactionType TransactionType) (Balance, error)` を追加する。
   - 戻り値は「取引種別を残高へ反映した後の残高」を表す。
   - validation / error 優先順位は、`balance.Validate()`、`amount.Validate()`、`transactionType.Validate()` の順にする。
   - 上記 validation のいずれかで error になった場合、元の `balance` を返す。
   - `deposit` と `transfer_credit` は `AddBalance` と同じ方向で残高を増やす。
   - `withdrawal` と `transfer_debit` は `SubtractBalance` と同じ方向で残高を減らす。
   - 残高不足、overflow、invalid amount、invalid balance は、既存 `money.go` の sentinel error をそのまま返す。
7. `internal/domain/transaction_test.go` を追加する。
   - `TransactionType.Validate()` が 4 種の valid type を受け付けること。
   - 空文字、未知値、`reversal` を `ErrInvalidTransactionType` で拒否すること。
   - `ApplyTransaction` が `deposit` と `transfer_credit` で残高を増やすこと。
   - `ApplyTransaction` が `withdrawal` と `transfer_debit` で残高を減らすこと。
   - `withdrawal` / `transfer_debit` で残高不足の場合、元の残高を返し、`ErrInsufficientBalance` を返すこと。
   - invalid starting balance の場合、元の残高を返し、`ErrBalanceMustBeNonNegative` を返すこと。
   - invalid amount の場合、元の残高を返し、`ErrAmountMustBePositive` を返すこと。
   - invalid transaction type の場合、元の残高を返し、`ErrInvalidTransactionType` を返すこと。
   - deposit-like type で overflow する場合、元の残高を返し、`ErrBalanceOverflow` を返すこと。
8. README を更新する。
   - 現在の実装範囲に、取引種別 validation と残高反映 helper を追記する。
   - 「取引履歴の永続化」「transaction row の作成」「`balance_after` の DB 保存」「監査ログ」「業務 API」は未実装のままと明記する。
9. `docs/ai/cycles/2026-07-05-001/implementer.md` を作成する。

### 実装しないこと

- `Transaction` aggregate、取引 ID、取引日時、説明、関連取引 ID、`balance_after` field を持つ永続 entity は作らない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、行ロックは作らない。
- 取引履歴 table への insert、監査ログ table への insert、成功 / 失敗監査ログの保存境界は実装しない。
- HTTP route、handler、request / response schema は追加しない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装しない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC は実装しない。
- 冪等性キー、request body hash、transfer request 状態遷移は実装しない。
- `reversal` / 取消 / 組戻し / 訂正は実装しない。`reversal` は `ErrInvalidTransactionType` で拒否する test 対象に限定する。
- 金額上限、残高上限、日次上限は今回の code に固定しない。
- domain error / API response / audit `failure_reason` mapping は今回確定しない。

### テスト方針

- 追加・変更した Go ファイルに `gofmt` を適用する。
- `go test ./...` を実行し、既存 server / router / money / account tests と新規 transaction type tests がすべて成功することを確認する。
- `git diff --name-only` で、変更が accepted scope 内の `internal/domain`、`README.md`、`docs/ai/cycles/2026-07-05-001/implementer.md` に限定されていることを確認する。
- 必要に応じて `rg -n "TransactionType|ApplyTransaction|ErrInvalidTransactionType|reversal" internal/domain README.md docs/ai/cycles/2026-07-05-001/implementer.md` で、取引種別と reversal 除外の説明が揃っていることを確認する。

### 作業仮定

- `TransactionType` は DB schema の最終 enum / CHECK constraint 名を確定するものではなく、現時点の Go domain helper 名とする。ただし文字列値は `docs/data-model.md` の `transaction_type` 値に合わせる。
- `ApplyTransaction` は取引履歴を作成しない。返す残高は、将来 transaction row の `balance_after` に保存すべき値の計算候補である。
- validation / error 優先順位は、破損した starting balance を最初に検出するため、`balance`、`amount`、`transactionType` の順にする。
- `transfer_debit` と `transfer_credit` は 1 件の振込における片側取引の残高方向だけを表す。2 口座の同一 DB transaction、lock 順序、冪等性キー、振込依頼状態は今回扱わない。
- `reversal` は data model では将来候補として残るが、human notes に従い MVP 初期 code では valid type に含めない。

### レビューで重点確認してほしい観点

- `deposit` / `transfer_credit` が増加方向、`withdrawal` / `transfer_debit` が減少方向として実装・テストされているか。
- `reversal` が誤って valid type に含まれていないか。
- invalid transaction type、invalid amount、invalid balance、残高不足、overflow が元の残高を返しているか。
- helper が HTTP、DB、認証、認可、監査ログ、冪等性に依存していないか。
- README が現行実装範囲を過大に主張せず、取引履歴永続化や業務 API が未実装であることを維持しているか。

## 実装しないこと

- この planner 作業では、`docs/ai/cycles/2026-07-05-001/planner.md` 以外を書き換えない。
- Go ソースコード、テストコード、README、通常 docs は変更しない。
- 親 agent として実装代行しない。implementer はこの accepted scope だけを実装する。
- 他 agent と直接同期しない。既存 cycle 成果物は repo 上の入力 artifact としてのみ扱う。
- 金融仕様の最終決定は行わず、既存 docs と human notes に沿う範囲を作業仮定として実装可能な小 scope に落とす。

## 作業仮定

- cycle id はユーザー指定どおり、current_date 2026-07-05 に基づく新規 `2026-07-05-001` とする。既存 `2026-07-01-001` は新規 cycle として再利用しない。
- planner の書き込みは `docs/ai/cycles/2026-07-05-001/planner.md` のみに限定する。
- この repo は学習用ミニバンキングシステムであり、本番金融システム相当の完成度をこの cycle で主張しない。
- Go 標準ライブラリのみの方針は維持する。外部ライブラリや DB driver は今回追加しない。
- 次の implementer は、accepted scope にない判断を増やさず、不明点は `implementer.md` の作業仮定として記録する。
