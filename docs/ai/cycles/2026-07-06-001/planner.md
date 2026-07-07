# planner: 2026-07-06-001

## repo現状

- 作業開始時に `git status --short` を確認した。表示は空で、未コミット変更は確認しなかった。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、`.agents/skills/banking-planning/SKILL.md` を確認した。planner は実装せず、agent 間同期は `docs/ai/cycles/<cycle-id>/` 配下の Markdown 成果物だけで行う。今回の保存先は `docs/ai/cycles/2026-07-06-001/planner.md` のみとする。
- README 上の現行実装範囲は、Go 標準ライブラリだけを使った最小 REST API server、`GET /healthz`、金額・残高 validation、口座ステータス validation、取引種別 validation と取引種別に応じた残高反映 helper である。
- 実装済み:
  - `cmd/server`: `BANK_SYSTEM_HTTP_ADDR` による listen address 設定、HTTP server timeout、`internal/httpapi.NewRouter()` 接続。
  - `internal/httpapi`: `GET /healthz`、method 制限、固定 JSON response、運用詳細を返さない test。
  - `internal/domain/money.go`: JPY の整数最小通貨単位を `int64` で扱う `Amount` / `Balance`、正の取引金額、0 以上の残高、残高加算、残高減算、残高不足、overflow、constructor bypass 値を再検証する `Validate()`。
  - `internal/domain/account.go`: `AccountStatus` の `active` / `suspended` / `closed` validation と、`active` のみ残高変更系操作へ進める `EnsureAccountCanTransact`。
  - `internal/domain/transaction.go`: MVP の `deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` の validation、`reversal` の拒否、取引種別に応じた `ApplyTransaction`。
- 設計済みだが未実装:
  - 顧客、ログインユーザー、口座、取引履歴、振込依頼、監査ログの DB schema / repository / migration。
  - 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API。
  - 認証、認可、Cookie session、CSRF token、password hash、RBAC、logout。
  - PostgreSQL transaction、悲観的行ロック、2 口座振込時の lock 順序、冪等性キー、失敗監査ログの独立 transaction。
  - domain error から API response / audit `failure_reason` / safe log category への mapping。
- 未設計または未確定:
  - 1 回あたりの金額上限、口座残高上限、日次上限。
  - `ip_address` / `user_agent` / request body hash の正規化、最大長、信頼境界。
  - DB transaction manager、分離レベル、repository interface、migration tool。
  - 口座 lifecycle の状態遷移表、残高あり解約の扱い、未完了振込依頼がある口座の解約可否。
  - 同一口座間振込の扱い、冪等性キーの一意範囲と重複時レスポンスの詳細。
- docs / 実装不一致:
  - README は現行実装と大きく矛盾していない。
  - `docs/security-notes.md`、`docs/design-principles.md`、`docs/data-model.md` は監査ログの `failure_reason` や安全な分類の必要性を示しているが、Go code には safe failure category / reason helper がまだない。
  - `docs/data-model.md` は `reversal` を将来候補として記載しているが、human notes と現行 code は MVP 初期で `reversal` を valid type に含めない方針で一致している。
- レビュー未反映:
  - 2026-07-05-001 reviewer 群は、取引種別・残高反映 helper に blocking を出していない。
  - 2026-07-05-001 reviewer 群は、次サイクル入力として domain error / API response / audit `failure_reason` mapping、入出金・振込 service 前の security gate 順序、DB transaction 境界、row lock、冪等性キー設計を挙げている。
- TODO / FIXME:
  - 実装コード上の明示的な TODO / FIXME は確認できなかった。未実装事項は README、docs、cycle artifact に記録されている。
- 現状確認として `go test ./...` を実行し、`cmd/server`、`internal/domain`、`internal/httpapi` のテストが成功した。

## 入力レビュー

- human notes:
  - `reversal` は MVP に含めず、通常取引を先に明確にする。
  - PostgreSQL は学習目的で行ロックによる悲観ロックを採用する。
  - 冪等性キーには操作種別、送信元口座、ログインユーザー、request body hash を含めるべきという入力がある。
  - MVP の冪等性キー重複時は既存結果返却ではなく拒否でよい。
  - 監査ログは成功・失敗とも残し、失敗監査ログも独立 transaction で残す。
  - 認証情報は Cookie とし、CSRF 対策用 token も別に持つ。
  - `operator` は MVP 対象外、`admin` は代行可能。
  - 直近 human review では、小さい実装の粒度を少し大きくしてもよいという入力がある。
