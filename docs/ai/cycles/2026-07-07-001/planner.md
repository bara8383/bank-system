# planner: 2026-07-07-001

## repo現状

- 作業開始時に `git status --short` を確認した。表示は空で、既存の未コミット変更は確認しなかった。
- 指示・運用ルール:
  - root `AGENTS.md` は、実装前の既存コード確認、重要な設計判断の `docs/` 記録、README 最新化、不明点を推測で決めず設計案として明示することを求めている。
  - `.codex/agents/README.md` と `docs/ai/cycles/README.md` は、cycle を `planner -> implementer -> reviewers` の順に進め、agent 間同期は `docs/ai/cycles/<cycle-id>/` 配下の Markdown 成果物だけで行うことを定めている。
  - planner は実装不可であり、本 turn の書き込みは `docs/ai/cycles/2026-07-07-001/planner.md` のみに限定する。
- 実装済み:
  - Go module は `module bank-system`、Go version は `1.24`。
  - `cmd/server` は標準ライブラリの HTTP server を起動し、既定 listen address は `127.0.0.1:8080`、`BANK_SYSTEM_HTTP_ADDR` で上書き可能。
  - `internal/httpapi` は `GET /healthz` のみを提供し、固定 JSON `{"status":"ok"}` を返す。unsupported method は `405 Method Not Allowed` と `Allow: GET` を返す。
  - `internal/domain` には次の pure domain helper がある。
    - `Amount` / `Balance`: JPY の整数最小通貨単位を `int64` で扱い、正の取引金額、非負残高、残高加算、残高減算、残高不足、overflow を検証する。
    - `AccountStatus`: `active` / `suspended` / `closed` を検証し、残高変更可能なのは `active` のみとする `EnsureAccountCanTransact` を提供する。
    - `TransactionType`: `deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` を検証し、`ApplyTransaction` で取引種別ごとの残高増減を計算する。`reversal` は valid type に含めていない。
    - `FailureReason`: 既存 domain sentinel error を `invalid_amount`、`invalid_balance_state`、`insufficient_balance`、`balance_overflow`、`invalid_account_status`、`account_not_active`、`invalid_transaction_type` へ写像する。未知 error と `nil` は `"", false` で未分類扱いにする。
  - unit test は server config、router、money、account status、transaction type / balance application、failure reason mapping を確認している。
- 設計済みだが未実装:
  - 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会。
  - PostgreSQL 接続、DB schema、migration、repository、transaction manager、行ロック、DB transaction 処理。
  - 取引履歴永続化、`transactions.balance_after` の保存、`accounts.balance_amount` との同一 transaction 整合。
  - 認証、認可、Cookie session、CSRF token、password hash、RBAC、admin 代行処理。
  - HTTP error response / status code mapping、public API error code / message。
  - 監査ログ table、audit repository、成功 / 失敗監査ログ永続化、失敗監査ログの独立 transaction。
  - 冪等性キー処理、request body hash、transfer request 状態遷移。
  - account lifecycle の詳細状態遷移、残高あり解約可否、未完了振込依頼がある口座の解約可否。
  - 金額上限、残高上限、日次上限。
- docs / 実装の一致:
  - README は現在の実装範囲を healthz と domain helper 群に限定し、業務 API / DB / 認証 / 監査 / 冪等性は未実装としているため、実装との大きな不一致はない。
  - `docs/design-principles.md`、`docs/data-model.md`、`docs/security-notes.md`、`docs/test-strategy.md` は、残高非負、整数金額、取引履歴、監査ログ、冪等性、認証認可、PostgreSQL transaction を要求しているが、現状では将来実装方針として記録されている。
  - `docs/security-notes.md` は、domain error 由来の `failure_reason` に `internal/domain` の safe failure category helper が返す固定分類値を使い、raw request body、password、token、secret、CSRF token、session ID、未加工の自由入力値を保存・返却しない方針を記録している。
- TODO / FIXME:
  - 実装コード上の明示的な `TODO` / `FIXME` は確認できなかった。未実装事項は README、docs、cycle artifact に記録されている。
- 現 cycle 成果物:
  - `docs/ai/cycles/2026-07-07-001/` は本 planner 作成時点で既存成果物なし。planner が最初に `planner.md` を作成する。

## 入力レビュー

