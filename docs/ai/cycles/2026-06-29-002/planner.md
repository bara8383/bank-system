# planner: 2026-06-29-002

## repo現状

### 作業開始時に確認したもの

- `git status --short`: 作業開始時点では未コミット変更なし。
- `AGENTS.md`: Go + REST + PostgreSQL を中心に、実装前に既存コードを確認し、重要な設計判断は `docs/` に記録し、README を最新に保つ。不明点は推測で決めず設計案として明示する。
- `.codex/agents/README.md`: cycle は `planner` -> `implementer` -> reviewer 群の直列で進める。agent 同士は直接同期せず、`docs/ai/cycles/<cycle-id>/` 配下の Markdown 成果物だけで連携する。planner は改善案と accepted scope を作り、実装はしない。
- `docs/ai/cycles/README.md`: `planner.md` の必須項目は「repo現状」「入力レビュー」「改善候補」「採択」「却下」「保留」「accepted scope」「実装しないこと」「作業仮定」。
- `.agents/skills/banking-planning/SKILL.md`: repo を確認したうえで候補を出し、採択・却下・保留を分け、implementer が判断を追加せずに作業できる粒度で accepted scope を書く。planner の書き込みは cycle 成果物に限定する。
- `README.md`: 現在の実装範囲は Go 標準ライブラリだけの最小 REST API server と `GET /healthz`。DB 接続、認証、業務 API、監査ログ、冪等性キー処理は未実装。
- `docs/START_HERE.md`, `docs/mvp.md`, `docs/domain-model.md`, `docs/data-model.md`, `docs/use-cases.md`, `docs/design-principles.md`, `docs/security-notes.md`, `docs/test-strategy.md`: MVP と金融品質観点を確認。
- `docs/ai/output/human/001-human-review.md`: reversal は MVP 対象外、PostgreSQL 行ロックによる悲観ロック、冪等性キー構成、重複再送は MVP では拒否、監査ログ書き込み失敗時は fail closed、失敗監査ログは独立保存、Cookie + CSRF、operator は MVP 対象外という回答を確認。
- `docs/ai/cycles/`: 既存 cycle `2026-06-28-001` から `2026-06-29-001` の planner / implementer / reviewer 成果物を確認。
- 既存コード: `go.mod`, `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go` を確認。
- `rg -n "TODO|FIXME|HACK|XXX" .`: 実装コード上の TODO/FIXME は確認できなかった。未実装事項は README、docs、cycle 成果物に記録されている。

### 実装済み

- Go module は `bank-system`、Go version は `go 1.24`。
- `cmd/server/main.go` は標準ライブラリ `net/http` で HTTP server を起動する。
- 既定 listen address は `127.0.0.1:8080`。`BANK_SYSTEM_HTTP_ADDR` で明示的に変更できる。
- `http.Server` には `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` が設定済み。
- `internal/httpapi/router.go` は `GET /healthz` のみを提供する。固定 JSON `{"status":"ok"}` を返し、`GET` 以外は `405 Method Not Allowed` と `Allow: GET` で拒否する。
- server 設定と `/healthz` の unit test が存在し、現状のテストコマンドは `go test ./...`。
- docs には、金額は整数の最小通貨単位で扱うこと、MVP は JPY のみであること、残高を 0 未満にしないこと、入金・出金・振込の取引履歴方向、成功時 transaction 境界、監査ログ境界、`transactions.balance_after >= 0` の制約案、rollback テスト観点が反映済み。

### 設計済みだが未実装

- MVP 業務機能: ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- 初期データモデル候補: `users`, `customers`, `accounts`, `transactions`, `transfer_requests`, `audit_logs`。
- 重要原則: 金額整数、JPY のみ、残高非負、残高変更ごとの取引履歴、監査ログ、認証認可、振込の原子性、冪等性、状態遷移。
- 金額・残高に関するテスト方針: 金額バリデーション、残高計算、残高不足拒否、`balance_after` 非負、取引履歴と現在残高の整合。

### 未設計または具体化不足

- 金額や残高の Go 側 domain 型・validation は未実装。現状は health check のみで、業務ロジックの土台となる package が存在しない。
- PostgreSQL migration ツール、DB 接続方法、transaction manager、repository 境界、ローカル DB 起動方法は未決定。
- `transfer_requests` の状態遷移、処理中再送、成功後再送、同一キー同一内容/異内容時の扱い、冪等性キーの一意スコープ、request body hash、保存期間は未整理。
- 並行出金・並行振込時の PostgreSQL 行ロック方式、2 口座振込時のロック順序、デッドロック回避方針は未整理。
- 監査ログ属性の `ip_address` / `user_agent` の信頼境界、正規化、最大長、制御文字除去、proxy header の扱いは未整理。
- 認証方式、パスワードハッシュ方式、Cookie + CSRF、ログアウト、レート制限、RBAC 権限表は未実装・未整理。

