# planner: 2026-06-28-005

## repo現状

### 作業開始時に確認したもの

- `git status --short`: 作業開始時点では未コミット変更なし。
- `.agents/skills/banking-planning/SKILL.md`: planner は repo 現状、human notes、reviewer 出力から改善案を分類し、`docs/ai/cycles/<cycle-id>/planner.md` に accepted scope を出す。実装、ソースコード変更、DB schema 確定、認証方式や金融仕様の最終決定は禁止。
- `.codex/agents/README.md`: planner / implementer / reviewer 群は直接同期せず、cycle 配下の Markdown 成果物で連携する。implementer は同一 cycle の accepted scope のみ実装する。
- `docs/ai/cycles/README.md`: planner 出力の必須項目と artifact protocol を確認。
- `AGENTS.md`: Go + REST + PostgreSQL を中心に、小さく実装し、設計判断は docs に記録し、不明点は推測で決めず設計案として明示する。
- `README.md`: 現在の実装範囲は Go 標準ライブラリのみの最小 REST API server と `GET /healthz`。DB、認証、業務 API、監査ログ、冪等性キー処理は未実装。
- `docs/START_HERE.md`: 最初のゴールは、顧客登録、口座作成、入金、出金、振込、残高・取引履歴照会、監査ログを安全に扱えるミニバンキングシステム。
- `docs/mvp.md`: MVP はユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録を含む。
- `docs/domain-model.md`: 顧客、ログインユーザー、口座、残高、取引、振込依頼、冪等性キー、監査ログ、認証、認可、トランザクションの用語を確認。
- `docs/data-model.md`: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs` の初期テーブル候補、金額整数、残高非負、冪等性キー一意制約案を確認。
- `docs/use-cases.md`: 入金、出金、振込では、口座状態、権限、正の整数金額、残高不足、冪等性、ロールバック、監査ログが重要。
- `docs/design-principles.md`: 残高非負、金額整数、取引履歴、監査ログ、認証認可、原子性、二重実行防止、状態遷移、調査可能なエラーを確認。
- `docs/security-notes.md`: 認証、認可、監査、秘密情報、個人情報、入力検証、ログ漏えい防止を確認。
- `docs/test-strategy.md`: 金額計算、残高更新、取引履歴、監査ログ、振込原子性、冪等性、認証認可を重点確認する方針。
- `docs/ai/output/README.md`: human notes の置き場を確認。`docs/ai/output/human/` は存在せず、追加 human notes はない。
- 過去 cycle `2026-06-28-001` から `2026-06-28-004`: planner、implementer、reviewer 出力を確認。
- 既存コード: `go.mod`、`cmd/server/main.go`、`cmd/server/main_test.go`、`internal/httpapi/router.go`、`internal/httpapi/router_test.go` を確認。
- `go test ./...`: この実行環境では `go` コマンドが見つからず実行不可。cycle 003 implementer では成功、cycle 004 reviewer 群でも同じく実行不可が記録されている。

### 実装済み

- Go module は `module bank-system`。
- `cmd/server/main.go` は標準ライブラリ `net/http` で HTTP server を起動する。
- 既定 listen address は `127.0.0.1:8080`。`BANK_SYSTEM_HTTP_ADDR` で明示的に変更できる。
- `http.Server` には `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` が設定済み。
- `internal/httpapi/router.go` は `GET /healthz` のみを提供する。固定 JSON `{"status":"ok"}` を返し、`GET` 以外は `405 Method Not Allowed` と `Allow: GET` で拒否する。
- server 設定と `/healthz` の unit test が存在する。
- README は、現状、起動方法、listen address 変更方法、テスト方法、未実装機能、学習用であり本番金融システムではない旨を記録している。

### 設計済みだが未実装

- MVP 業務機能: ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- 初期データモデル候補: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs`。
- 重要原則: 金額整数、残高非負、残高変更と取引履歴の整合性、監査ログ、認証認可、振込の原子性、冪等性、状態遷移。
- テスト戦略: 金額、残高、取引履歴、監査ログ、振込原子性、冪等性、認証認可の重点確認。

### 未設計または具体化不足

