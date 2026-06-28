# planner: 2026-06-28-004

## repo現状

### 作業開始時に確認したもの

- `git status --short`: 作業開始時点では未コミット変更なし。
- `.codex/agents/README.md`: planner は repo 現状、reviewer 出力、未実装領域から改善案を作り、`docs/ai/cycles/<cycle-id>/planner.md` に accepted scope を出す。implementer は同一 cycle の accepted scope だけを実装する。
- `docs/ai/cycles/README.md`: cycle artifact protocol と、planner 出力の必須項目を確認。
- `AGENTS.md`: 学習用の銀行・金融システムであり、Go + REST + PostgreSQL を前提とする。作業ルールは「小さく実装する」「実装前に既存コードを確認する」「設計判断は docs/ に記録する」「README を最新状態に保つ」。
- `README.md`: 現在の実装範囲は、Go 標準ライブラリのみの最小 REST API server と `GET /healthz`。DB、認証、業務 API、監査ログ、冪等性キー処理は未実装。
- `docs/START_HERE.md`: 最初のゴールは、顧客登録、口座作成、入金、出金、振込、残高・取引履歴照会、監査ログを安全に扱えるミニバンキングシステム。
- `docs/mvp.md`: MVP はユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録を含む。
- `docs/domain-model.md`: 顧客、ログインユーザー、口座、残高、取引、振込依頼、監査ログ、認証、認可、トランザクションなどの用語を確認。
- `docs/data-model.md`: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs` の初期テーブル候補と、金額整数、残高非負、冪等性キー一意制約案を確認。
- `docs/use-cases.md`: 入出金・振込では、口座状態、権限、正の整数金額、残高不足、冪等性、ロールバックが重要。
- `docs/design-principles.md`: 残高非負、金額整数、取引履歴、監査ログ、認証認可、原子性、二重実行防止、状態遷移、調査可能なエラーを確認。
- `docs/security-notes.md`: 認証、認可、監査、秘密情報、個人情報、入力検証、今後の対策を確認。
- `docs/test-strategy.md`: 金額計算、残高更新、取引履歴、監査ログ、振込原子性、冪等性、認証認可を重点確認する方針。
- `docs/ai/output/README.md`: human notes は `docs/ai/output/human/` 配下に置く運用を確認。
- `docs/ai/output/human/`: ディレクトリは存在せず、追加の human notes はない。
- 過去 cycle `2026-06-28-001`、`2026-06-28-002`、`2026-06-28-003`: planner、implementer、reviewer 出力を確認。
- 既存コード: `go.mod`、`cmd/server/main.go`、`cmd/server/main_test.go`、`internal/httpapi/router.go`、`internal/httpapi/router_test.go` を確認。
- TODO/FIXME: 明示的な `TODO` / `FIXME` は見つからない。未定義・未実装事項は docs と cycle reviewer 出力に記録されている。

### 実装済み

- Go module は `module bank-system`。
- `cmd/server/main.go` に、標準ライブラリ `net/http` を使う最小 HTTP server がある。
- 既定 listen address は `127.0.0.1:8080`。`BANK_SYSTEM_HTTP_ADDR` で明示的に変更できる。
- `http.Server` には `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` が設定済み。
- `internal/httpapi/router.go` に、`GET /healthz` のみを提供する router / handler がある。
- `/healthz` は固定 JSON `{"status":"ok"}` を返し、`GET` 以外は `405 Method Not Allowed` と `Allow: GET` で拒否する。
- `cmd/server/main_test.go` と `internal/httpapi/router_test.go` に、server 設定、timeout、`/healthz` の成功、method 拒否、不要情報非露出の unit test がある。
- `README.md` に、現状、起動方法、listen address 変更方法、テスト方法、未実装機能、学習用であり本番金融システムではない旨が記録されている。

### 設計済みだが未実装

- MVP 業務機能: ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- 初期データモデル候補: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs`。
- 重要原則: 金額整数、残高非負、残高変更と取引履歴の整合性、監査ログ、認証認可、振込の原子性、冪等性、状態遷移。
- テスト戦略: 金額・残高・取引履歴・監査ログ・振込原子性・冪等性・認証認可を重点確認する方針。

### 未設計または具体化不足

