# planner: 2026-06-29-001

## repo現状

### 作業開始時に確認したもの

- `git status --short`: 作業開始時点では未コミット変更なし。
- `.codex/agents/planner.toml`: planner は repo 現状、人間レビュー、reviewer 群の出力を読み、MVP/改善案と accepted scope を `docs/ai/cycles/<cycle-id>/planner.md` に書く。実装、ソースコード変更、DB schema 確定、認証方式や金融仕様の最終決定は行わない。
- `.codex/agents/README.md`: cycle は `planner` -> `implementer` -> reviewer 群の直列で進め、agent 同士は直接同期せず、`docs/ai/cycles/<cycle-id>/` 配下の Markdown 成果物で連携する。
- `docs/ai/cycles/README.md`: planner 出力は「repo現状」「入力レビュー」「改善候補」「採択」「却下」「保留」「accepted scope」「実装しないこと」「作業仮定」を含める。
- `.agents/skills/banking-planning/SKILL.md`: planner は repo-grounded に候補を出し、採択/却下/保留を明示し、implementer が判断を追加せずに作業できる accepted scope を作る。書き込みは planner 成果物に限定する。
- `AGENTS.md`: Go + REST + PostgreSQL を中心に、設計判断は docs に記録し、不明点は推測で決めず設計案として明示する。
- `README.md`: 現在の実装範囲は Go 標準ライブラリだけの最小 REST API server と `GET /healthz`。DB 接続、認証、業務 API、監査ログ、冪等性キー処理は未実装。
- `docs/START_HERE.md`, `docs/mvp.md`, `docs/domain-model.md`, `docs/data-model.md`, `docs/use-cases.md`, `docs/design-principles.md`, `docs/security-notes.md`, `docs/test-strategy.md`: MVP と金融品質観点を確認。
- `docs/ai/output/human/001-human-review.md`: planner からの人間確認事項への回答を確認。
- `docs/ai/cycles/`: 既存 cycle `2026-06-28-001` から `2026-06-28-005` と、同一 cycle `2026-06-29-001` の既存成果物を確認。
- 既存コード: `go.mod`, `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go` を確認。
- `rg -n "TODO|FIXME|HACK|XXX" .`: 明示的な未処理タグは cycle 成果物内の記録のみで、実装コード上の TODO/FIXME は確認できなかった。

### 実装済み

- Go module は `bank-system`。
- `cmd/server/main.go` は標準ライブラリ `net/http` で HTTP server を起動する。
- 既定 listen address は `127.0.0.1:8080`。`BANK_SYSTEM_HTTP_ADDR` で明示的に変更できる。
- `http.Server` には `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` が設定済み。
- `internal/httpapi/router.go` は `GET /healthz` のみを提供する。固定 JSON `{"status":"ok"}` を返し、`GET` 以外は `405 Method Not Allowed` と `Allow: GET` で拒否する。
- server 設定と `/healthz` の unit test が存在する。
- `docs/design-principles.md`, `docs/data-model.md`, `docs/test-strategy.md` には、前回 scope に相当する元帳・残高方向・成功時 DB transaction 境界の具体化が反映済み。

### 設計済みだが未実装

- MVP 業務機能: ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- 初期データモデル候補: `users`, `customers`, `accounts`, `transactions`, `transfer_requests`, `audit_logs`。
- 重要原則: 金額整数、残高非負、残高変更ごとの取引履歴、監査ログ、認証認可、振込の原子性、冪等性、状態遷移。
- 成功時の最小 transaction 境界: 入金は対象口座残高増加と入金取引履歴、出金は対象口座残高減少と出金取引履歴、振込は両口座残高更新・2 件の取引履歴・振込依頼成功状態更新を同一 DB transaction に含める方針。
- 取引履歴: `transaction_type` ごとの残高増減方向、`balance_after` は対象口座に当該取引を適用した直後の残高、振込は debit / credit を別行で記録する方針。

### 未設計または具体化不足