- `transaction_type` ごとの残高増減方向と `balance_after` の意味。
- 入金、出金、振込における残高更新、取引履歴作成、振込依頼更新、監査ログ記録の DB transaction 境界。
- `reversal`、取消、組戻し、訂正の扱い。
- 冪等性キーの一意スコープ、リクエスト同一性検証、同一キー異内容時の扱い、保存期間。
- 並行出金・並行振込時の残高保護方式、ロック順序、デッドロック回避方針。
- 監査ログの成功・失敗時境界、監査ログ書き込み失敗時の業務処理継続可否、マスキング規則。
- 認証方式、パスワードハッシュ方式、セッション/トークン、CSRF、ログアウト、レート制限。
- RBAC の権限表、管理者・運用担当者の責務分離。
- PostgreSQL migration ツール、DB 接続方法、transaction manager、repository 境界、ローカル DB 起動方法。
- API 入力検証、検索制限、エラー応答形式、ログ出力規則、データ分類、マスキング規則。

### docs/実装不一致

- README と現行実装の大きな不一致はない。
- docs は MVP 全体の設計方針を示しているが、実装は health check のみであり、業務機能は未実装として明示されている。
- `docs/data-model.md` は `transactions.transaction_type` 候補と `balance_after` を持つが、残高増減方向、`balance_after` の検証ルール、`reversal` の扱いはまだ具体化されていない。

### レビュー未反映

- cycle 004 implementer は `blocked: accepted scope not found` を記録し、ソースコード、README、設計文書、DB schema、migration を変更していない。
- cycle 004 planner は「元帳・残高方向・成功時 transaction 境界を docs に具体化する」scope を作ったが、並列実行の順序により同 cycle implementer には参照されず、実装未反映。
- cycle 004 code-reviewer は、業務 API 前に DB transaction 境界、認証認可、エラー分類、監査ログ境界のいずれかを小さな accepted scope にすることを推奨。
- cycle 004 security-reviewer は、認証/RBAC、入力検証/エラー応答/ログマスキング、監査ログ境界、health/readiness 公開範囲を次候補として推奨。
- cycle 004 banking-reviewer は、`transaction_type` ごとの残高増減方向、`balance_after` の意味、取引履歴不可変性、冪等性、並行更新、監査ログ境界を業務 API / DB schema 前に扱うことを推奨。

## 入力レビュー

### human notes

- `docs/ai/output/human/` は存在しないため、追加の human notes はない。

### cycle 004 implementer

- 同一 cycle の accepted scope を確認できなかったため、`blocked: accepted scope not found` と記録。
- ソースコード、README、設計文書、DB schema、migration は変更していない。
- 次に必要な入力として、planner が対象ファイル、変更しないこと、テスト方針を含む accepted scope を記録することを挙げている。

### cycle 004 code-reviewer

- 現行 Go skeleton に修正必須のコード不具合はない。
- PostgreSQL schema と transaction 境界が未具体化のまま業務 API に進むと整合性リスクが高い。
- Go の業務レイヤ、エラー分類、HTTP 応答変換の境界が未定義。
- 次 cycle では、DB transaction 境界と残高更新方針、認証/RBAC、API エラー形式/入力検証/ログ方針、PostgreSQL migration 方針のいずれか 1 つへ絞ることを推奨。

### cycle 004 security-reviewer

- 業務 API 追加前の認証・認可仕様が実装可能な粒度まで確定していない。
- 入力検証、エラー応答、ログマスキングの共通標準が不足している。
- 監査ログの記録境界、失敗時扱い、改ざん耐性が未確定。
- 現在の HTTP skeleton 自体には新規の High / Medium セキュリティ問題は確認されていない。

### cycle 004 banking-reviewer