- `transaction_type` ごとの残高増減方向と `balance_after` の意味。
- 入金・出金・振込における、残高更新、取引履歴作成、振込依頼更新、監査ログ記録の transaction 境界。
- 失敗時監査ログを業務 transaction と同じ DB transaction に含めるか、別経路で残すか。
- 並行出金・並行振込時の残高保護方式、ロック順序、デッドロック回避方針。
- 冪等性キーの一意スコープ、リクエスト同一性検証、同一キー異内容時の扱い、保存期間。
- 振込ステータス遷移、失敗した振込依頼の保存境界、処理結果不明時の扱い。
- 認証方式、パスワードハッシュ方式、セッション/トークン、CSRF、ログアウト、レート制限。
- RBAC の権限表、管理者・運用担当者の責務分離。
- PostgreSQL migration ツール、DB 接続方法、transaction manager、repository 境界、ローカル DB 起動方法。
- API 入力検証、検索制限、エラー応答形式、ログ出力規則、データ分類、マスキング規則。

### docs/実装不一致

- `README.md` は現状の実装範囲と一致しており、大きな不一致はない。
- docs は MVP 全体の設計方針を示しているが、実装は health check のみであり、業務機能は未実装として明示されている。
- `docs/data-model.md` には `transaction_type` の候補として `deposit`、`withdrawal`、`transfer_debit`、`transfer_credit`、`reversal` があるが、残高増減方向、`reversal` の扱い、`balance_after` の検証ルールはまだ具体化されていない。

### レビュー未反映

- cycle 003 code-reviewer: HTTP server hardening は accepted scope に適合し、修正必須 finding はない。次 cycle では、元帳・残高方向・transaction 境界・冪等性キー一意スコープ・監査ログ境界、または認証/RBAC 方針を docs に具体化することを推奨。
- cycle 003 security-reviewer: 修正必須の新規 security finding はない。次 cycle では、認証・認可・CSRF/Bearer token・RBAC 方針、API エラー応答・入力検証・ログマスキング、health/readiness 公開範囲、監査ログ境界を候補にすることを推奨。
- cycle 003 banking-reviewer: 今回実装差分は金融ドメイン状態を変更しておらず、直接的な事故リスクは増えていない。次 cycle では、`transaction_type` ごとの残高増減方向、`balance_after` の意味、残高非負制約、残高変更と取引履歴作成の同一 DB transaction 境界を設計文書に具体化することを推奨。

## 入力レビュー

### human notes

- `docs/ai/output/human/` は存在しないため、追加の human notes はない。

### cycle 003 implementer

- HTTP listen address の既定値を `127.0.0.1:8080` に変更し、`BANK_SYSTEM_HTTP_ADDR` による明示的な override を追加済み。
- `http.Server` timeout を設定済み。
- README と unit test を更新済み。
- DB、認証、顧客、口座、入出金、振込、残高、取引履歴、監査ログ、冪等性キー処理は未実装として記録済み。
- 未確認事項として、業務 API 追加前の認証・認可・CSRF または bearer token 方針、元帳・残高方向・冪等性キー一意スコープ・監査ログ書き込み失敗時の扱いが残っている。

### cycle 003 code-reviewer

- 修正必須 finding はない。
- HTTP server hardening は小さく完了しているため、次 cycle では業務 API 前提の設計に戻れる。
- 次候補として、元帳・残高方向・transaction 境界・冪等性キー一意スコープ・監査ログ境界、認証/RBAC、DB transaction manager / repository 境界が挙げられている。

### cycle 003 security-reviewer

- High / Medium 相当の security finding はない。
- `BANK_SYSTEM_HTTP_ADDR` により外部 interface で待ち受けられるため、将来の業務 API 追加時に公開範囲、認証必須 endpoint、health/readiness 公開方針を整理する必要がある。
- 業務 API 実装前に、認証・認可・CSRF/Bearer token・RBAC、入力検証、エラー応答、ログマスキング、監査ログ境界を docs scope として具体化することを推奨。

### cycle 003 banking-reviewer