- 監査ログについて、成功・失敗の記録要否は原則として示されているが、成功監査ログを業務 DB transaction に含めるか、失敗監査ログを rollback と独立してどう残すか、監査ログ書き込み失敗時に業務処理をどう扱うかが docs 間で実装可能な粒度に揃っていない。
- `audit_logs` の閲覧可能ロール、マスキング対象、秘密情報・個人情報・request body の扱いが未具体化。
- `transactions.balance_after >= 0` が本文にはあるが、`docs/data-model.md` の「主な制約案」に未反映。
- transaction rollback テストは業務拒否と途中失敗注入の区別が不足。
- `transfer_requests` の状態遷移、処理中再送、成功後再送、同一キー異内容時の扱いが未整理。
- 冪等性キーの一意スコープ、request hash、保存期間、衝突時の audit 対象が未反映。
- 並行出金・並行振込時の PostgreSQL 行ロック方式、ロック順序、デッドロック回避方針が未反映。
- 認証方式、パスワードハッシュ方式、Cookie + CSRF、ログアウト、レート制限、RBAC 権限表は未実装・未反映。
- PostgreSQL migration ツール、DB 接続方法、transaction manager、repository 境界、ローカル DB 起動方法は未決定。

### docs/実装不一致

- README と現行 Go 実装の大きな不一致はない。README は業務 API、DB、認証、監査ログ、冪等性キー処理を未実装としている。
- `docs/design-principles.md` と `docs/security-notes.md` は「重要操作の成功と失敗を監査ログに残す」原則を持つ。一方、元帳 transaction 境界の追記では「失敗時監査ログ、監査ログ書き込み失敗時の業務処理」を未確定としているため、記録要否と保存境界を分けて整理する必要がある。
- `docs/use-cases.md` は同じ冪等性キーの成功済み依頼では既存結果を返すとしているが、人間レビューでは MVP は拒否のみ、将来は既存結果返却とする方針が示されている。これは次回以降の冪等性 scope で同期が必要。

### レビュー未反映

- 同一 cycle の既存 code-reviewer は、`transactions.balance_after >= 0` を制約案へ追加すること、transaction rollback テスト観点を業務拒否と途中失敗に分けることを推奨している。
- 同一 cycle の既存 security-reviewer は、監査ログ境界、失敗時扱い、閲覧権限、マスキング方針を次の security design scope 候補にするよう推奨している。
- 同一 cycle の既存 banking-reviewer は、失敗時監査ログの扱いが既存 docs と衝突して見えること、振込依頼の状態遷移、`balance_after` の連続性検証ルールを次候補に挙げている。
- human notes は、reversal を MVP に含めない、PostgreSQL 行ロックで悲観ロック、冪等性キーには操作種別・送信元口座・ログインユーザー・リクエスト本文 hash を含める、MVP の重複再送は拒否、監査ログ書き込み失敗時は業務処理を失敗させる、失敗ログは rollback と独立して残す、認証は Cookie + CSRF、operator は MVP に含めないという方向性を示している。

## 入力レビュー

### human notes

- `reversal` は MVP に含めない。通常取引をまず明確にし、取消・組戻し・訂正は既存取引との関係、二重取消防止、取消可否、監査が重いため後回しにする。
- 並行更新制御は PostgreSQL の行ロックによる悲観ロックを学習目的で採用する方向。
- 冪等性キーには操作種別、送信元口座、ログインユーザー、リクエスト本文 hash をすべて含める方向。
- MVP の冪等性重複再送は、既存結果返却ではなく拒否を優先する方向。将来は UX のため既存結果返却を検討する。
- 監査ログ書き込み失敗時は、MVP では業務処理を失敗させる方向。将来は別経路の補償を検討する。
- 失敗時監査ログは、業務 transaction の rollback と独立して残す方向。
- 認証情報は Cookie とし、CSRF 対策用 token を別に持つ方向。
- `admin` はシステム管理者で代行可能、`operator` は銀行業務代行者として理解するが MVP には含めない方向。

### 同一 cycle の既存 implementer / reviewer 成果物

- 同一 cycle `2026-06-29-001` には、既存の planner / implementer / reviewer 成果物が存在する。今回の作業では他 agent と直接同期せず、既存 repo artifact として読み取った。
- 既存 implementer は元帳・残高方向・成功時 DB transaction 境界の docs-only scope を実施し、Go ソース、DB schema、migration、業務 API は変更していない。
- code-reviewer は docs-only 変更が概ね scope に適合するとしつつ、`transactions.balance_after >= 0` の制約案未反映と rollback テスト観点の不足を指摘した。
- security-reviewer は、成功した残高変更の監査ログが欠落する余地、失敗時監査ログと rollback の関係、監査ログ閲覧権限・マスキング・書き込み失敗時方針の不足を指摘した。
- banking-reviewer は、失敗時監査ログの扱いが既存 docs と衝突して見えること、振込依頼状態遷移、`balance_after` の連続性検証不足を指摘した。

### 過去 cycle

