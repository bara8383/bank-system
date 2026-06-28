# planner: 2026-06-28-001

## repo現状

- `git status --short`: 作業開始時点では出力なし。既存の未コミット変更は確認されなかった。
- `.codex/agents/README.md`: planner は repo 現状、reviewer 出力、未実装領域から改善案を作り、同一 cycle の `planner.md` に accepted scope を残す。implementer は同一 cycle の accepted scope だけを実装する。
- `docs/ai/cycles/README.md`: 成果物は `docs/ai/cycles/<cycle-id>/` 配下の Markdown で同期し、agent 同士は直接同期しない。
- `AGENTS.md`: 学習用の銀行・金融システム。Go + REST + PostgreSQL を前提に、小さく実装し、設計判断は docs に記録する。
- `README.md`: 現在の実装範囲は Go 標準ライブラリのみの最小 REST API server と `GET /healthz`。DB、認証、業務 API、監査ログ、冪等性キー処理は未実装。
- `docs/START_HERE.md` / `docs/mvp.md`: MVP は顧客登録、口座作成、入金、出金、振込、残高・取引履歴照会、監査ログを安全に扱うことを目指す。
- `docs/domain-model.md` / `docs/data-model.md`: 顧客、ユーザー、口座、残高、取引、振込依頼、監査ログ、`users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs` の初期案がある。
- `docs/design-principles.md`: 金額整数、残高非負、取引履歴追記、監査ログ、認証認可、振込原子性、冪等性、状態遷移を原則としている。
- `docs/use-cases.md`: 入金、出金、振込では、口座状態、権限、正の整数金額、残高不足、冪等性、ロールバックが重要。
- `docs/security-notes.md`: 認証、認可、監査、秘密情報、個人情報、入力検証の基本方針がある。
- `docs/test-strategy.md`: 金額計算、残高更新、取引履歴、監査ログ、異常系、振込原子性、冪等性、認証認可を重点テスト対象としている。
- human notes: `docs/ai/output/human/` は存在せず、追加の人間メモは確認できなかった。
- 既存コード: `go.mod`、`cmd/server/main.go`、`cmd/server/main_test.go`、`internal/httpapi/router.go`、`internal/httpapi/router_test.go` がある。`/healthz` は固定 JSON のみを返し、server は `127.0.0.1:8080` 既定、`BANK_SYSTEM_HTTP_ADDR` override、HTTP timeout を持つ。
- TODO/FIXME: 明示的な `TODO` / `FIXME` は見つからない。未定義事項は docs と cycle reviewer 出力に記録されている。

### 実装済み

- 最小 Go module と HTTP server。
- `GET /healthz`、unsupported method の `405`、固定レスポンスの unit test。
- server listen address と timeout の unit test。
- README の現状、起動方法、テスト方法、未実装範囲。

### 設計済みだが未実装

- ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- PostgreSQL schema、migration、repository、DB transaction。
- 認証方式、RBAC、監査ログ方式、冪等性キー処理。

### 未設計または具体化不足

- `transaction_type` ごとの残高増減方向と `balance_after` の意味。
- 残高更新、取引履歴作成、振込依頼更新、監査ログ記録の DB transaction 境界。
- 並行出金・並行振込時の残高保護方式。
- 冪等性キーの一意スコープ、同一キー異内容時の扱い、保存期間。
- 認証方式、RBAC、入力検証、エラー応答、ログマスキング、監査ログ失敗時境界。

### docs/実装不一致

- README と現行実装は一致している。
- MVP docs は金融業務全体の方針を示すが、実装は health check のみであることが README に明記されている。
- `docs/data-model.md` は `deposit`、`withdrawal`、`transfer_debit`、`transfer_credit`、`reversal` を候補にしているが、残高方向と `balance_after` 検証ルールは未具体化。

## 入力レビュー