- HTTP server hardening は元帳・残高・取引履歴上の直接的な事故リスクを増やしていない。
- 次 cycle の高優先候補として、`transaction_type` ごとの残高増減方向、`balance_after` の意味、残高非負制約、残高変更と取引履歴作成の同一 DB transaction 境界を設計文書に具体化することを推奨。
- 業務 API 実装に進む場合は、先に「入金」「出金」「振込」のうち 1 つだけを選び、元帳記録・残高更新・監査ログ・冪等性の最小単位を明示した accepted scope にすることを推奨。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. 元帳・残高方向・成功時 transaction 境界を docs に具体化する | `docs/design-principles.md` は残高非負、取引履歴、原子性を定義済み。`docs/data-model.md` は `transactions` と `balance_after` を持つ。cycle 003 banking-reviewer が最優先候補として推奨。 | `transaction_type` ごとの増減方向、`balance_after` の意味、成功時に残高更新と取引履歴作成を同一 DB transaction に入れるルールが明文化不足。 | 入金・出金・振込や DB schema に進む前に、金融事故リスクの核となる整合性ルールを共有できる。 | banking-reviewer: 元帳整合性。code-reviewer: DB transaction 境界。security-reviewer: 失敗時監査ログ境界の未確定分離。 | `reversal`、並行更新方式、冪等性スコープ、失敗時監査ログ書き込み失敗時の扱いは人間確認事項として残す。 |
| B. 認証・RBAC・CSRF/Bearer token 方針を docs に具体化する | `docs/mvp.md` と `docs/design-principles.md` は認証・認可を必須としている。cycle 003 security-reviewer が高優先候補に挙げている。 | Cookie session / Bearer token、パスワードハッシュ、CSRF、ロール権限表、管理者作成方法が未定義。 | MVP の業務 API は認証認可なしでは安全に追加できない。 | security-reviewer: 水平権限不備、認証強度、秘密情報。code-reviewer: middleware / handler 境界。 | 安全上重要な仕様であり、人間確認なしに最終確定しない。 |
| C. API エラー応答・入力検証・ログ/マスキング方針を docs に具体化する | `docs/security-notes.md` は入力検証と秘密情報ログ禁止を示す。cycle 003 security-reviewer が候補に挙げている。 | エラー JSON 形式、request id、validation error 表現、request body size limit、検索上限、ログ出力項目、機微情報マスキングが未定義。 | 業務 API 追加時の情報漏えいとテストばらつきを防ぐ。 | security-reviewer: 情報露出。code-reviewer: エラー分類とテスト。banking-reviewer: 失敗時証跡。 | 監査ログ境界と重なるため、失敗時監査ログの最終方針は人間確認事項として分離する。 |
| D. PostgreSQL migration 方針と最小 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。code-reviewer は DB 制約の具体化を推奨。 | migration ツール、ID 型、制約、index、DB 起動方法、冪等性 scope、残高競合方式が未定義。 | 残高非負、金額正数、取引履歴、冪等性を DB 側でも守る土台になる。 | code-reviewer: migration と transaction。banking-reviewer: 残高・元帳・冪等性。security-reviewer: 個人情報とシークレット。 | schema は後戻りしにくい。今回の設計整理後に採択する。 |
| E. health / readiness / metrics の公開範囲を docs に具体化する | cycle 003 security-reviewer が `BANK_SYSTEM_HTTP_ADDR` と将来の readiness 公開範囲を注意点として挙げている。 | 公開可能 endpoint、認証必須 endpoint、詳細 readiness、metrics、reverse proxy 前提が未定義。 | 業務 API 追加後の情報露出リスクを抑える。 | security-reviewer: 情報露出と公開範囲。code-reviewer: endpoint 責務。banking-reviewer: 金融ドメイン情報を公開しないこと。 | 現在は `/healthz` のみで急ぎの修正は不要。業務 API や DB readiness 追加前に扱う。 |

## 採択

### 採択: A. 元帳・残高方向・成功時 transaction 境界を docs に具体化する