- human notes:
  - `001-human-review.md` は、MVP では `reversal` を含めず通常取引を先に明確化すること、PostgreSQL は学習目的で行ロックによる悲観ロックを採ること、冪等性キーには操作種別・送信元口座・ログインユーザー・request body hash を含めること、MVP の重複時挙動は既存結果返却ではなく拒否でよいこと、監査ログは失敗時も独立して残すこと、認証は Cookie + CSRF token 方向、`operator` は MVP 対象外で `admin` は代行可能という判断を示している。
  - `002-human-review.md` は、直近レビューで大きな問題がなく、小さい実装の粒度をもう少し大きくしてもよさそうという示唆を出している。
- 直近 cycle `2026-07-06-001`:
  - implementer は `FailureReason` helper、unit test、README、security notes を追加・更新した。
  - code-reviewer は blocking なしとしつつ、複数 sentinel error を含む joined / wrapped error の優先順位が未設計であること、`FailureReason` をどの境界で `Validate()` するか、public API message / HTTP status / audit `failure_reason` / internal log を分けることを次入力にした。
  - security-reviewer は blocking なしとしつつ、public API と audit / structured log の分類粒度を分離確認すること、unknown error fallback で raw `err.Error()` を保存・返却しない rule、入出金・振込 service 前の security gate 順序を docs 化することを次入力にした。
  - banking-reviewer は blocking なしとしつつ、`FailureReason` の利用境界、unknown error の扱い、冪等性キー範囲、PostgreSQL 行ロック順序、口座 lifecycle、金額上限等を次入力にした。
- 過去 cycle から継続する重要入力:
  - `reversal` は MVP 初期 code では valid type に含めない方針が確定している。
  - DB / API / service へ進む前に、domain helper を金融整合性全体と誤認しない説明を README / docs へ維持する必要がある。
  - 業務 API skeleton は、認証、認可、DB transaction、監査ログ、冪等性が未実装のまま増やすと危険なため、これまでは却下または保留されてきた。
  - 一方で human note は粒度拡大を許容しており、次 scope は docs-only ではなく小さな code-changing scope を含めるべきである。
- repo-local skill `banking-planning` の観点:
  - 実装済み / 未実装 / docs 不一致 / reviewer 未反映を分け、accepted scope は implementer が追加判断せず実装できる粒度にする。
  - 未確定事項は作業仮定として分離し、code-changing scope を少なくとも 1 つ採択する。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. unknown error 用の safe failure category と audit fallback helper を追加する | 直近 reviewer が unknown error の扱いと raw `err.Error()` 保存・返却禁止を次入力にした。既存 `FailureReasonFromError` は未知 error を `"", false` にするのみ。 | caller が `ok == false` を誤って raw error message へ fallback する余地が残る。public API / audit の境界説明も未整理。 | 監査ログ永続化・HTTP error response 前に、安全な失敗分類の最終防衛線を強められる。pure domain code + docs で小さく実装可能。 | security: raw error / secret 流出防止。code: API と audit の責務分離。banking: 失敗監査ログ分類の土台。 | public API response や status code は確定しない。未知 error の詳細は分類値に入れない。既存 `FailureReasonFromError` の semantics を壊さない。 |
| B. 入出金・振込 service gate 順序を docs 化する | 直近 reviewer 全員が、認証、CSRF、認可、口座状態、金額、冪等性、行ロック、残高計算、取引履歴、監査ログの順序固定を推奨。 | service / handler 実装前の処理順序が docs にまとまっていない。 | 業務 API 前の事故予防に直結する。 | security: horizontal auth と CSRF。banking: transaction 境界。code: service 責務。 | docs-only になりやすいため、今回単独採択では code-changing 要件を満たさない。 |
| C. 冪等性キー scope の domain helper を追加する | human note は操作種別・送信元口座・ログインユーザー・request body hash を含める方針、MVP では重複時拒否を示した。 | transfer request / DB 一意制約 / request body hash 方式が未実装。 | 二重入金・二重出金・二重送金防止の中核。 | banking: 二重実行防止。security: request body hash と actor。code: DB constraint との整合。 | ID 型、hash アルゴリズム、保存 TTL、DB schema が絡み、今回の小 scope には大きい。 |
| D. account lifecycle transition helper を追加する | banking-reviewer は状態遷移表、残高あり解約、未完了振込依頼がある口座の解約可否を次入力にした。既存 code は状態 validation と transact 可否のみ。 | `active` / `suspended` / `closed` の遷移可否が code / docs にない。 | 口座管理 API 前に closed 復活などを防ぐ。 | banking: closed 復活禁止。security: admin 代行監査。code: pure domain helper。 | 残高 0 条件、未完了振込依頼、admin 権限、監査が絡むため、実装するなら docs と併せて大きめ scope。 |
| E. PostgreSQL schema / migration skeleton に進む | data model は table 候補と constraints を持つ。domain helper が増えてきた。 | DB 接続、migration tool、repository、transaction manager がない。 | MVP 業務 API へ進む大きな前進。 | banking: constraints / row lock。security: SQL injection / secrets。code: migration 方針。 | driver / migration tool / lock 順序 / audit / idempotency が絡み、現時点では scope が大きい。 |
| F. 業務 API skeleton を追加する | README 未実装一覧の中心機能。 | healthz 以外の endpoint がない。 | 見える進捗は大きい。 | security: 未認証 endpoint 化の危険。banking: DB transaction なしの残高操作危険。 | 認証、認可、DB transaction、監査、冪等性なしの endpoint は引き続き premature。 |