- cycle 001 から 003 では、初期 docs、HTTP server skeleton、`/healthz`、README、テストが整備された。
- cycle 004 と 005 では、元帳・残高方向・成功時 DB transaction 境界が繰り返し候補化されたが、並列実行順序により一度は blocked になった。
- 現在の repo では、元帳・残高方向・成功時 DB transaction 境界の docs 具体化は反映済みであり、次は監査ログ境界を整える価値が高い。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. 監査ログ境界、失敗時扱い、閲覧権限、マスキング方針を docs に具体化する | `docs/design-principles.md`、`docs/security-notes.md`、`docs/test-strategy.md` は監査ログを必須としている。human notes は監査ログ書き込み失敗時は業務処理を失敗、失敗ログは独立保存と回答。security / banking reviewer が次候補として推奨。 | 成功監査ログの transaction 境界、失敗監査ログの rollback 独立性、書き込み失敗時、閲覧可能ロール、秘密情報・個人情報・raw body の扱いが分かれていない。 | 残高更新、取引履歴、認可拒否、不正試行を後から説明するため、業務 API / DB 実装前に必要。 | security-reviewer: 証跡欠落、マスキング、閲覧権限。banking-reviewer: 失敗証跡と金融事故調査。code-reviewer: transaction 境界。 | 実装や schema 確定には進まず、human notes の方向性を設計案として docs に反映する。 |
| B. `transactions.balance_after` 制約案と rollback テスト観点を補強する | code-reviewer が `transactions.balance_after >= 0` の制約案未反映と rollback テスト不足を指摘。 | 本文と制約案の同期不足、業務拒否と途中失敗注入の区別不足。 | DB schema / transaction manager 実装時の取りこぼしを防ぐ。 | code-reviewer: DB 制約とテスト可能性。banking-reviewer: 元帳整合性。 | 小さい docs-only 補修として A と同時に扱えるが、A の監査ログ scope が膨らむ場合は分離する。 |
| C. 振込依頼状態遷移と冪等性キー衝突時の扱いを docs に具体化する | `docs/data-model.md` は `accepted`, `processing`, `succeeded`, `failed`, `cancelled` を候補に持つ。human notes は冪等性キー構成と重複時拒否方針を回答。banking-reviewer が状態遷移不足を指摘。 | 状態遷移表、処理中再送、成功後再送、同一キー異内容、request hash、保存期間が未整理。 | 二重送金防止と再送時の安全な結果返却の前提になる。 | banking-reviewer: 二重送金リスク。security-reviewer: 架空送金・衝突悪用。code-reviewer: 一意制約と race。 | audit logging と関連するが scope が大きいため別 cycle が望ましい。 |
| D. PostgreSQL 行ロックによる並行残高保護方針を docs に具体化する | human notes は PostgreSQL 行ロックで悲観ロックを採用する方向。docs は残高非負と原子性を定義済み。 | `SELECT ... FOR UPDATE` 相当のロック対象、2 口座振込のロック順序、デッドロック回避、テスト方針が未整理。 | 同時出金・同時振込で残高非負を守るために必須。 | banking-reviewer: lost update 防止。code-reviewer: transaction manager。 | DB 実装前に必要だが、監査ログ境界と分けて扱う。 |
| E. 認証 Cookie + CSRF と MVP RBAC を docs に具体化する | human notes は Cookie + CSRF、operator は MVP 対象外、admin は代行可能と回答。docs は認証認可必須。 | Cookie 属性、CSRF token、password hash、管理者作成方法、顧客/管理者の権限表が未整理。 | 業務 API は認証認可なしでは追加できない。 | security-reviewer: 水平権限不備、CSRF、secret。code-reviewer: middleware 境界。 | セキュリティ上重要で scope が大きいため、監査ログ後に別 scope とする。 |
| F. PostgreSQL migration 方針と最小 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。 | migration ツール、ID 型、index、ロック方式、監査ログ境界、冪等性が未確定。 | DB 側の不変条件を守る土台になる。 | code-reviewer: migration。banking-reviewer: 制約。security-reviewer: 個人情報。 | 設計未反映事項が残るため、まだ採択しない。 |

## 採択

### 採択: A. 監査ログ境界、失敗時扱い、閲覧権限、マスキング方針を docs に具体化する