- 理由: cycle 003 の HTTP server hardening は完了しており、次に業務 API や DB schema へ進む前の最大リスクは、残高更新と取引履歴の整合性ルールが実装可能な粒度で共有されていないこと。ここを先に docs 化することで、入金・出金・振込の accepted scope を小さく安全に切れる。
- banking-reviewer 入力への対応: `transaction_type` ごとの残高増減方向、`balance_after` の意味、残高非負、残高更新と取引履歴作成の同一 DB transaction 境界を明示する。
- code-reviewer 入力への対応: 実装ではなく設計文書の更新に限定し、DB schema や migration ツールは確定しない。将来の repository / transaction manager 実装時に参照できる最小ルールへ落とす。
- security-reviewer 入力への対応: 監査ログが重要であることは維持しつつ、失敗時監査ログや監査ログ書き込み失敗時の扱いは安全上重要な仕様として人間確認事項に残す。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| D. PostgreSQL migration 方針と最小 schema を作る | DB schema は後戻りしにくく、元帳方向、冪等性スコープ、監査境界、認証方式、migration ツールが未確定。今回の accepted scope はその前提となる docs 整理に限定する。 | 元帳・残高方向・成功時 transaction 境界が docs に反映され、人間確認事項が減った後。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| B. 認証・RBAC・CSRF/Bearer token 方針を docs に具体化する | 業務 API 前に必須だが、今回の cycle では banking-reviewer が強く推奨した元帳・残高方向を先に扱う。認証方式は安全上重要で、人間確認なしに最終確定しない。 | Cookie session か Bearer token か。CSRF、ログアウト、パスワードハッシュ、管理者作成方法、運用担当者ロール。 | 次 cycle 以降で security design scope として採択する。 |
| C. API エラー応答・入力検証・ログ/マスキング方針を docs に具体化する | 業務 API 追加前には必要だが、現在の API は `/healthz` の固定応答のみ。監査ログ方針との関係もある。 | 失敗監査ログの保存境界、request id の扱い、ログに含める actor / IP / User-Agent、検索上限、マスキング対象。 | 認証・業務 API または監査ログ設計の前に docs scope として採択する。 |
| E. health / readiness / metrics の公開範囲を docs に具体化する | 現在は `/healthz` の固定応答のみで、DB readiness や metrics は未実装。業務 API や DB 接続を追加する前に扱えばよい。 | 詳細 readiness を公開するか、内部 network 限定または認証必須にするか。metrics を MVP に含めるか。 | DB 接続や readiness endpoint を追加する前の cycle で採択する。 |

## accepted scope

### 目的

- 入金・出金・振込・DB schema 実装へ進む前に、残高変更と取引履歴の最小整合性ルールを設計文書へ反映する。
- `transaction_type` ごとの残高増減方向と、`transactions.balance_after` の意味を明確にする。
- 成功した残高変更では、口座残高更新と取引履歴作成を同一 DB transaction に入れる方針を明確にする。
- 後戻りしにくい金融仕様は最終決定せず、人間確認事項として分離する。

### 対象ファイル/領域

- `docs/design-principles.md`
  - 残高変更と取引履歴の同一 DB transaction 方針を追記する。
  - 成功時と失敗時で確定済みルールと未確定事項を分ける。
- `docs/data-model.md`
  - `transactions.transaction_type` ごとの残高増減方向表を追記する。
  - `transactions.balance_after` の意味と制約案を追記する。
  - `reversal` は候補として残しつつ、MVP 初期での扱いは未確定であることを明記する。
- `docs/test-strategy.md`
  - 将来の入金・出金・振込テストで、残高変更と取引履歴作成の同一 transaction、`balance_after`、残高非負、失敗時に残高と取引履歴が変わらないことを確認する方針を追記する。
- `docs/ai/cycles/2026-06-28-004/implementer.md`
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
5. `docs/ai/cycles/2026-06-28-004/implementer.md` に、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

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
- cycle 001 / 002 / 003 の成果物は編集しない。

### テスト方針

- docs-only 変更のため、実行必須の業務テストはない。
- 変更後に `go test ./...` を実行し、既存コードが壊れていないことを確認する。
- Markdown の表や見出しが既存 docs の構成と矛盾していないか確認する。
- 金額を浮動小数点で扱う記述が混入していないか確認する。
- 保留事項や人間確認事項を、確定済みルールとして書いていないか確認する。

### レビューで重点確認してほしい観点

- banking-reviewer:
  - `transaction_type` の残高増減方向と `balance_after` の説明が、残高非負・取引履歴追記型・振込の二面性と矛盾しないか。
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
- 本ファイル `docs/ai/cycles/2026-06-28-004/planner.md` 以外へ書き込まない。
- cycle 001 / 002 / 003 の成果物を修正しない。
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