## 採択

### A. unknown error 用の safe failure category と audit fallback helper を追加する

- 採択理由:
  - 直近 cycle で追加された `FailureReasonFromError` は既知 domain error の分類として妥当だが、unknown error 時に caller が raw `err.Error()` を代替保存する事故余地が reviewer から明示された。
  - 監査ログ永続化や HTTP error response へ進む前に、unknown error を安全な固定分類へ寄せる helper と docs を用意すれば、後続実装が raw DB error、秘密情報、自由入力値を `failure_reason` へ入れるリスクを下げられる。
  - pure domain helper と unit test、README / security docs 更新に限定でき、HTTP / DB / auth / audit persistence に踏み込まないため、現在の repo 段階に合う。
  - code-changing scope を満たしつつ、直近 reviewer の「public API と audit の境界」「unknown error fallback」入力へ直接応答できる。
- 採択の範囲:
  - `FailureReason` に unknown / internal error 用の safe category を追加する。
  - 既知 domain error は従来どおり分類し、未知 non-nil error は raw message ではなく固定 category に寄せる helper を追加する。
  - `nil` を failure として分類しない挙動は維持する。
  - README / security docs で、audit / safe log 用の fallback であり public API response の最終仕様ではないことを明記する。

### B. 入出金・振込 service gate 順序を docs に短く接続する（A の補助範囲）

- 採択理由:
  - A の helper は audit fallback の部品であり、単独では service gate 順序を保証しない。docs 追記の中で、helper の利用位置を「失敗監査ログ / safe log 分類」に限定して明示することで、後続 service 実装時の誤用を防ぐ。
  - ただし今回の主目的は A の code-changing scope であり、包括的な service 設計書や全 gate 順序表を完成させることはしない。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| E. PostgreSQL schema / migration skeleton | migration tool、driver、transaction manager、行ロック順序、監査ログ保存境界、冪等性一意制約まで絡み、今回の safe failure category 強化と同時に進めるには大きい。 | failure category fallback、service gate 順序、idempotency scope、row lock 順序が docs 化された後。 |
| F. 業務 API skeleton | 認証、認可、DB transaction、監査ログ、冪等性が未実装のまま endpoint を増やすと、学習用でも危険な半端な業務 API になる。 | 最小の auth / authorization / DB transaction / audit / idempotency 方針が accepted scope 化され、実装土台が揃った後。 |

## 保留

| 候補 | 保留理由 | 次に扱う条件 |
| --- | --- | --- |
| C. 冪等性キー scope の domain helper | human note の方針は具体的だが、request body hash 算出、ID 型、DB 一意制約、transfer request status、重複時 failure reason と一体で扱う必要がある。 | 次 cycle 以降で「操作種別、送信元口座、ログインユーザー、request body hash、MVP では重複時拒否」を docs + domain helper または DB schema scope として採択する。 |
| D. account lifecycle transition helper | 重要だが、残高 0 条件、未完了振込依頼、admin 権限、監査ログを同時に整理しないと不完全な helper になりやすい。 | 口座作成 / 停止 / 解約 API の前、または account lifecycle docs scope と合わせて採択する。 |
| service gate 順序の完全 docs | reviewer 入力として高優先だが、今回は unknown error fallback を先に code 化する。 | 次 cycle で docs-only ではなく、service skeleton または domain / adapter helper と組み合わせて採択する。 |
| public API error response schema | `FailureReason` と関連するが、HTTP status、利用者向け message、認証失敗時の扱い、i18n が絡む。 | handler / API を追加する直前に、audit reason とは別の public error contract として設計する。 |
| 金額上限・残高上限・日次上限 | 値の業務判断が強く、現時点で固定すると学習用前提でも後戻りが大きい。 | 業務 API 接続前に、学習用の暫定上限を作業仮定として置くかを planner で判断する。 |