- 2026-07-05-001 code-reviewer:
  - blocking finding はない。
  - `ErrInvalidTransactionType` が追加されたため、既存の `ErrAmountMustBePositive`、`ErrInsufficientBalance`、`ErrInvalidAccountStatus`、`ErrAccountNotActive` と合わせて、API response / audit `failure_reason` mapping を決める必要がある。
  - DB 実装へ進む場合は、Go domain helper と PostgreSQL CHECK constraint を対応付けることを推奨。
  - 入出金 service 前に、`EnsureAccountCanTransact` と `ApplyTransaction` の呼び出し順序、失敗時監査ログの transaction 境界を明示することを推奨。
- 2026-07-05-001 security-reviewer:
  - blocking finding はない。
  - 外部攻撃面は増えていないが、`ApplyTransaction` は認証、CSRF、owner / role authorization、口座状態 gate、冪等性、監査ログ、DB transaction を保証しない。
  - API response / audit `failure_reason` mapping を次 cycle で設計し、raw request body、password、token、secret、CSRF token、session ID、過剰な個人情報、未加工の自由入力値を監査ログに保存しないことを推奨。
- 2026-07-05-001 banking-reviewer:
  - blocking finding はない。
  - 取引種別ごとの残高方向は設計と一致しているが、現 helper は `balance_after` の計算候補であり、元帳完成ではない。
  - 次に service を実装する前に、認証・認可、口座存在、口座ステータス、金額、取引種別、行ロック、残高計算、取引履歴、成功監査ログ、失敗監査ログの順序を固定することを推奨。
  - `ErrInvalidTransactionType`、金額・残高・口座状態系 error を API response と audit `failure_reason` にどう写像するかを docs 化することを推奨。
- 既存設計文書:
  - `docs/design-principles.md` は、成功した残高変更で口座残高、取引履歴、成功監査ログを同じ PostgreSQL transaction に含め、失敗監査ログは独立 transaction で残す方針を定義している。
  - `docs/security-notes.md` は、`failure_reason` は利用者向け詳細ではなく安全な分類または短い理由にし、raw request body や secret を監査ログに保存しない方針を示している。
  - `docs/test-strategy.md` は、監査ログの failure reason、マスキング、閲覧権限、DB transaction rollback を重点テスト対象にしている。

## 改善候補

| 候補 | 根拠 | MVP に入れる理由 | reviewer 観点 | 注意点 |
| --- | --- | --- | --- | --- |
| A. domain error を safe failure category に写像する Go helper を追加する | 2026-07-05-001 reviewer 群が API response / audit `failure_reason` mapping を次優先に挙げた。既存 domain sentinel error は揃いつつある | 業務 API / 監査ログ実装前に、外部応答・監査分類・ログ分類で使える安全な機械可読分類を固定できる | security: 情報露出抑制、raw input 保存禁止。code: sentinel error mapping。banking: 残高・口座状態・取引種別の失敗分類 | API response schema や audit table は実装しない。mapping helper を「domain error の安全な分類」に限定する |
| B. 入出金・振込 service gate 順序を docs 化する | banking/security reviewer が `ApplyTransaction` 単独利用の危険を指摘 | service 実装前に、認証、認可、口座状態、金額、取引種別、DB transaction、監査ログの順序を共有できる | banking/security: 金融事故防止、認可 bypass 防止 | docs-only になり、今回の code-changing accepted scope 要件を単独では満たしにくい |
| C. 口座状態遷移の domain helper を追加する | 過去 reviewer が状態遷移表未定義を指摘。現 code は status validation と active gate のみ | 口座作成・停止・再開・解約 API 前に、不正な closed 復活や残高あり解約の前提を固定できる | banking: lifecycle 事故防止。security: admin 代行監査 | 残高あり解約、未完了振込依頼あり解約の仕様が未確定。今回の reviewer 最優先は error mapping |
| D. PostgreSQL schema / migration skeleton を追加する | data model と reviewer が DB 制約を推奨 | DB 制約で残高・取引履歴・status・transaction type の最終防衛線を作れる | code/banking/security: CHECK、FK、UNIQUE、row lock | migration tool、driver、transaction manager、lock 順序、監査境界が未確定で今回には大きい |
| E. 認証・認可 skeleton を追加する | 業務 API 前に必要。security-reviewer が owner / role check の混同を指摘 | active な他人口座への操作を防ぐ前提を作れる | security: horizontal authorization、CSRF、RBAC | Cookie/CSRF/session/password hash/admin 初期化が絡み、現段階の code scope には大きい |
| F. 最小入金 API / service skeleton を追加する | domain helper が増え、human notes は粒度拡大を許容 | MVP の業務 API 実装へ進める | banking: 残高・履歴・監査。security: 認証認可 | DB、認証、監査、冪等性、transaction 境界が未実装のまま endpoint を増やすのは premature |