- 現行実装は業務データを扱わず、現時点の元帳・残高事故リスクは発火しない。
- 元帳モデルは追記型方針を持つが、取引種別と取消の意味が実装可能な粒度ではない。
- 冪等性キーのスコープと同一キー異内容時の扱いが未確定。
- 並行出金・並行振込時の残高保護方式が未確定。
- 監査ログの成功・失敗時境界が未整理。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. 元帳・残高方向・成功時 transaction 境界を docs に具体化する | `docs/design-principles.md` は残高非負、取引履歴、原子性を定義済み。`docs/data-model.md` は `transactions` と `balance_after` を持つ。cycle 004 planner で採択済みだが未実装。 | `transaction_type` ごとの増減方向、`balance_after` の意味、成功時に残高更新と取引履歴作成を同一 DB transaction に入れるルールが不足。 | 入金、出金、振込、DB schema に進む前に、金融事故リスクの核となる整合性ルールを共有できる。 | banking-reviewer: 元帳整合性。code-reviewer: DB transaction 境界。security-reviewer: 監査ログ未確定事項の分離。 | `reversal`、並行更新方式、冪等性スコープ、監査ログ書き込み失敗時の扱いは人間確認事項に残す。 |
| B. MVP 認証方式と RBAC / 水平権限チェック方針を docs に具体化する | `docs/mvp.md` と `docs/design-principles.md` は認証・認可を必須としている。cycle 004 security-reviewer が High として指摘。 | Cookie session / Bearer token、パスワードハッシュ、CSRF、ロール権限表、管理者作成方法が未定義。 | 業務 API は認証認可なしでは安全に追加できない。 | security-reviewer: 水平権限不備、認証強度、秘密情報。code-reviewer: middleware / handler 境界。 | 安全上重要な仕様であり、人間確認なしに最終確定しない。 |
| C. REST API 入力検証・エラー応答・SQL injection 防止・ログマスキング方針を docs 化する | `docs/security-notes.md` は入力検証と秘密情報ログ禁止を示す。cycle 004 security-reviewer が Medium として指摘。 | request body size limit、金額/口座番号/login_id/email/検索条件制約、エラー JSON 形式、ログ禁止項目が未定義。 | 業務 API 追加時の情報漏えい、検証漏れ、テストばらつきを防ぐ。 | security-reviewer: 情報露出。code-reviewer: エラー分類。banking-reviewer: 失敗時証跡。 | 監査ログ境界と重なるため、失敗監査ログの最終方針は分離する。 |
| D. 監査ログ境界、失敗時扱い、閲覧権限、マスキング方針を docs 化する | `docs/design-principles.md` と `docs/data-model.md` は監査ログ方針とテーブル候補を持つ。cycle 004 security/banking reviewer が指摘。 | 成功ログを同一 transaction に含めるか、失敗ログを rollback 後にどう残すか、書き込み失敗時に fail closed するか未確定。 | 重要操作の説明責任と事故調査の土台になる。 | security-reviewer: 機微情報と閲覧権限。banking-reviewer: 残高変更との整合性。code-reviewer: transaction 境界。 | 書き込み失敗時の業務処理継続可否は人間確認事項。 |
| E. PostgreSQL migration 方針と最小 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。cycle 004 code-reviewer が High として DB transaction / schema 未具体化を指摘。 | migration ツール、ID 型、制約、index、DB 起動方法、冪等性 scope、残高競合方式が未定義。 | DB 側でも残高非負、金額正数、取引履歴、冪等性を守る土台になる。 | code-reviewer: migration と transaction。banking-reviewer: 残高・元帳・冪等性。security-reviewer: 個人情報とシークレット。 | schema は後戻りしにくい。元帳・認証・監査・冪等性の未確認事項を減らしてから採択する。 |

## 採択

### 採択: A. 元帳・残高方向・成功時 transaction 境界を docs に具体化する

- 理由: cycle 004 の同 scope は未実装であり、直近 reviewer 群も業務 API / DB schema 前に元帳・残高・transaction 境界を具体化する必要を繰り返し指摘している。これは実装や schema 確定に進まず、既存 docs の設計原則を実装可能な粒度へ寄せる小さい改善である。
- banking-reviewer 入力への対応: `transaction_type` ごとの残高増減方向、`balance_after` の意味、取引履歴の追記型、成功時の残高更新と取引履歴作成の同一 DB transaction 境界を明示する。
- code-reviewer 入力への対応: DB schema、migration、repository、transaction manager 実装は行わず、将来実装の参照点になる docs に限定する。
- security-reviewer 入力への対応: 監査ログの重要性は維持するが、失敗時監査ログ、監査ログ書き込み失敗時、認証/RBAC、ログマスキングは今回確定せず人間確認事項または保留に分離する。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| E. PostgreSQL migration 方針と最小 schema を作る | DB schema は後戻りしにくく、元帳方向、冪等性スコープ、監査境界、認証方式、migration ツール、残高競合方式が未確定。今回の accepted scope はその前提となる docs 整理に限定する。 | 元帳・残高方向・成功時 transaction 境界が docs に反映され、人間確認事項が減った後。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| B. MVP 認証方式と RBAC / 水平権限チェック方針を docs に具体化する | 業務 API 前に必須だが、認証方式は安全上重要で、人間確認なしに最終確定しない。今回の cycle では未実装の元帳・残高 scope を先に完了させる。 | Cookie session + CSRF か Bearer token か。パスワードハッシュ方式、ログアウト、管理者作成方法、`operator` を MVP に含めるか。 | 次 cycle 以降の security design scope として採択する。 |
| C. REST API 入力検証・エラー応答・SQL injection 防止・ログマスキング方針を docs 化する | 業務 API 追加前には必要だが、現在の API は `/healthz` の固定応答のみ。監査ログ方針との関係もある。 | request id、エラー応答形式、検索上限、ログに含める actor / IP / User-Agent、マスキング対象。 | 認証・業務 API または監査ログ設計の前に docs scope として採択する。 |
| D. 監査ログ境界、失敗時扱い、閲覧権限、マスキング方針を docs 化する | 重要度は高いが、監査ログ書き込み失敗時に業務処理を止めるかは安全上重要な判断。今回 scope では成功時の残高変更と取引履歴の整合性までに留める。 | 監査ログ書き込み失敗時に fail closed するか。失敗ログを rollback 後でも残すか。閲覧可能ロールとマスキング対象。 | 元帳 docs 反映後に、監査ログ専用 scope として扱う。 |