- cycle 001 implementer は、当時 accepted scope がないとして blocked を記録していた。現在は後続 cycle により Go skeleton と HTTP hardening が実装済み。
- cycle 002 reviewer 群は、最小 Go REST API skeleton に修正必須 finding はなく、次は server hardening、元帳・残高方向、認証/RBAC、DB transaction 方針を小さく扱うことを推奨した。
- cycle 003 implementer は、listen address 既定値、環境変数 override、HTTP timeout、README、unit test を実装済み。reviewer 群は修正必須 finding なしとした。
- cycle 004 implementer は、同一 cycle の accepted scope 不在として blocked を記録した。
- cycle 004 code-reviewer は、次に DB transaction 境界、認証認可、エラー分類、監査ログ境界のいずれかを小さく scope 化することを推奨した。
- cycle 004 security-reviewer は、業務 API 前に認証/RBAC、入力検証・エラー応答・ログマスキング、監査ログ境界を docs 化することを推奨した。
- cycle 004 banking-reviewer は、次に `transaction_type` ごとの残高方向、`balance_after`、追記型履歴、冪等性、並行更新、監査ログ境界を小さく設計文書化することを推奨した。
- 不確実性: cycle 004 planner には docs-only accepted scope が記録されている一方、cycle 004 implementer は planner 不在として blocked している。並列実行タイミング差による不整合として扱い、今回の指定 cycle では現在の repo 状態を基準に accepted scope を再提示する。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. 元帳・残高方向・成功時 transaction 境界を docs に具体化する | `docs/design-principles.md` は残高非負、取引履歴、原子性を定義済み。`docs/data-model.md` は `transactions` と `balance_after` を持つ。banking-reviewer が継続して高優先としている。 | `transaction_type` の増減方向、`balance_after` の意味、成功時に残高更新と取引履歴作成を同一 DB transaction に入れるルールが未具体化。 | 入金・出金・振込や DB schema に進む前に、金融事故リスクの核となる整合性ルールを共有できる。 | banking-reviewer: 元帳整合性。code-reviewer: transaction 境界。security-reviewer: 監査ログ未確定事項の分離。 | `reversal`、並行更新方式、冪等性スコープ、失敗時監査ログは人間確認事項として残す。 |
| B. MVP 認証方式と RBAC / 水平権限チェック方針を docs 化する | `docs/mvp.md` と `docs/design-principles.md` は認証認可を必須としている。security-reviewer が High としている。 | Cookie session / Bearer token、パスワードハッシュ、CSRF、ロール権限表が未定義。 | 業務 API は認証認可なしに安全に追加できない。 | security-reviewer: 水平権限不備、認証強度、権限分離。 | 安全上重要な仕様であり、人間確認なしに最終確定しない。 |
| C. REST API 入力検証・エラー応答・ログマスキング方針を docs 化する | `docs/security-notes.md` と `docs/test-strategy.md` は入力検証と秘密情報ログ禁止を示す。 | request body size limit、エラー JSON、検索上限、ログ禁止項目が未定義。 | 業務 API 追加時の情報漏えいとテストばらつきを防ぐ。 | security-reviewer: 情報露出。code-reviewer: error mapping。banking-reviewer: 失敗時証跡。 | 監査ログ境界と重なるため、失敗監査ログの最終方針は分離する。 |
| D. PostgreSQL migration 方針と最小 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。 | migration ツール、ID 型、制約、index、DB 起動方法、冪等性 scope、残高競合方式が未定義。 | DB 側で残高非負、金額正数、冪等性を守る土台になる。 | code-reviewer: migration と repository。banking-reviewer: 残高・元帳。security-reviewer: 個人情報と secret。 | schema は後戻りしにくく、今回の設計整理後に扱う。 |
| E. health / readiness / metrics の公開範囲を docs 化する | `BANK_SYSTEM_HTTP_ADDR` により外部 interface 待ち受けが可能。security-reviewer が将来 readiness の情報露出に注意している。 | 公開可能 endpoint、詳細 readiness、metrics の公開範囲が未定義。 | 業務 API / DB readiness 追加時の情報露出を抑える。 | security-reviewer: 公開範囲。banking-reviewer: 金融ドメイン情報非公開。 | 現在は `/healthz` 固定応答のみで緊急度は低い。 |

## 採択

### 採択: A. 元帳・残高方向・成功時 transaction 境界を docs に具体化する