## 採択

### 採択 A: domain error を safe failure category に写像する Go helper を追加する

- 採択理由:
  - 直近 3 reviewer が共通して、業務 API / 監査ログ実装前に domain error から API response / audit `failure_reason` への mapping を明示する必要を挙げている。
  - 既存 code には金額、残高、口座状態、取引種別の sentinel error が揃っており、DB / HTTP / 認証境界を増やさず pure Go helper と unit test に落とせる。
  - docs-only ではなく、次の implementer が code-changing scope として `internal/domain` に小さな分類 helper と test を追加できる。
  - audit table や HTTP response schema を先に作らず、「安全な分類値」を先に固定することで、後続 handler / service / audit 実装時の情報露出と分類揺れを減らせる。
  - human notes の「失敗監査ログも独立して残す」方針に対し、失敗監査ログへ保存してよい分類値の土台を作れる。
- 期待する効果:
  - 後続 API が domain error の `Error()` 文字列や未加工 input をそのまま利用者応答・監査ログへ流すリスクを下げる。
  - `ErrInvalidTransactionType` を含む新旧 domain error の分類が unit test で固定される。
  - 入出金・振込 service 実装前に、業務拒否理由を安全な機械可読値として扱う準備ができる。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| D. PostgreSQL schema / migration skeleton | 重要だが、migration tool、driver、transaction manager、lock 順序、監査ログ保存境界、冪等性一意制約まで絡み、今回の小 scope には大きい | safe failure category と service gate 順序、DB transaction / lock 方針が揃った後 |
| E. 認証・認可 skeleton | Cookie + CSRF、session、password hash、RBAC、admin 代行範囲、監査 actor 設計が絡む。中途半端な auth stub は後続 API で誤用されやすい | 認証方式、CSRF、owner / role authorization、admin 初期化の docs scope を先に採択した後 |
| F. 最小入金 API / service skeleton | 認証、認可、DB transaction、取引履歴、監査ログ、冪等性が未実装のまま endpoint を増やすと、学習用でも危険な半端な業務 API になる | DB/repository/service/audit/auth の最小方針が固まり、failure category と gate 順序が実装・文書化された後 |
| `reversal` / 取消 / 組戻し / 訂正の実装 | human notes で MVP に含めない方針。通常取引、取引履歴、監査、冪等性が固まる前に扱うと仕様が大きい | 通常の入金・出金・振込、二重実行防止、監査ログが揃った後、別 cycle で設計する |

## 保留

| 候補 | 保留理由 | 次のアクション |
| --- | --- | --- |
| B. 入出金・振込 service gate 順序 docs | 今回は code-changing scope を優先するが、A の helper を後続 service で安全に使うため高優先 | 次 cycle で、認証、CSRF、owner / role authorization、口座状態、金額、取引種別、行ロック、取引履歴、監査ログの順序を docs 化する |
| C. 口座状態遷移 helper | 重要だが、今回 reviewer の最優先入力は failure mapping。残高あり解約など未確定も残る | 口座作成・停止・再開・解約 API 前に、状態遷移表と helper を採択する |
| DB transaction / row lock 方針 | 入出金・振込 service 前に必須だが、今回の helper 追加と同時に進めるには大きい | 2 口座振込時の lock 順序、分離レベル、失敗監査ログ独立 transaction を docs 化する |
| 冪等性キー設計 | human notes に具体入力があり重要だが、transfer request / DB 一意制約と同時に扱う必要がある | 操作種別、送信元口座、ログインユーザー、request body hash の組み合わせと、重複時拒否の分類を docs 化する |
| API response schema | failure category と関連するが、HTTP API 全体の error format、status code、認証失敗時の扱いが絡む | safe failure category helper の後、handler 追加前に別 scope で設計する |
| 金額上限・残高上限・日次上限 | API 接続前に必要だが、上限値は業務判断が強い | 暫定上限を作業仮定として置けるか次 cycle 以降で判断する |

