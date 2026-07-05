# planner: 2026-07-01-001

## repo現状

- 作業開始時に `git status --short` を確認した。表示は空で、未コミット変更は確認しなかった。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md` を確認した。cycle は `planner -> implementer -> reviewer 群` の順で、agent 間の同期は `docs/ai/cycles/<cycle-id>/` 配下の成果物だけで行う。
- README 上の現行実装範囲は、Go 標準ライブラリだけの最小 REST API server、`GET /healthz`、JPY `int64` の金額・残高 domain helper である。
- 実装済み:
  - `cmd/server`: `BANK_SYSTEM_HTTP_ADDR` による listen address 設定、HTTP server timeout、`internal/httpapi.NewRouter()` 接続。
  - `internal/httpapi`: `GET /healthz` と method 制限、固定 JSON response、運用詳細を返さない test。
  - `internal/domain`: `Amount` / `Balance`、0 円以下の取引金額拒否、負の残高拒否、残高加算、残高減算、残高不足、overflow 検出、constructor bypass 値の `Validate()`。
- 設計済みだが未実装:
  - 顧客、ログインユーザー、口座、取引履歴、振込依頼、監査ログの DB schema / repository / migration。
  - 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API。
  - 認証、認可、Cookie + CSRF、RBAC、password hash、logout。
  - PostgreSQL transaction、悲観的行ロック、冪等性キー、監査ログ永続化、失敗監査ログの独立 transaction。
- 未設計または未確定:
  - 1 回あたりの金額上限、口座残高上限、日次上限。
  - domain error から API response / audit `failure_reason` への mapping。
  - `ip_address` / `user_agent` / request body hash の正規化、最大長、信頼境界。
  - DB transaction manager、2 口座振込時の lock 順序、分離レベル。
- docs / 実装不一致:
  - README は現行実装と大きく矛盾していない。
  - docs は口座ステータスとして有効・停止中・解約済み、停止中/解約済み口座の入出金・振込拒否を要求しているが、Go domain code にはまだ口座状態の domain helper がない。
  - docs は DB 制約・監査ログ・冪等性を要求しているが、README どおり未実装であり、現時点では未実装事項として整合している。
- レビュー未反映:
  - 2026-06-30-001 reviewer 群は blocking を出していない。
  - 次工程入力として、DB 制約、行ロック、監査分類、金額上限、口座状態、repository 境界 validation を挙げている。
- TODO / FIXME:
  - 実装コード上の明示的な TODO / FIXME は確認できなかった。未実装事項は README、docs、cycle artifact に記録されている。

## 入力レビュー

- human notes:
  - reversal は MVP に含めない。
  - PostgreSQL は学習目的で行ロックによる悲観ロックを採用する方針。
  - 冪等性キーには操作種別、送信元口座、ログインユーザー、request body hash を含めるべきという意見。
  - MVP の冪等性キー重複は既存結果返却ではなく拒否でよい。
  - 監査ログは成功・失敗とも残す。失敗監査ログも独立 transaction で残す。
  - 認証情報は Cookie、CSRF token も別に持つ。
  - `operator` は MVP 対象外、`admin` は代行可能。
  - 直近 human review では、小さい実装の粒度を少し大きくしてもよいという入力がある。
- 2026-06-30-001 code-reviewer:
  - 開始 `Balance` 再検証は scope どおりで blocking なし。
  - 将来の error mapping / 監査分類で `ErrBalanceMustBeNonNegative` と通常の金額不備・残高不足を分ける必要がある。
  - DB / transaction / lock の検証はまだ存在しない。
- 2026-06-30-001 security-reviewer:
  - 今回までの差分で新しい認証・認可リスクは増えていない。
  - 業務上限と audit `failure_reason` は API 接続前に設計固定が必要。
  - DB 制約・行ロック・監査ログ永続化は未実装で、domain helper だけを最終防衛線にしない。
- 2026-06-30-001 banking-reviewer:
  - 破損残高からの演算継続リスクは低下した。
  - 元帳調査性のため、破損残高・残高不足・入力不備の分類を監査へ接続する必要がある。
  - 口座状態、取引履歴、DB transaction、行ロックは未着手。
  - 金額上限・残高上限は API 接続前に潰すべき。

## 改善候補

| 候補 | 根拠 | MVP に入れる理由 | reviewer 観点 | 注意点 |
| --- | --- | --- | --- | --- |
| A. 口座ステータス domain helper を追加する | `docs/domain-model.md`、`docs/mvp.md`、`docs/use-cases.md` は停止中・解約済み口座の入出金・振込拒否を要求しているが code 未実装 | DB/API 前に、口座状態の許可/拒否を service ごとに重複実装しない土台を作る | banking: 停止/解約済み口座の残高変更拒否。security: 認可とは別の業務状態 gate。code: 小さい pure domain + unit test | 口座番号採番、所有者認可、残高更新、DB schema は含めない |
| B. MVP 金額上限・残高上限を docs 化し、domain helper に上限を追加する | reviewers が上限未定義を指摘 | API 接続前の事故シナリオ削減 | security/banking: 異常高額取引拒否、監査分類 | 上限値は業務判断が強く、今回 code-changing scope としては仮定が大きい |
| C. domain error から API response / audit `failure_reason` への mapping docs を追加する | reviewers が監査分類未接続を指摘 | handler / audit 実装前に必要 | security/code: 外部露出メッセージと運用分類の分離 | docs-only になりやすく、今回の code-changing 要件を満たしにくい |
| D. PostgreSQL schema / migration skeleton を追加する | data model と reviewer が DB 制約を推奨 | 残高・取引履歴の最終防衛線を作れる | banking/code/security: CHECK 制約、transaction、lock | transaction manager、監査ログ境界、driver 選定が未確定で今回には大きい |
| E. healthz 以外の業務 API skeleton を追加する | MVP の進捗として見えやすい | 入出金などへ進める | security: 認証認可未実装 endpoint は危険 | DB、認証、監査、冪等性なしの endpoint は金融事故リスクが高い |

## 採択

### 採択 A: 口座ステータス domain helper を追加する

- 採択理由:
  - 既存 docs で明確に要求されている「有効口座のみ入出金・振込可能」を、DB/API 前の小さな pure domain code と test に落とせる。
  - 直近レビューの「domain helper だけに依存しない」「口座状態を確認する security gate」という入力に対し、まず口座状態 gate の最小部品を作れる。
  - human note の「実装粒度を少し大きくしてもよい」に沿いつつ、業務 API や DB へ踏み込まずレビューしやすい。
  - 既存 `internal/domain/money.go` と同じ粒度で、Go 標準ライブラリのみ・DB 非依存・HTTP 非依存にできる。
- 期待する効果:
  - 後続の入金・出金・振込 use case で、停止中/解約済み口座の拒否を一貫した sentinel error と unit test で扱える。
  - 金額 validation と口座状態 validation を分離し、認証・認可・監査ログとは別 gate であることを明確にできる。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| E. 業務 API skeleton | 認証、認可、DB transaction、監査ログ、冪等性が未実装のまま endpoint を増やすと、学習用でも危険な半端な業務 API になる | 認証/認可 gate、DB transaction、監査分類の最小設計後 |
| D. PostgreSQL schema / migration skeleton | 重要だが、driver / migration tool / transaction manager / lock 順序まで絡み、今回の小 scope には大きい | 口座・取引履歴・監査ログの domain / mapping 方針をもう一段固めた後 |
| B の domain 上限実装部分 | 上限値は業務判断が強く、人間確認なしで code に固定すると後で修正範囲が大きい | docs で暫定上限と監査分類を採択した後 |

## 保留

| 候補 | 保留理由 | 次のアクション |
| --- | --- | --- |
| B. MVP 金額上限・残高上限 docs | API 接続前に必要だが、今回は口座状態 helper を優先する | 次 cycle で `docs/mvp.md` / `docs/security-notes.md` / `docs/test-strategy.md` の docs+必要なら domain 上限へ進む |
| C. domain error mapping / audit `failure_reason` | 監査ログ実装前に必要だが、今回は pure domain code の追加を優先する | 口座状態 error も含めて mapping 表を作る |
| 行ロック・DB transaction 方針 | 重要だが、DB 実装前の設計 scope として別 cycle が適切 | `docs/design-principles.md` / `docs/data-model.md` に lock 順序と transaction 境界を書く |
| 監査ログ属性正規化 | security 優先度は高いが、監査ログ persistence 前の別 scope | `ip_address`、`user_agent`、request body hash、制御文字、最大長を定義する |

## accepted scope

### 目的

- 口座状態が「有効」のときだけ入金・出金・振込などの残高変更系操作に進める、Go domain 層の最小 helper を追加する。
- 停止中・解約済み口座の残高変更拒否を、後続 service / handler / repository が共通利用できる sentinel error と unit test で固定する。
- 認証・認可・DB・監査ログ・冪等性には踏み込まず、既存 money domain helper と同じく pure domain code と README 更新に限定する。

### 対象ファイル/領域

- `internal/domain/`
  - 推奨: `account.go` を追加し、口座ステータスと validation helper を実装する。
  - 既存 `money.go` の package と同じ `domain` package に置く。
- `internal/domain/*_test.go`
  - 推奨: `account_test.go` を追加する。
- `README.md`
  - 現在の実装範囲に、口座ステータス validation の domain 土台が追加されたことを反映する。
  - 未実装機能リストと矛盾しないよう、口座作成 API / DB / 認証 / 監査ログは引き続き未実装と明記する。
- `docs/ai/cycles/2026-07-01-001/implementer.md`
  - implementer は、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

### 実装対象

1. 口座ステータス型を追加する。
   - `AccountStatus` のような string-based type でよい。
   - MVP の有効値は `active`、`suspended`、`closed` とする。
   - docs 上の「有効」「停止中」「解約済み」に対応する。
2. validation を追加する。
   - `AccountStatus.Validate() error` または同等の helper を追加する。
   - `active`、`suspended`、`closed` は valid とする。
   - 空文字や未知の status は `ErrInvalidAccountStatus` のような sentinel error で拒否する。
3. 残高変更可否 helper を追加する。
   - `EnsureAccountCanTransact(status AccountStatus) error` または同等の命名でよい。
   - `active` のみ nil を返す。
   - `suspended` と `closed` は `ErrAccountNotActive` のような sentinel error で拒否する。
   - unknown status は status validation error を返し、停止/解約済みとは区別する。
4. unit test を追加する。
   - `active` / `suspended` / `closed` の status validation が成功する。
   - 空文字・未知 status の validation が `ErrInvalidAccountStatus` で失敗する。
   - `active` は残高変更可として成功する。
   - `suspended` と `closed` は残高変更不可として `ErrAccountNotActive` で失敗する。
   - unknown status は `ErrInvalidAccountStatus` で失敗し、`ErrAccountNotActive` と混同しない。
5. README を更新する。
   - 現在の実装範囲に、金額・残高 helper に加えて口座ステータス validation helper があることを書く。
   - 業務 API、DB 接続、認証、認可、監査ログ、冪等性キー処理は未実装のままと明記する。
6. `docs/ai/cycles/2026-07-01-001/implementer.md` を作成する。

### 実装しないこと

- HTTP route、handler、request / response schema は追加しない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装しない。
- Account aggregate、Account ID、account number 採番、customer ID、owner relation、残高 field を持つ永続 entity は作らない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、行ロックは作らない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC は実装しない。
- 監査ログ永続化、audit `failure_reason` mapping、失敗監査ログの独立 transaction は実装しない。
- 冪等性キー、request body hash、transfer request 状態遷移は実装しない。
- 金額上限、残高上限、日次上限は今回の code に固定しない。
- reversal / 取消、多通貨、利息、手数料、外部銀行連携は実装しない。

### テスト方針

- 追加・変更した Go ファイルに `gofmt` を適用する。
- `go test ./...` を実行し、既存 server / router / money tests と新規 account status tests がすべて成功することを確認する。
- `git diff --name-only` で、変更が accepted scope 内の `internal/domain`、`README.md`、`docs/ai/cycles/2026-07-01-001/implementer.md` に限定されていることを確認する。
- 必要に応じて `rg -n "AccountStatus|ErrAccount" internal/domain README.md` で README と domain 実装の用語が揃っていることを確認する。

### 作業仮定

- MVP の口座状態の内部表現は `active` / `suspended` / `closed` とする。これは docs の有効 / 停止中 / 解約済みに対応する実装上の英語名であり、API schema の最終決定ではない。
- 残高変更系操作とは、入金・出金・振込のように口座残高へ影響する操作を指す。残高照会や取引履歴照会に同じ helper を使うかは今回決めない。
- `suspended` と `closed` はどちらも残高変更不可として同じ sentinel error でよい。利用者向けメッセージや監査分類で分けるかは次 cycle の error mapping / audit design で扱う。
- unknown status はデータ破損または mapper 不備に近い扱いとして、業務上の停止口座とは別 error にする。
- README は実装で現行実装範囲が変わるため implementer が更新する。planner は README を変更しない。

### レビューで重点確認してほしい観点

- docs の口座ステータス「有効・停止中・解約済み」と code の `active` / `suspended` / `closed` が対応しているか。
- `active` だけが残高変更可で、`suspended` / `closed` が拒否されるか。
- unknown status が `ErrAccountNotActive` ではなく invalid status として区別されるか。
- 口座状態 helper が認証・認可・DB・監査ログ・HTTP に依存していないか。
- README の現行実装範囲と未実装一覧が矛盾していないか。

## 実装しないこと

- この planner 作業では、`docs/ai/cycles/2026-07-01-001/planner.md` 以外を書き換えない。
- Go ソースコード、テストコード、README、通常 docs は変更しない。
- 親 agent として実装代行しない。implementer はこの accepted scope だけを実装する。
- 他 agent と直接同期しない。既存 cycle 成果物は repo 上の入力 artifact としてのみ扱う。
- 金融仕様の最終決定は行わず、既存 docs と human notes に沿う範囲を作業仮定として実装可能な小 scope に落とす。

## 作業仮定

- cycle id はユーザー指定どおり `2026-07-01-001` とする。ユーザーの「新規だから7/1かな？」について、現在日付が 2026-07-01 であり、既存 cycle に 2026-07-01 のものがないため `2026-07-01-001` を新規 cycle id として使う。
- planner の書き込みは `docs/ai/cycles/2026-07-01-001/planner.md` のみに限定する。
- この repo は学習用ミニバンキングシステムであり、本番金融システム相当の完成度をこの cycle で主張しない。
- Go 標準ライブラリのみの方針は維持する。外部ライブラリや DB driver は今回追加しない。
- 次の implementer は、accepted scope にない判断を増やさず、不明点は `implementer.md` の作業仮定として記録する。