- 理由: 現在の docs は監査ログの必要性を明記している一方、成功監査ログ、失敗監査ログ、rollback、書き込み失敗時の扱いが混在している。security / banking reviewer は次 scope として監査ログ境界を明示的に推奨しており、human notes で安全側の方向性も示されたため、業務 API / DB schema 実装前の小さい docs scope として採択する。
- human notes への対応: 監査ログ書き込み失敗時は MVP では業務処理を失敗させる、失敗監査ログは業務 transaction の rollback と独立して残す、operator は MVP に含めない、という方向性を「MVP 初期の設計案」として明示する。
- security-reviewer 入力への対応: 成功監査ログと失敗監査ログの境界、閲覧権限、マスキング、秘密情報・個人情報・raw request body を監査ログへ直接残さない方針を docs に追加する。
- banking-reviewer 入力への対応: 「失敗時監査ログは不要」ではなく「失敗証跡は残すが rollback と独立して保存する」と読めるよう、既存 docs の衝突を解消する。
- code-reviewer 入力への対応: 実装や schema には進まず、将来 transaction manager / repository 実装で参照できる境界とテスト観点に限定する。

### 採択に含める小補修: B の一部

- `transactions.balance_after >= 0` を `docs/data-model.md` の「主な制約案」に追加する。
- `docs/test-strategy.md` に、業務拒否とは別に、残高更新後・取引履歴作成前などの途中失敗を注入して rollback を確認する観点を追加する。
- 理由: A と同じ docs-only で小さく、前 cycle reviewer の具体指摘を短時間で解消でき、監査ログ境界の記述とも衝突しない。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| F. PostgreSQL migration 方針と最小 schema を作る | 監査ログ境界、冪等性、行ロック、認証/RBAC がまだ docs に反映し切れていない。schema は後戻りしにくいため、今回の scope では採択しない。 | 監査ログ境界、冪等性、並行更新、認証/RBAC の初期 docs が揃った後。 |
| 業務 API の実装に進む | 認証、認可、監査ログ、DB transaction、冪等性、行ロック、エラー応答が未具体化。現時点で handler を追加すると設計判断が実装者依存になる。 | 最低限の認証/RBAC、監査ログ、DB transaction / repository 方針が docs 化された後。 |
| `reversal` / 取消 / 組戻し / 訂正を MVP に含める | human notes で MVP に含めない方針が示されている。通常取引の整合性を先に明確にする。 | 通常の入金・出金・振込・監査ログ・冪等性が実装され、取消専用の設計 cycle を持てる段階。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| C. 振込依頼状態遷移と冪等性キー衝突時の扱いを docs に具体化する | human notes によって方向性は出たが、状態遷移、request hash、同一キー同一内容/異内容、処理中再送、保存期間まで含めると今回 scope が大きくなる。 | 同一キー同一内容も MVP では拒否でよいか。拒否時に `transfer_requests` と `audit_logs` のどちらに何を残すか。 | 次 cycle の banking/security scope として採択候補にする。 |
| D. PostgreSQL 行ロックによる並行残高保護方針を docs に具体化する | human notes で悲観ロック方針は示されたが、2 口座振込のロック順序やテストが別途必要。監査ログ境界後に扱う。 | 口座 ID 昇順ロックでよいか。残高照会はロック不要でよいか。 | DB transaction / repository 設計前の scope として採択候補にする。 |
| E. 認証 Cookie + CSRF と MVP RBAC を docs に具体化する | 業務 API 前に重要だが、監査ログ閲覧権限と密接に関連する。今回 scope では監査ログ閲覧を admin 限定案として最小限扱い、認証方式詳細は分離する。 | Cookie 属性、CSRF token 発行/検証、password hash 方式、admin 初期作成方法。 | security design scope として別 cycle で扱う。 |
| `balance_after` の連続性検証と定期 reconciliation | 元帳品質として重要だが、今回の監査ログ scope に含めると広がる。 | 取引順序キー、同時刻取引の順序、定期検証の実行主体。 | `docs/data-model.md` / `docs/test-strategy.md` の元帳検証 scope として別 cycle で扱う。 |
| health / readiness / metrics の公開範囲 | 現在の `/healthz` は固定応答のみで直接リスクは低い。業務 API / readiness / metrics 追加時に必要。 | readiness を同じ server に置くか、内部限定にするか。 | 業務 API または運用 endpoint 追加前に security scope として扱う。 |

## accepted scope

### 目的