### docs/実装不一致

- README と現行 Go 実装の大きな不一致はない。README は業務 API、DB、認証、監査ログ、冪等性キー処理を未実装としている。
- `docs/use-cases.md` は同じ冪等性キーの成功済み依頼では既存結果を返すとしているが、human notes では MVP は拒否のみ、将来は既存結果返却とする方針が示されている。これは次回以降の冪等性 scope で同期が必要。
- docs は金額整数・残高非負を明確にしているが、Go コードにはまだ金額・残高 validation がない。業務 API 未実装のため不整合による実害はまだ発火しないが、次に入出金・振込へ進む前の小さな実装候補になる。

### レビュー未反映

- `2026-06-29-001` code-reviewer は、成功監査ログ insert 失敗時に同一 DB transaction 全体が rollback されることを `docs/test-strategy.md` に明記する小 scope を推奨している。
- `2026-06-29-001` security-reviewer は、監査ログの `ip_address` / `user_agent` の信頼境界・正規化、`request_body_hash` と冪等性キーの接続、管理者代行操作の actor / subject 分離を次候補として挙げている。
- `2026-06-29-001` banking-reviewer は、振込依頼状態遷移と冪等性キー衝突、口座別取引順序・残高連続性、PostgreSQL 行ロックによる残高保護方針を DB / 業務 API 実装前の候補として挙げている。
- これらの reviewer 入力はいずれも重要だが、今回 cycle では docs-only の追加整理だけではなく、既存 docs で既に明確な「金額整数・残高非負」を Go の小さな domain 実装へ落とし、後続の入出金・振込実装に使える土台を作る余地がある。

## 入力レビュー

### human notes

- `reversal` は MVP に含めない。通常取引を先に明確にする。
- 並行更新制御は PostgreSQL の行ロックによる悲観ロックを学習目的で採用する方向。
- 冪等性キーには操作種別、送信元口座、ログインユーザー、リクエスト本文 hash を含める方向。
- MVP の冪等性重複再送は既存結果返却ではなく拒否を優先する方向。将来は UX のため既存結果返却を検討する。
- 監査ログ書き込み失敗時は、MVP では業務処理を失敗させる方向。将来は別経路の補償を検討する。
- 失敗時監査ログは、業務 transaction の rollback と独立して残す方向。
- 認証情報は Cookie とし、CSRF 対策 token を別に持つ方向。
- `admin` はシステム管理者で代行可能、`operator` は銀行業務代行者として理解するが MVP には含めない方向。

### 直近 cycle `2026-06-29-001`

- implementer は監査ログ境界、失敗時扱い、閲覧権限、マスキング、`transactions.balance_after >= 0`、rollback テスト観点を docs-only で反映した。
- code-reviewer は、差分は概ね scope 適合としつつ、成功監査ログ insert 失敗を rollback 注入点として明示することを推奨した。
- security-reviewer は、監査ログ header 属性の正規化、request body hash と冪等性キーの接続、admin 代行操作の actor / subject 分離を推奨した。
- banking-reviewer は、振込依頼状態遷移と冪等性キー衝突、口座別取引順序・残高連続性、PostgreSQL 行ロックを DB / 業務 API 前の候補として挙げた。

### 過去 cycle