## accepted scope

### 目的

- 既存 `FailureReasonFromError` の「既知 domain error のみ分類し、未知 error は `"", false`」という用途を維持しつつ、監査ログ / safe structured log 用に未知 non-nil error を raw message ではなく固定の safe category へ寄せる helper を追加する。
- 後続の audit writer / service / handler が unknown error 時に `err.Error()`、raw request body、password、token、secret、CSRF token、session ID、未加工の自由入力値を `failure_reason` として保存・返却する事故を防ぐ土台を作る。
- public API response と audit `failure_reason` の責務差を README / security docs に明記し、今回追加する category を「利用者向け message の最終仕様」と誤読させない。

### 対象

- 対象ファイル / 領域:
  - `internal/domain/failure_reason.go`
  - `internal/domain/failure_reason_test.go`
  - `README.md`
  - `docs/security-notes.md` または `docs/design-principles.md`（security notes を優先。必要なら design principles へ短く接続）
  - `docs/ai/cycles/2026-07-07-001/implementer.md`
- 既存実装のうち参照対象:
  - `internal/domain/money.go`
  - `internal/domain/account.go`
  - `internal/domain/transaction.go`

### 実装対象

1. `internal/domain/failure_reason.go` を更新する。
   - `FailureReasonInternalError = "internal_error"` を追加する。
     - 名前は audit / safe log 向けの固定分類として使い、raw DB error、panic detail、自由入力値、秘密情報は含めない。
     - 既存 constants の string 値は変更しない。
   - `FailureReason.Validate()` の allow-list に `FailureReasonInternalError` を追加する。
   - 既存 `FailureReasonFromError(err error) (FailureReason, bool)` は semantics を維持する。
     - 既知 domain sentinel error は従来どおり分類する。
     - `nil`、未知 error、対象外 error は引き続き `"", false` を返す。
   - 監査ログ / safe structured log 用 helper を追加する。
     - 推奨名: `SafeFailureReasonFromError(err error) (FailureReason, bool)`。
     - 挙動:
       - `FailureReasonFromError(err)` が `ok == true` の場合は、その分類と `true` を返す。
       - `err == nil` の場合は `"", false` を返す。
       - non-nil かつ未知 error の場合は `FailureReasonInternalError, true` を返す。
     - コメントで、これは audit / safe structured log の fallback 用であり、public API response body や HTTP status code の最終仕様ではないこと、raw error message を返さないことを明記する。
2. `internal/domain/failure_reason_test.go` を更新する。
   - `FailureReasonValidateAcceptsSupportedReasons` に `internal_error` を追加する。
   - unsafe / unknown rejection test は維持し、raw error string、secret-like value が invalid のままであることを確認する。
   - 既存 `FailureReasonFromError` tests は維持し、未知 error が `"", false` のままであることを確認する。
   - 新規 `SafeFailureReasonFromError` tests を追加する。
     - 既知 domain sentinel error は既存と同じ分類になること。
     - wrapped domain error でも同じ分類になること。
     - `nil` は `"", false` になること。
     - 未知 non-nil error は `FailureReasonInternalError, true` になり、raw `err.Error()` が返らないこと。
     - `FailureReasonInternalError.Validate()` が成功すること。
     - `errors.Join` を使う場合は、既知 domain error を含む joined error が既知分類になることを確認してよい。ただし複数 domain sentinel error の優先順位仕様は今回確定しない。テストするなら「既知 1 件 + unknown 1 件」に限定する。
3. README を更新する。
   - 現在の実装範囲に、unknown non-nil error を audit / safe log 用の `internal_error` 固定分類へ寄せる fallback helper が追加されたことを短く追記する。
   - `FailureReasonFromError` は既知 domain error の分類、`SafeFailureReasonFromError` は未知 error を raw message へ fallback しないための audit / safe log 用 helper、という違いを誤解なく説明する。
   - 未実装機能リストでは、HTTP error response / status code mapping、監査ログ永続化、DB schema、業務 API、認証、認可、冪等性キー処理が未実装であることを維持する。
4. `docs/security-notes.md` を更新する。
   - 監査 `failure_reason` では、既知 domain error は固定分類、未知 non-nil error は raw error message ではなく `internal_error` のような固定分類に寄せる方針を追記する。
   - `internal_error` は調査のための相関 ID や構造化ログ分類と組み合わせる候補であり、DB 接続文字列、SQL、stack trace、request body、password、token、secret、CSRF token、session ID、個人情報、未加工の自由入力値を保存・返却する用途ではないことを明記する。
   - public API response 用の code / message / HTTP status は別途設計し、audit `failure_reason` をそのまま利用者向け message としない方針を明記する。