## accepted scope

### 目的

- 既存 domain sentinel error を、利用者応答・監査ログ・安全なログ分類で再利用しやすい stable な failure category に写像する pure Go helper を追加する。
- 業務 API / 監査ログ永続化の前に、domain error の `Error()` 文字列、未加工 request body、未加工 transaction type、秘密情報、過剰な個人情報を外部応答や監査ログへ流さないための分類土台を作る。
- `ErrInvalidTransactionType` を含む現行 domain error 群を unit test で分類固定し、未知 error は安全に「未分類」として扱えるようにする。

### 対象ファイル/領域

- `internal/domain/`
  - `failure_reason.go` または同等名の新規ファイルを追加する。
  - 既存 `money.go`、`account.go`、`transaction.go` の sentinel error を mapping 対象にする。既存 sentinel error の文言や semantics は変更しない。
- `internal/domain/*_test.go`
  - `failure_reason_test.go` または同等名の新規テストを追加する。
- `README.md`
  - 現在の実装範囲に、domain error を安全な failure category へ写像する helper が追加されたことを反映する。
  - 業務 API、HTTP error response、監査ログ永続化、DB schema、認証、認可は引き続き未実装と明記する。
- `docs/security-notes.md` または `docs/design-principles.md`
  - 既存の「`failure_reason` は安全な分類にする」「raw request body や secret を保存しない」方針と、新規 helper の分類値を矛盾なく接続する短い追記を行う。
- `docs/ai/cycles/2026-07-06-001/implementer.md`
  - implementer は、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

### 実装対象

1. `internal/domain/failure_reason.go` を追加する。
2. `FailureReason` を string-based type として定義する。
3. MVP 初期の safe failure category として、少なくとも次の constants を追加する。
   - `FailureReasonInvalidAmount = "invalid_amount"`
   - `FailureReasonInvalidBalanceState = "invalid_balance_state"`
   - `FailureReasonInsufficientBalance = "insufficient_balance"`
   - `FailureReasonBalanceOverflow = "balance_overflow"`
   - `FailureReasonInvalidAccountStatus = "invalid_account_status"`
   - `FailureReasonAccountNotActive = "account_not_active"`
   - `FailureReasonInvalidTransactionType = "invalid_transaction_type"`
4. `func (r FailureReason) Validate() error` を追加する。
   - 上記 constants のみ valid とする。
   - 空文字、未知値、raw error message 風の値、自由入力値は invalid とする。
   - invalid 時の sentinel error 名は `ErrInvalidFailureReason = errors.New("invalid failure reason")` とする。
5. `func FailureReasonFromError(err error) (FailureReason, bool)` を追加する。
   - `errors.Is` を使い、wrapped error にも対応する。
   - `ErrAmountMustBePositive` -> `FailureReasonInvalidAmount`
   - `ErrBalanceMustBeNonNegative` -> `FailureReasonInvalidBalanceState`
   - `ErrInsufficientBalance` -> `FailureReasonInsufficientBalance`
   - `ErrBalanceOverflow` -> `FailureReasonBalanceOverflow`
   - `ErrInvalidAccountStatus` -> `FailureReasonInvalidAccountStatus`
   - `ErrAccountNotActive` -> `FailureReasonAccountNotActive`
   - `ErrInvalidTransactionType` -> `FailureReasonInvalidTransactionType`
   - `nil`、未知 error、対象外 error は `"", false` を返す。
6. `internal/domain/failure_reason_test.go` を追加する。
   - `FailureReason.Validate()` が定義済み constants を受け付けること。
   - 空文字、未知値、既存 sentinel error の raw `Error()` 文字列、`password=...` のような secret 風値を `ErrInvalidFailureReason` で拒否すること。
   - `FailureReasonFromError` が各 sentinel error を期待する `FailureReason` に写像すること。
   - wrapped error でも `errors.Is` により写像できること。
   - `nil` と未知 error は `"", false` になること。
   - `FailureReasonFromError` で得た値が `Validate()` に通ること。