- cycle `2026-06-28-001` から `2026-06-28-003` では、初期 docs、HTTP server skeleton、`/healthz`、README、テストが整備された。
- cycle `2026-06-28-004` と `2026-06-28-005` では、元帳・残高方向・成功時 DB transaction 境界が候補化され、その後 docs に反映された。
- 現在は、業務 API や DB schema に進むにはまだ認証、冪等性、行ロック、監査属性などの未整理事項がある。一方で、金額・残高の最小 validation は既存 docs の方針だけで小さく実装できる。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. 金額・残高の最小 domain 型と validation を Go に追加する | `docs/design-principles.md` は金額を整数の最小通貨単位、MVP では JPY のみ、残高非負と定義。`docs/data-model.md` は `amount > 0`, `balance_amount >= 0`, `balance_after >= 0` の制約案を持つ。`docs/test-strategy.md` は金額バリデーションと残高計算を最優先としている。 | Go コードには金額・残高 validation がなく、業務 API 追加時に handler / repository ごとに個別実装される余地がある。 | 入金・出金・振込へ進む前に、0 円以下拒否、残高非負、残高不足拒否という金融事故防止の最小土台を作れる。DB / 認証なしでも unit test で安全に実装できる。 | code-reviewer: package 境界、エラー設計、過剰設計回避。banking-reviewer: 残高マイナス防止、金額整数。security-reviewer: 入力値を早期拒否し、秘密情報や監査仕様に踏み込まない。 | 通貨は JPY 固定、整数 `int64` の最小単位に限定する。丸め、小数、多通貨、DB schema、HTTP API は扱わない。README は実装範囲更新に限定する。 |
| B. 振込依頼状態遷移と冪等性キー衝突時の扱いを docs に具体化する | human notes と直近 security / banking reviewer が強く推奨。`docs/use-cases.md` と human notes の重複再送方針に差がある。 | 状態遷移、処理中再送、成功後再送、同一キー同一内容/異内容、request body hash、保存期間が未整理。 | 二重送金防止と再送時の安全な結果返却の前提になる。 | banking-reviewer: 二重送金リスク。security-reviewer: replay / 架空送金。code-reviewer: 一意制約と race。 | 重要だが docs-only で範囲が広い。今回の小さな code-changing scope とは分離する。 |
| C. PostgreSQL 行ロックによる並行残高保護方針を docs に具体化する | human notes は PostgreSQL 行ロックで悲観ロックを採用する方向。banking-reviewer が DB 前の候補として推奨。 | `SELECT ... FOR UPDATE` 相当のロック対象、2 口座振込のロック順序、デッドロック時の扱い、テスト方針が未整理。 | 同時出金・同時振込で残高非負を守るために必須。 | banking-reviewer: lost update 防止。code-reviewer: transaction manager。 | DB 実装前に必要だが、今回の domain validation scope とは分けて扱う。 |
| D. 監査ログ属性の正規化・信頼境界を docs に具体化する | security-reviewer が `ip_address` / `user_agent` の信頼境界、最大長、制御文字除去、proxy header の扱いを推奨。 | header 値の正規化、保存長、マスキング、偽装 proxy header の扱いが未整理。 | 監査証跡汚染やログ注入を防ぐ。 | security-reviewer: ログ注入、秘密情報混入。code-reviewer: middleware / audit logger 境界。 | 監査ログ実装前の docs-only scope として有効だが、今回は採択しない。 |
| E. 成功監査ログ insert 失敗時 rollback テスト観点を補強する | code-reviewer が直近で推奨。 | `docs/test-strategy.md` に成功監査ログ insert 失敗の明示的な注入点が不足。 | fail closed を「レスポンス失敗」ではなく「DB 副作用なし」まで含めて共有できる。 | code-reviewer: transaction テスト。banking-reviewer: 証跡欠落防止。 | 小さい docs-only 補修として後続 cycle で実施可能。 |
| F. PostgreSQL migration 方針と最小 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。 | migration ツール、ID 型、index、ロック方式、冪等性、認証/RBAC が未確定。 | DB 側の不変条件を守る土台になる。 | code-reviewer: migration。banking-reviewer: 制約。security-reviewer: 個人情報。 | 設計未反映事項が残るため今回採択しない。 |

## 採択

### 採択: A. 金額・残高の最小 domain 型と validation を Go に追加する