5. `docs/ai/cycles/2026-07-07-001/implementer.md` を作成する。
   - 参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

### 非対象

- HTTP route、handler、request / response schema、status code mapping、public API error format は追加しない。
- public API 用の利用者向け message、i18n、error code 体系全体は確定しない。
- 監査ログ table、audit repository、audit service、監査ログ永続化、成功 / 失敗監査ログ transaction は実装しない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager、row lock は作らない。
- 認証、認可、Cookie session、CSRF token、password hash、RBAC、admin 代行処理は実装しない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装しない。
- 取引履歴 row、`balance_after` 永続化、transfer request 状態遷移、冪等性キー処理、request body hash 算出は実装しない。
- `reversal` / 取消 / 組戻し / 訂正は実装しない。
- 複数 domain sentinel error を含む joined error の優先順位表は今回確定しない。実装上の `switch` 順に依存する既存挙動を外部仕様として文書化しない。
- `internal_error` を「利用者へ必ず返す API code」として固定しない。今回の固定対象は audit / safe log 用 failure category である。

### テスト方針

- 追加・変更した Go ファイルに `gofmt` を適用する。
- `go test ./...` を実行し、既存 server / router / money / account / transaction tests と更新後 failure reason tests がすべて成功することを確認する。
- `git diff --name-only` で、変更が accepted scope 内の `internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`README.md`、`docs/security-notes.md` または必要最小限の docs、`docs/ai/cycles/2026-07-07-001/implementer.md` に限定されていることを確認する。
- `git diff --check` を実行し、空白エラーがないことを確認する。
- `rg -n "err\.Error\(\)|internal_error|FailureReasonInternalError|SafeFailureReasonFromError|password|token|secret|CSRF|session|request body" internal/domain README.md docs/security-notes.md` などで、unknown error の raw message を failure category として返す実装や、secret / raw request body を保存する説明になっていないことを確認する。

### レビューで重点確認してほしい観点

- `FailureReasonFromError` の既存 semantics が壊れていないか。特に未知 error が `"", false` のままか。
- 新規 `SafeFailureReasonFromError` が unknown non-nil error で raw `err.Error()` を返さず、`internal_error` 固定分類へ寄せるか。
- `nil` を誤って failure reason として分類していないか。
- `internal_error` が audit / safe structured log 向け fallback として説明され、public API response の最終仕様と混同されていないか。
- README / security docs が、監査ログ永続化や HTTP error response 実装済みと誤読されないか。
- helper が HTTP / DB / auth / audit persistence / idempotency / business API に踏み込んでいないか。

## 実装しないこと

- planner として実装代行しない。今回の変更は `docs/ai/cycles/2026-07-07-001/planner.md` の作成のみとする。
- agent 間の直接同期は行わない。implementer / reviewer への入力は、この Markdown 成果物だけに残す。
- accepted scope 外の code、README、docs、DB schema、migration、API、認証、監査ログ永続化、冪等性処理をこの turn で変更しない。
- 本番金融システムとしての完成や安全性を断定しない。学習用ミニバンキングシステムの小さな前進として扱う。

## 作業仮定

- `FailureReasonFromError` は「既知 domain error を分類できるか」を判定する低レベル helper として維持する。
- 新規 helper `SafeFailureReasonFromError` は audit / safe structured log 用の fallback helper とし、public API response body / HTTP status code の最終仕様ではない。
- unknown non-nil error は、監査 `failure_reason` として raw message を保存・返却せず、`internal_error` の固定分類へ寄せる。
- `internal_error` の詳細調査には、将来 correlation ID、安全な構造化ログ、運用ログ保管場所を別途設計して使う。今回の helper は詳細文字列を扱わない。
- `nil` は failure ではないため `"", false` とする。
- `reversal` は引き続き MVP 初期では invalid transaction type として扱う。
- `operator` は MVP 対象外、`admin` は将来の代行操作候補だが、今回 scope では認証・認可を実装しない。
- 冪等性キーは human note に従い、将来「操作種別、送信元口座、ログインユーザー、request body hash」を含む設計にする。ただし今回 scope では実装しない。
- PostgreSQL は将来、学習目的で行ロックによる悲観ロックを採用する方針だが、今回 scope では DB / row lock を実装しない。