## accepted scope

### 目的

- 入金、出金、振込、DB schema 実装へ進む前に、残高変更と取引履歴の最小整合性ルールを設計文書へ反映する。
- `transaction_type` ごとの残高増減方向と、`transactions.balance_after` の意味を明確にする。
- 成功した残高変更では、口座残高更新と取引履歴作成を同一 DB transaction に入れる方針を明確にする。
- 後戻りしにくい金融仕様は最終決定せず、人間確認事項として分離する。

### 対象ファイル/領域

- `docs/design-principles.md`
  - 残高変更と取引履歴の同一 DB transaction 方針を追記する。
  - 成功時に確定する最小ルールと、失敗時監査ログなどの未確定事項を分ける。
- `docs/data-model.md`
  - `transactions.transaction_type` ごとの残高増減方向表を追記する。
  - `transactions.balance_after` の意味と制約案を追記する。
  - `reversal` は候補として残しつつ、MVP 初期での扱いは未確定であることを明記する。
- `docs/test-strategy.md`
  - 将来の入金、出金、振込テストで、残高変更と取引履歴作成の同一 transaction、`balance_after`、残高非負、失敗時に残高と取引履歴が変わらないことを確認する方針を追記する。
- `docs/ai/cycles/2026-06-28-005/implementer.md`
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
   - `reversal`: 取消・組戻し・訂正の設計が未確定のため、MVP 初期では方向と利用条件を確定しない。
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
5. `docs/ai/cycles/2026-06-28-005/implementer.md` に、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

### 非対象

- Go ソースコード、HTTP handler、server 設定、DB 接続コードは変更しない。
- PostgreSQL migration、DB schema、SQL、repository、transaction manager は作らない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログの業務 API は実装しない。
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウトは実装・確定しない。
- `reversal`、取消、組戻し、訂正の詳細仕様は確定しない。
- 並行出金・並行振込の制御方式として、行ロック、条件付き UPDATE、その他方式のどれを使うかは確定しない。
- 冪等性キーの一意スコープ、同一キー異内容時の扱い、保存期間は確定しない。
- 監査ログ書き込み失敗時に業務処理を失敗させるか、補償するかは確定しない。
- `README.md` は、今回の docs-only scope では変更しない。実装済み機能に変化がないため。
- cycle 001 から 004 の成果物は編集しない。

### テスト方針

- docs-only 変更のため、実行必須の業務テストはない。
- Go toolchain が利用できる環境では `go test ./...` を実行し、既存コードが壊れていないことを確認する。
- この実行環境のように `go` コマンドがない場合は、`go test ./...` は未実行として `implementer.md` に明記する。
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
- 本ファイル `docs/ai/cycles/2026-06-28-005/planner.md` 以外へ書き込まない。
- 他 agent と直接同期しない。連携は cycle ディレクトリ内の Markdown 成果物のみに限定する。
- 保留事項を accepted scope に混ぜない。

## 人間確認事項

1. MVP で `reversal` を実装対象に含めるか、初期は「取消未実装だが履歴は不可変」と明記するだけにするか。
2. 並行出金・並行振込時の残高保護を、PostgreSQL の行ロック、条件付き UPDATE、または別方式のどれで学習・実装するか。
3. 冪等性キーの一意スコープを、操作種別、送信元口座、ログインユーザー、リクエスト本文 hash のどこまで含めるか。
4. 同一冪等性キーで異なる金額、振込元、振込先、通貨が送られた場合に、拒否、既存結果返却、監査アラートのどれを採るか。
5. 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するか。
6. 失敗時監査ログを、業務 transaction の rollback と独立して残す必要があるか。
7. 業務 API 追加前に、認証方式を Cookie session + CSRF と Bearer token のどちらで検討するか。
8. `admin` が顧客口座の入金・出金・振込を代行できるか、また `operator` を MVP に含めるか。