- 理由: 既存 docs で「金額は整数の最小通貨単位」「MVP は JPY のみ」「取引金額は 0 より大きい」「残高と `balance_after` は 0 以上」がすでに明確であり、追加の人間判断なしに小さく実装できる。業務 API、DB schema、認証、監査ログに踏み込まず、後続の入金・出金・振込実装で共通利用できる金融事故防止の土台になる。
- 学習効果: 浮動小数点を使わない金額表現、0 円以下拒否、残高不足時の残高非変更、残高非負を unit test で確認できる。
- scope の小ささ: 新規 domain package と unit test、README の現状更新、implementer 成果物に限定する。HTTP route、DB、migration、業務 API は追加しない。
- reviewer 入力への関係: 直近 reviewer が推奨した冪等性・行ロック・監査属性は重要だが、いずれも範囲が広い。今回の金額・残高 validation は、それらより下位の不変条件として先に実装しても後戻りが小さい。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| F. PostgreSQL migration 方針と最小 schema を作る | 認証/RBAC、冪等性キー、行ロック、監査ログ属性、migration ツールが未確定。schema は後戻りしにくく、今回の小 scope ではない。 | 冪等性、行ロック、監査属性、認証/RBAC の初期 docs が揃った後。 |
| 業務 API の実装に進む | 認証、認可、監査ログ、DB transaction、冪等性、行ロック、エラー応答が未具体化。現時点で handler を追加すると設計判断が実装者依存になる。 | 最低限の認証/RBAC、監査ログ、DB transaction / repository 方針が docs 化された後。 |
| `reversal` / 取消 / 組戻し / 訂正を MVP に含める | human notes で MVP に含めない方針が示されている。通常取引の整合性を先に明確にする。 | 通常の入金・出金・振込・監査ログ・冪等性が実装され、取消専用の設計 cycle を持てる段階。 |
| 多通貨・小数・丸め処理を金額 domain に含める | MVP は JPY のみで、金額は整数の最小通貨単位とする方針が既存 docs にある。多通貨や小数を入れると scope が広がる。 | 多通貨対応を MVP 外から将来 scope へ昇格する場合。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| B. 振込依頼状態遷移と冪等性キー衝突時の扱いを docs に具体化する | human notes と直近 reviewer が推奨しており優先度は高いが、今回の実装 scope に含めると大きくなる。 | 同一キー同一内容も MVP では拒否でよいか。拒否時に `transfer_requests` と `audit_logs` のどちらに何を残すか。 | 次 cycle の banking/security docs scope として採択候補にする。 |
| C. PostgreSQL 行ロックによる並行残高保護方針を docs に具体化する | DB / repository 実装前に必要だが、domain validation とは別の設計論点。 | 2 口座振込は内部 `account_id` 昇順ロックでよいか。残高照会はロック不要でよいか。 | DB transaction / repository 設計前の docs scope として採択候補にする。 |
| D. 監査ログ属性の正規化・信頼境界を docs に具体化する | security 上重要だが、監査ログ実装前の別 docs scope として扱う方がよい。 | 信頼する reverse proxy を MVP で想定するか。`User-Agent` の最大長と制御文字除去方針。 | `docs/security-notes.md`, `docs/data-model.md`, `docs/test-strategy.md` の小 scope として扱う。 |
| E. 成功監査ログ insert 失敗時 rollback テスト観点を補強する | 小さい docs-only 補修として有効だが、今回の code-changing scope からは外す。 | なし。既存方針をテスト観点へ明示するだけで進められる。 | 監査ログ / transaction manager 実装前の補修 scope として採択候補にする。 |
| 認証 Cookie + CSRF と MVP RBAC | 業務 API 前に重要だが、password hash、Cookie 属性、CSRF token、admin 作成、代行範囲まで含み大きい。 | Cookie 属性、CSRF token 発行/検証、password hash 方式、admin 初期作成方法。 | security design scope として別 cycle で扱う。 |
| 口座別取引順序・残高連続性・reconciliation 方針 | 元帳品質として重要だが、DB schema と取引順序キーの検討が必要。 | 取引順序キー、同時刻取引の tie-breaker、定期検証の実行主体。 | `docs/data-model.md` / `docs/test-strategy.md` の元帳検証 scope として別 cycle で扱う。 |

## accepted scope

### 目的

- 既存 docs の金額・残高ルールを、Go の小さな domain 実装と unit test に落とす。
- 後続の入金・出金・振込実装で、0 円以下の取引金額、残高マイナス、残高不足時の残高変更を個別 handler / repository ごとに重複実装しないための最小土台を作る。
- 業務 API、DB、認証、監査ログ、冪等性、行ロックには踏み込まず、実装可能でレビューしやすい差分にする。

### 対象ファイル / 領域

- 新規または既存の Go domain 領域。
  - 推奨: `internal/domain/` 配下に金額・残高用の小さな package を追加する。
  - 既存 package と整合するなら別名でもよいが、HTTP handler には置かない。
- 新規 unit test。
  - 推奨: 追加する package と同じ配下に `_test.go` を置く。
- `README.md`。
  - 現在の実装範囲に「金額・残高 validation の domain 土台」を追加し、未実装一覧との矛盾をなくす。
- `docs/ai/cycles/2026-06-29-002/implementer.md`。
  - 参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

### 実装対象