7. README を更新する。
   - 現在の実装範囲に safe failure category helper を追記する。
   - 未実装機能リストでは、HTTP error response、監査ログ永続化、DB schema、業務 API、認証、認可が未実装であることを維持する。
8. `docs/security-notes.md` または `docs/design-principles.md` を更新する。
   - 監査 `failure_reason` では helper の safe category を使う方針を短く追記する。
   - helper は raw request body、token、secret、CSRF token、session ID、自由入力値を保存するものではないことを明記する。
9. `docs/ai/cycles/2026-07-06-001/implementer.md` を作成する。

### 実装しないこと

- HTTP route、handler、request / response schema、status code mapping は追加しない。
- API response body の最終形式、利用者向け message 文言、i18n、エラーコード体系全体は確定しない。
- 監査ログ table、audit repository、audit service、成功 / 失敗監査ログの永続化は実装しない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、row lock は作らない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC、admin 代行処理は実装しない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装しない。
- 取引履歴 row、`balance_after` 永続化、transfer request 状態遷移、冪等性キー処理は実装しない。
- `reversal` / 取消 / 組戻し / 訂正は実装しない。
- 金額上限、残高上限、日次上限は今回の code に固定しない。
- Go の sentinel error message を外部仕様として扱わない。helper が返す string constants だけを安全な分類値候補とする。

### テスト方針

- 追加・変更した Go ファイルに `gofmt` を適用する。
- `go test ./...` を実行し、既存 server / router / money / account / transaction tests と新規 failure reason tests がすべて成功することを確認する。
- `git diff --name-only` で、変更が accepted scope 内の `internal/domain`、`README.md`、`docs/security-notes.md` または `docs/design-principles.md`、`docs/ai/cycles/2026-07-06-001/implementer.md` に限定されていることを確認する。
- `rg -n "password|token|secret|CSRF|session|request body" internal/domain docs README.md` などで、新規 helper や docs が raw secret / raw request body を保存する実装に読めないことを確認する。

### レビューで重点確認してほしい観点

- `FailureReasonFromError` が `errors.Is` を使い、wrapped sentinel error を安全に分類できるか。
- 未知 error や `nil` を安全に未分類扱いにし、誤って raw error message を category として返していないか。
- safe category の名前が API / audit で使っても過剰な内部情報や利用者の自由入力を含まないか。
- README と security docs が、helper を「監査ログ永続化済み」「HTTP error response 実装済み」と誤読させていないか。
- `ErrInvalidTransactionType`、口座状態 error、金額・残高 error の分類が後続の入出金・振込 service で使いやすいか。
- 今回 scope が HTTP / DB / auth / audit persistence / idempotency へ踏み込んでいないか。

## 実装しないこと

- planner として実装代行しない。今回の変更は `docs/ai/cycles/2026-07-06-001/planner.md` の作成のみとする。
- agent 間の直接同期は行わない。implementer / reviewer への入力は、この Markdown 成果物だけに残す。
- accepted scope 外の code、README、docs、DB schema、migration、API、認証、監査ログ永続化、冪等性処理をこの turn で変更しない。
- 本番金融システムとしての完成や安全性を断定しない。学習用ミニバンキングシステムの小さな前進として扱う。

## 作業仮定

- `FailureReason` は audit `failure_reason` と API error response の共通候補になり得るが、今回 scope では最終的な HTTP status code、response schema、利用者向け文言を確定しない。
- `FailureReason` は raw request body、password、token、secret、CSRF token、session ID、過剰な個人情報、未加工の自由入力値を格納する場所ではない。
- `ErrBalanceMustBeNonNegative` は通常の利用者入力不備というより永続化境界や内部状態異常に近いため、safe category は `invalid_balance_state` として扱う。
- `ErrBalanceOverflow` は現在の `int64` 上限到達を示す domain error として `balance_overflow` に分類する。金額上限・残高上限の業務仕様は別 cycle で扱う。
- `ErrAccountNotActive` は suspended / closed の詳細を外部へ出さない分類として `account_not_active` にまとめる。詳細な状態別監査が必要かは後続 cycle で検討する。
- `reversal` は引き続き MVP 初期では invalid transaction type として扱う。
- 次の implementer は、accepted scope にない判断を増やさず、不明点は `implementer.md` の作業仮定として記録する。