- 監査ログについて、記録要否、保存境界、書き込み失敗時、閲覧権限、マスキングを実装前に揃える。
- 既存 docs の「成功・失敗を監査ログに残す」原則と、「失敗時監査ログは未確定」という記述を衝突しない形へ整理する。
- human notes の回答を MVP 初期の設計案として docs に反映し、将来の DB / API 実装で implementer が判断を追加しなくて済むようにする。
- 前 cycle reviewer の小さな指摘である `transactions.balance_after >= 0` 制約案と rollback テスト観点を補強する。

### 対象ファイル/領域

- `docs/design-principles.md`
  - 「重要操作に監査ログを残す」または「お金の移動は原子的に扱う」の周辺に、監査ログの記録要否と保存境界を追記する。
  - 成功監査ログ、業務拒否の失敗監査ログ、DB transaction 途中失敗の失敗監査ログ、監査ログ書き込み失敗時を分ける。
  - 失敗時監査ログは「不要」ではなく「業務 transaction の rollback と独立して残す必要がある」と明記する。
- `docs/security-notes.md`
  - 監査ログに含める情報、含めない情報、閲覧権限、マスキング方針を追記する。
  - MVP では監査ログ照会を `admin` に限定し、`operator` は MVP に含めない方針を追記する。
- `docs/data-model.md`
  - `audit_logs` の説明に、失敗時は `target_id` が未確定または null 相当になり得ること、raw request body や秘密情報は保存しないこと、request body は必要なら hash として扱うことを追記する。
  - 「主な制約案」に `transactions.balance_after` は 0 以上にすることを追加する。
- `docs/test-strategy.md`
  - 監査ログの成功・失敗・閲覧権限・マスキング・書き込み失敗時 fail closed のテスト観点を追記する。
  - 業務拒否とは別に、DB transaction 途中失敗を注入した rollback テスト観点を追記する。
- `docs/ai/cycles/2026-06-29-001/implementer.md`
  - implementer は、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

### 実装対象

1. `docs/design-principles.md` の監査ログ原則を次の粒度に具体化する。
   - 重要操作の成功と失敗は監査ログ対象である。
   - 残高変更を伴う成功操作では、MVP 初期案として、業務データ更新、取引履歴、振込依頼状態更新と成功監査ログを同じ PostgreSQL DB transaction に含める。
   - 権限不足、残高不足、入力不備、存在しない対象などの業務拒否は、業務データを更新せず、失敗監査ログを独立した DB transaction で残す。
   - DB transaction 途中失敗で rollback する場合も、rollback 後に失敗監査ログを独立した DB transaction で残す方針にする。
   - 監査ログ書き込みに失敗した場合、MVP では対象業務処理を成功扱いにしない。成功監査ログが書けない残高変更や権限変更は fail closed とする。
   - 監査ログ自体の書き込み失敗をどう運用通知・再送・補償するかは将来検討として残す。
2. `docs/security-notes.md` の監査・秘密情報・認可の記述を補強する。
   - 監査ログには、actor、action_type、target_type、target_id、result、failure_reason、occurred_at、ip_address、user_agent を含める方針を明記する。
   - パスワード、token、secret、CSRF token、セッション ID、raw request body、過剰な個人情報は監査ログに保存しない。
   - リクエスト同一性の調査に必要な場合は raw body ではなく request body hash を保存する設計案にする。
   - 監査ログ照会は MVP では `admin` のみに限定する。`operator` ロールは MVP 対象外として明記する。
3. `docs/data-model.md` を補強する。
   - `audit_logs.failure_reason` は利用者向け詳細ではなく、運用調査に必要な安全な分類または短い理由にする。
   - 失敗時に具体的な対象が確定しない場合、`target_id` は未設定になり得ることを補足する。
   - raw request body や秘密情報を audit_logs に保存しない。必要なら request body hash 用の属性を将来候補として記録する。
   - 主な制約案に `transactions.balance_after` は 0 以上にすることを追加する。
4. `docs/test-strategy.md` を補強する。
   - 入金、出金、振込の成功時に、業務データ、取引履歴、成功監査ログが整合して作られることを確認する。
   - 残高不足、権限不足、不正金額、存在しない対象などの業務拒否時に、業務データが変わらず失敗監査ログが残ることを確認する。
   - 残高更新後・取引履歴作成前、振込元更新後・振込先更新前、取引履歴作成後・commit 前などの疑似エラーで rollback され、rollback 後に失敗監査ログが独立して残ることを確認する。
   - 監査ログ書き込み失敗時は、MVP では対象業務処理が成功扱いにならないことを確認する。
   - 監査ログに password、token、secret、CSRF token、セッション ID、raw request body、過剰な個人情報が含まれないことを確認する。
   - `admin` 以外が監査ログを照会できないことを確認する。`operator` は MVP では権限表・照会対象に含めない。