1. 金額型または validation 関数を追加する。
   - 金額は `int64` の整数最小通貨単位として扱う。
   - MVP の通貨は JPY 固定であり、通貨変換や小数は扱わない。
   - 取引金額は 0 より大きい値だけを有効にする。
   - 残高は 0 以上を有効にする。
2. 残高計算の最小 helper を追加する。
   - 入金相当の加算で残高が増えることを確認できるようにする。
   - 出金相当の減算では、減算額が残高を超える場合にエラーとし、残高不足を表現できるようにする。
   - エラー時に呼び出し側が元の残高を保持できる API 形にする。
3. unit test を追加する。
   - 正の取引金額は受け付ける。
   - 0 円・負の取引金額は拒否する。
   - 0 円残高・正の残高は受け付ける。
   - 負の残高は拒否する。
   - 加算で残高が増える。
   - 残高内の減算は成功する。
   - 残高不足の減算はエラーになり、戻り値の扱いで残高を変えない実装にできる。
4. README を更新する。
   - 現在の実装範囲に、Go domain 層の金額・残高 validation が追加されたことを記録する。
   - 業務 API、DB 接続、認証、監査ログ、冪等性キー処理は引き続き未実装と明記する。
5. `docs/ai/cycles/2026-06-29-002/implementer.md` を作成する。

### 実装しないこと

- HTTP route、handler、request / response schema は追加しない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装しない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager は作らない。
- 認証、認可、ユーザー登録、パスワードハッシュ、Cookie session、CSRF token、ログアウトは実装しない。
- 監査ログ永続化、監査ログ正規化、監査ログ照会、outbox、非同期補償は実装しない。
- 冪等性キー、`transfer_requests` 状態遷移、request body hash、同一キー衝突処理は実装・確定しない。
- PostgreSQL 行ロック、ロック順序、デッドロック回避、分離レベルは実装・確定しない。
- 多通貨、為替、小数、丸め、手数料、利息、取消 / reversal は実装しない。
- 外部依存ライブラリは追加しない。Go 標準ライブラリのみで実装する。

### テスト方針

- `go test ./...` を実行し、既存 server / router tests と新規 domain tests がすべて成功することを確認する。
- `gofmt` を追加・変更した Go ファイルに適用する。
- `git diff --name-only` で、変更が accepted scope 内の Go domain package、README、同 cycle の `implementer.md` に限定されていることを確認する。
- `rg -n "float32|float64"` を追加 domain package に対して実行し、金額実装に浮動小数点を使っていないことを確認する。

### レビューで重点確認してほしい観点

- 金額が浮動小数点ではなく整数最小通貨単位で表現されているか。
- 取引金額 0 以下と残高 0 未満を拒否できるか。
- 残高不足の減算で、呼び出し側が元残高を維持できる API になっているか。
- domain package が HTTP、DB、認証、監査ログの未確定仕様に依存していないか。
- README の現行実装範囲と未実装一覧が矛盾していないか。

## 実装しないこと

- この planner 作業では、`docs/ai/cycles/2026-06-29-002/planner.md` 以外を書き換えない。
- Go ソースコード、テストコード、README、通常 docs は変更しない。
- DB schema、migration、SQL、repository、transaction manager は作らない。
- 業務 API、認証、認可、監査ログ永続化、冪等性キー処理、PostgreSQL 行ロックは実装しない。
- 他 agent と直接同期しない。既存 cycle 成果物は repo 上の入力 artifact としてのみ扱う。
- 金融仕様の最終決定は行わず、既存 docs で明確な範囲だけを implementer 向け accepted scope に落とす。

## 作業仮定

- cycle id はユーザー指定どおり `2026-06-29-002` とする。
- planner の書き込みは `docs/ai/cycles/2026-06-29-002/planner.md` のみに限定する。
- 金額は既存 docs の方針どおり、JPY の最小通貨単位を `int64` 整数で扱う。小数、丸め、多通貨は MVP 対象外とする。
- 取引金額は 0 より大きい必要があり、残高は 0 以上である必要がある。
- 加算時の `int64` overflow は、実装者が小さな helper 内で検出できるなら検出する。検出が大きくなる場合は、今回 scope では少なくとも unit test しやすい validation API を優先し、overflow 詳細は implementer.md に残す。
- README は、実装で現行実装範囲が変わる場合に implementer が更新する。planner は README を変更しない。
- 今回の accepted scope は、直近 reviewer が推奨した冪等性・行ロック・監査属性 docs scope を否定しない。それらは DB / 業務 API 前の高優先候補として保留する。