- 理由: Go skeleton と HTTP hardening は完了しており、次に業務 API や DB schema へ進む前の最大リスクは、残高更新と取引履歴の整合性ルールが実装可能な粒度で共有されていないこと。
- banking-reviewer 入力への対応: `transaction_type` ごとの残高増減方向、`balance_after`、追記型履歴、成功時 transaction 境界を明示する。
- code-reviewer 入力への対応: 実装ではなく設計文書の更新に限定し、DB schema や migration ツールは確定しない。
- security-reviewer 入力への対応: 監査ログの重要性は維持しつつ、失敗時監査ログ、監査ログ書き込み失敗時の扱い、マスキング詳細は人間確認事項として分離する。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| D. PostgreSQL migration 方針と最小 schema を作る | DB schema は後戻りしにくく、元帳方向、冪等性スコープ、監査境界、認証方式、migration ツールが未確定。今回の accepted scope はその前提となる docs 整理に限定する。 | 元帳・残高方向・成功時 transaction 境界が docs に反映され、人間確認事項が減った後。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| B. MVP 認証方式と RBAC / 水平権限チェック方針を docs 化する | 業務 API 前に必須だが、今回の cycle では金融事故リスクの核である元帳・残高方向を先に扱う。 | Cookie session か Bearer token か。CSRF、ログアウト、パスワードハッシュ、管理者作成方法、operator を含めるか。 | 次 cycle 以降で security design scope として採択する。 |
| C. REST API 入力検証・エラー応答・ログマスキング方針を docs 化する | 業務 API 前に必要だが、現在の API は `/healthz` の固定応答のみ。 | request id、検索上限、ログに含める actor / IP / User-Agent、マスキング対象、失敗監査ログ境界。 | 認証または業務 API の前に docs scope として採択する。 |
| E. health / readiness / metrics の公開範囲を docs 化する | 現在は `/healthz` の固定応答のみで、DB readiness や metrics は未実装。 | 詳細 readiness を公開するか、内部 network 限定または認証必須にするか。metrics を MVP に含めるか。 | DB 接続や readiness endpoint 追加前に採択する。 |

## accepted scope

### 目的

- 入金・出金・振込・DB schema 実装へ進む前に、残高変更と取引履歴の最小整合性ルールを設計文書へ反映する。
- `transaction_type` ごとの残高増減方向と、`transactions.balance_after` の意味を明確にする。
- 成功した残高変更では、口座残高更新と取引履歴作成を同一 DB transaction に入れる方針を明確にする。
- 後戻りしにくい金融仕様は最終決定せず、人間確認事項として分離する。

### 対象ファイル/領域

- `docs/design-principles.md`
  - 残高変更と取引履歴の同一 DB transaction 方針を追記する。
  - 成功時に確定する最小ルールと、失敗時・監査ログ関連の未確定事項を分ける。
- `docs/data-model.md`
  - `transactions.transaction_type` ごとの残高増減方向表を追記する。
  - `transactions.balance_after` の意味と制約案を追記する。
  - `reversal` は候補として残しつつ、MVP 初期での扱いは未確定と明記する。
- `docs/test-strategy.md`
  - 将来の入金・出金・振込テストで、残高変更と取引履歴作成の同一 transaction、`balance_after`、残高非負、失敗時に残高と取引履歴が変わらないことを確認する方針を追記する。
- `docs/ai/cycles/2026-06-28-001/implementer.md`
  - 実装結果、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

### 実装対象

1. `docs/design-principles.md` に、残高変更成功時の最小 transaction 方針を追記する。
   - 入金成功時は、対象口座の残高増加と入金取引履歴作成を同一 DB transaction に含める。
   - 出金成功時は、対象口座の残高減少と出金取引履歴作成を同一 DB transaction に含める。
   - 振込成功時は、振込元残高減少、振込先残高増加、振込出金取引履歴、振込入金取引履歴、振込依頼の成功状態更新を同一 DB transaction に含める。
   - 金額は正の整数、通貨は MVP では JPY、残高は 0 未満にしない。
   - 残高変更に成功したのに取引履歴がない状態、または取引履歴だけがあり残高が変わらない状態を禁止する。
   - 失敗時監査ログ、監査ログ書き込み失敗時の業務処理、並行更新制御方式は未確定として人間確認事項に残す。
2. `docs/data-model.md` に、`transactions.transaction_type` の残高増減方向表を追記する。
   - `deposit`: 対象口座の残高を増やす。
   - `withdrawal`: 対象口座の残高を減らす。
   - `transfer_debit`: 振込元口座の残高を減らす。
   - `transfer_credit`: 振込先口座の残高を増やす。
   - `reversal`: 取消・組戻しの設計が未確定のため、MVP 初期では方向と利用条件を確定しない。
3. `docs/data-model.md` に、`balance_after` の意味を追記する。
   - `balance_after` は、その取引を対象口座へ適用した直後の口座残高を表す。
   - `balance_after` は 0 以上である。
   - 1 件の振込では、振込元の `transfer_debit` と振込先の `transfer_credit` が別々の `transactions` 行を持ち、それぞれの対象口座に対する `balance_after` を持つ。
   - 履歴は追記型を基本とし、誤り訂正や取消は既存行の更新・削除ではなく追加記録で表現する方針を維持する。ただし `reversal` の詳細は未確定として残す。