5. `docs/ai/cycles/2026-06-29-001/implementer.md` に、docs-only の変更結果、scope 適合性、実装しなかったこと、テスト結果、残った人間確認事項を記録する。

### 非対象

- Go ソースコード、HTTP handler、server 設定、DB 接続コードは変更しない。
- PostgreSQL migration、DB schema、SQL、repository、transaction manager は作らない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装しない。
- 認証、認可、ユーザー登録、パスワードハッシュ、Cookie session、CSRF token、ログアウトは実装しない。
- `reversal`、取消、組戻し、訂正は MVP 初期に含めない方針を確認するに留め、詳細仕様は作らない。
- 冪等性キーの複合一意制約、同一キー同一内容/異内容、処理中再送、保存期間は今回確定しない。
- PostgreSQL 行ロック、ロック順序、デッドロック回避、分離レベルは今回確定しない。
- 監査ログの改ざん検知、outbox、非同期補償、運用アラート、外部 SIEM 連携は実装・確定しない。
- `README.md` は、今回の docs-only 設計整理でユーザー向け実行方法や現行実装範囲が変わらないため変更しない。

### テスト方針

- docs-only 変更のため、新規 Go unit test は追加しない。
- 可能であれば `go test ./...` を実行し、既存 Go skeleton に影響がないことを確認する。環境に `go` がない場合は、その旨を `implementer.md` に warning として記録する。
- `git diff -- docs/design-principles.md docs/security-notes.md docs/data-model.md docs/test-strategy.md docs/ai/cycles/2026-06-29-001/implementer.md` で、変更が accepted scope 内の docs-only であることを確認する。
- `rg -n "監査ログ|audit_logs|failure_reason|balance_after|rollback|ロールバック|password|token|secret|CSRF|operator" docs/design-principles.md docs/security-notes.md docs/data-model.md docs/test-strategy.md` で、監査ログ境界、マスキング、制約、テスト観点が反映されていることを確認する。

### レビューで重点確認してほしい観点

- 失敗時監査ログについて、「記録要否」と「保存 transaction 境界」が分離され、既存 docs と矛盾しないか。
- 成功監査ログを同一 DB transaction に含める MVP 初期案が、将来の PostgreSQL 実装で過度に複雑すぎないか。
- 監査ログ書き込み失敗時 fail closed が、人間レビューの意図と合っているか。
- 監査ログに raw request body や秘密情報を残さない方針が明確か。
- `admin` 限定、`operator` MVP 対象外の記述が認証/RBAC の将来 scope と衝突しないか。
- `transactions.balance_after >= 0` の制約案と rollback テスト観点が reviewer 指摘を満たしているか。

## 実装しないこと

- この planner 作業では、`docs/ai/cycles/2026-06-29-001/planner.md` 以外を書き換えない。
- Go ソースコード、テストコード、README、設計文書本体は変更しない。
- DB schema、migration、SQL、repository、transaction manager は作らない。
- 業務 API、認証、認可、監査ログ永続化、冪等性キー処理、PostgreSQL 行ロックは実装しない。
- 他 agent と直接同期しない。既存の同一 cycle 成果物は repo 上の入力 artifact としてのみ扱う。
- 金融仕様の最終決定は行わず、human notes の回答を反映する場合も MVP 初期の設計案として記録する。

## 作業仮定

- 同一 cycle に既存 implementer / reviewer 成果物があるが、ユーザーの指示により、それらは既存 artifact として読み取り、今回の planner 成果物だけを更新する。
- human notes の「MVPとしては失敗させる」は、監査ログ書き込み失敗時に対象業務処理を成功扱いにしない、という回答として扱う。
- human notes の「独立して残す」は、失敗時監査ログを業務 DB transaction の rollback と独立して残す、という回答として扱う。
- 成功監査ログを同一 DB transaction に含める案は、MVP 初期の学習用設計として採択する。将来、outbox や非同期補償が必要になった場合は別 scope で見直す。
- `operator` は MVP 対象外とし、監査ログ閲覧は `admin` のみに限定する設計案で進める。
- `reversal` は MVP 初期に含めない。既存取引履歴は追記型・削除禁止の原則を維持し、取消・組戻し・訂正は将来 scope とする。
- `README.md` は現行実装範囲と実行方法に変更がないため、今回 scope では更新不要とする。