4. `docs/test-strategy.md` に、将来のテスト観点を追記する。
   - 入金成功時に残高と `deposit` 取引履歴が同時に作られ、`balance_after` が更新後残高と一致すること。
   - 出金成功時に残高と `withdrawal` 取引履歴が同時に作られ、残高不足時は残高も取引履歴も変わらないこと。
   - 振込成功時に 2 口座の残高、2 件の取引履歴、振込依頼状態が同一 transaction で整合すること。
   - 振込失敗時に片方の残高だけ、または片方の取引履歴だけが残らないこと。
5. `docs/ai/cycles/2026-06-28-001/implementer.md` に、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

### 実装しないこと

- Go ソースコード、HTTP handler、server 設定、DB 接続コードは変更しない。
- PostgreSQL migration、DB schema、SQL、repository、transaction manager は作らない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログの業務 API は実装しない。
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウトは実装・確定しない。
- `reversal`、取消、組戻しの詳細仕様は確定しない。
- 並行出金・並行振込の制御方式として、行ロック、条件付き UPDATE、その他方式のどれを使うかは確定しない。
- 冪等性キーの一意スコープ、同一キー異内容時の扱い、保存期間は確定しない。
- 監査ログ書き込み失敗時に業務処理を失敗させるか、補償するかは確定しない。
- `README.md` は、今回の docs-only scope では変更しない。実装済み機能に変化がないため。
- cycle 002 / 003 / 004 の成果物は編集しない。

### テスト方針

- docs-only 変更のため、実行必須の業務テストはない。
- 変更後に `go test ./...` を実行できる環境であれば、既存コードが壊れていないことを確認する。
- この planner 作業では `go test ./...` を試したが、現在の環境では `go` コマンドが見つからず実行できなかった。
- Markdown の表や見出しが既存 docs の構成と矛盾していないか確認する。
- 金額を浮動小数点で扱う記述が混入していないか確認する。
- 保留事項や人間確認事項を、確定済みルールとして書いていないか確認する。

### レビューで重点確認してほしい観点

- banking-reviewer:
  - `transaction_type` の残高増減方向と `balance_after` の説明が、残高非負、取引履歴追記型、振込の二面性と矛盾しないか。
  - 成功時 transaction 境界が、片側だけ成功する振込や取引履歴欠落を防ぐ表現になっているか。
  - `reversal`、並行更新制御、冪等性、監査ログ失敗時扱いを勝手に確定していないか。
- code-reviewer:
  - docs の追記が実装時に参照できる粒度で、かつ DB schema や migration を過度に先取りしていないか。
  - 将来の repository / transaction manager / test 実装へつながるテスト観点が明確か。
- security-reviewer:
  - 監査ログや失敗時ログの未確定事項が、人間確認事項として分離されているか。
  - エラー詳細、個人情報、秘密情報、内部情報のログ出力方針を今回 scope で不用意に確定していないか。

## 実装しないこと

- planner として、ソースコード、設計文書、DB schema、認証方式、金融仕様の実装・最終決定は行わない。
- 本ファイル `docs/ai/cycles/2026-06-28-001/planner.md` 以外へ書き込まない。
- cycle 002 / 003 / 004 の成果物を修正しない。
- ユーザー作業や他 agent 作業を revert しない。
- 保留事項や人間確認事項を accepted scope に混ぜない。

## 人間確認事項

1. `reversal`、取消、組戻しを MVP 初期の実装対象に含めるか。含めない場合、初期 MVP では「既存取引の訂正・取消は未対応」と明示してよいか。
2. 並行出金・並行振込時の残高保護を、PostgreSQL の行ロック、条件付き UPDATE、または別方式のどれで学習・実装するか。
3. 冪等性キーの一意スコープを、操作種別、送信元口座、ログインユーザー、リクエスト本文 hash のどこまで含めるか。
4. 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するか。
5. 失敗時監査ログを、業務 transaction の rollback と独立して残す必要があるか。
6. 業務 API 追加前に、認証方式を Cookie session + CSRF と Bearer token のどちらで検討するか。
7. PostgreSQL migration ツール、ID 型、口座番号採番方式をいつ確定するか。
