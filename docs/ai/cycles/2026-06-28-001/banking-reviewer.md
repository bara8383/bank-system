# banking-reviewer: 2026-06-28-001

## レビュー対象

- 対象 cycle: `2026-06-28-001`
- 役割: `banking-reviewer`
- 対応 skill: `.agents/skills/banking-ledger-review`
- レビュー種別: repo-wide review
- 理由: `git status --short` は空で、未コミットの実装差分は確認できなかった。同一 cycle の `implementer.md` は `blocked: accepted scope not found` を記録しており、cycle 001 としての実装差分はない。

## 確認した入力

- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `.agents/skills/banking-ledger-review/SKILL.md`
- `.agents/skills/banking-ledger-review/references/banking-quality-rubric.md`
- `AGENTS.md`
- `README.md`
- `docs/START_HERE.md`
- `docs/design-principles.md`
- `docs/domain-model.md`
- `docs/data-model.md`
- `docs/use-cases.md`
- `docs/mvp.md`
- `docs/security-notes.md`
- `docs/test-strategy.md`
- `docs/decision-logs/2026-06-27-subagent-parallel-cycle.md`
- 同一 cycle 成果物: `planner.md`, `implementer.md`, `code-reviewer.md`, `security-reviewer.md`
- 既存 cycle 成果物: cycle 002, 003, 004 の `banking-reviewer.md` と関連する planner / implementer
- 現在の Go 実装: `go.mod`, `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go`

## 現状整理

- 現在の実装は Go 標準ライブラリによる最小 HTTP server と `GET /healthz` に限定されている。
- `/healthz` は固定 JSON `{"status":"ok"}` を返すだけで、口座、残高、取引履歴、振込依頼、監査ログ、DB 接続、認証認可を扱わない。
- README は、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、PostgreSQL、監査ログ、冪等性キー処理を未実装として明示している。
- 設計文書には、金額整数、残高非負、追記型取引履歴、振込の原子性、冪等性、監査ログの重要性が記録されている。

## Finding

### Finding 1: 現在の実装は金融ドメイン状態を変更せず、元帳・残高上の直接的な事故リスクは発火しない

#### 根拠

- `internal/httpapi/router.go` は `/healthz` の handler のみを登録している。
- `cmd/server/main.go` は HTTP server の起動、listen address、timeout の設定だけを行い、DB 接続や業務処理を持たない。
- `README.md` は、DB 接続、認証、業務 API、監査ログ、冪等性キー処理が未導入であることを明示している。
- 同一 cycle の `implementer.md` は accepted scope 不在により blocked であり、cycle 001 の実装変更を行っていない。

#### 影響

- 現時点では、二重送金、残高マイナス、振込の片側成功、取引履歴欠落、取消時の履歴改変、監査ログと残高変更の不一致は発生しない。
- 一方で、MVP の中心機能である入出金・振込・取引履歴・監査ログは未実装であり、実装前の設計具体化が事故予防の主な課題になる。

#### 推奨修正

- 現在の Go 実装に対する banking-reviewer 観点の必須修正はない。
- `/healthz` を将来 readiness に拡張する場合も、残高、口座数、取引件数、DB 接続先、内部設定などの金融・運用詳細を公開レスポンスに含めない方針を維持する。

#### 次サイクル planner への入力

- 業務 API や DB schema に進む前に、元帳・残高・取引履歴・監査ログ・冪等性の設計 scope を小さく切り出す。

### Finding 2: `transaction_type` ごとの残高増減方向と `balance_after` の検証ルールが未確定

#### 根拠

- `docs/data-model.md` は `transactions.transaction_type` として `deposit`, `withdrawal`, `transfer_debit`, `transfer_credit`, `reversal` を候補に挙げている。
- `transactions.amount` は正の整数として保存する方針だが、残高計算時に各取引種別を加算・減算のどちらとして扱うかの正式な対応表はまだ不足している。
- `transactions.balance_after` は候補カラムとして存在するが、「対象口座へ当該取引を適用した直後の残高」なのか、振込全体の後残高なのかを実装者が迷わない粒度では未整理である。

#### 影響

- 入金・出金・振込出金・振込入金・取消の符号解釈が処理ごとに分散すると、`accounts.balance_amount` と `transactions.balance_after` の不一致を検出しづらくなる。
- 取消や補正取引を追加したとき、正の `amount` と残高方向の関係が曖昧になり、明細表示、残高再計算、事故調査で説明できない履歴が生まれる可能性がある。

#### 推奨修正

- `transaction_type` ごとの残高方向を docs に明文化する。
- `deposit` と `transfer_credit` は対象口座を増加、`withdrawal` と `transfer_debit` は対象口座を減少として定義する。
- `balance_after` は「その取引を対象口座へ適用した直後の対象口座残高」と定義し、0 以上であることを制約案に含める。
- 検証ルールとして「初期残高 + 取引種別別の符号付き合計 = 現在残高」を将来の整合性チェック候補にする。

#### 次サイクル planner への入力

- `docs/data-model.md` と `docs/design-principles.md` に、取引種別、残高方向、`balance_after`、残高検証ルールを追記する docs-only scope を候補にする。

### Finding 3: 取消・組戻しを追記型で扱う方針はあるが、`reversal` の意味が未定義

#### 根拠

- `docs/design-principles.md` は、取引履歴を原則削除せず、取消が必要な場合は取消取引を追加するとしている。
- `docs/data-model.md` は `reversal` を取引種別候補に含めている。
- しかし、`reversal` が元取引の逆方向を表すのか、誤記訂正、取消、組戻しを同じ種別で扱うのか、二重取消をどう防ぐのかは未確定である。

#### 影響

- 取消仕様が曖昧なまま実装されると、既存取引の削除・更新で残高を合わせる実装が入り込み、取引履歴の不可逆性と監査可能性が損なわれる。
- `related_transaction_id` の使い方が曖昧だと、どの取引がどの取引を打ち消したのか追跡できず、利用者問い合わせや内部調査に耐えにくい。

#### 事故シナリオ

1. 誤入金を取り消すため、既存の `deposit` 取引行を削除または金額更新する。
2. `accounts.balance_amount` は手作業で修正される。
3. 元の誤入金が発生した事実と取消判断の証跡が取引履歴から消え、監査時に残高変動を説明できない。

#### 推奨修正

- MVP 初期で取消を実装しない場合でも、「既存取引行は削除・更新しない」「将来の取消・訂正は追記取引で表現する」と明記する。
- `reversal` を使う場合は、取消対象 `related_transaction_id`、残高方向、取消可能な元取引種別、同一元取引への二重取消禁止を別途設計する。

#### 次サイクル planner への入力

- `reversal` を MVP に含めるか、人間確認事項に分離する。
- 初期 docs scope では、少なくとも取引履歴の不可変性と追記型取消方針を明記する。

### Finding 4: 冪等性キーのスコープ、同一キー異内容時の扱い、保存期間が未確定

#### 根拠

- `docs/design-principles.md` は、同じ振込依頼が複数回処理されないよう冪等性キーを使うとしている。
- `docs/data-model.md` は、`transfer_requests.idempotency_key` を依頼者または振込元口座の範囲で一意にする案を示している。
- 同じキーで金額、振込元、振込先、通貨が異なるリクエストが来た場合の扱い、`processing` 再送時の扱い、キーの保存期間は未確定である。

#### 影響

- 同一キーを異なる内容に再利用したときに既存結果を返すと、利用者が意図しない振込結果を正常応答として受け取る可能性がある。
- 一意制約違反だけに任せると、利用者向けエラー、監査ログ、失敗依頼の保存有無がばらつき、二重送金防止と事故調査の両方が弱くなる。

#### 推奨修正

- 「同じスコープ、同じ冪等性キー、同じリクエスト内容なら同じ結果を返す」と定義する。
- 「同じキーだが内容が異なる場合は冪等性キー衝突として拒否し、監査対象にする」と定義する。
- 比較対象として、操作種別、振込元、振込先、金額、通貨、または request hash を保存する方針を検討する。

#### 次サイクル planner への入力

- 冪等性キーの一意スコープを `requested_by_user_id + idempotency_key`、`source_account_id + idempotency_key`、または操作種別・request hash を含む複合条件から選ぶ人間確認事項として残す。
- 同一キー異内容時の拒否仕様と監査記録を docs scope 候補にする。

### Finding 5: 並行出金・並行振込時の残高保護方式が未確定

#### 根拠

- `docs/design-principles.md` は、残高をマイナスにしないことと、振込を原子的に扱うことを基本原則にしている。
- `docs/test-strategy.md` は、将来追加するテストとして並行実行時の残高競合テストを挙げている。
- PostgreSQL を中心にする方針はあるが、行ロック、条件付き UPDATE、分離レベル、楽観ロック、複数口座ロック順序はまだ決まっていない。

#### 影響

- 単純な「残高読取、アプリケーションで残高不足判定、後で更新」では、同一口座への同時出金や同時振込で lost update が起き得る。
- 冪等性キーは同一リクエストの重複を防ぐ仕組みであり、別キーの並行出金による過剰引落は防げない。

#### 事故シナリオ

1. 口座 A の残高が 10,000 円。
2. 8,000 円の振込依頼が同時に 2 件処理される。
3. 両方が更新前残高 10,000 円を読み、残高不足ではないと判定する。
4. 排他制御が不十分だと 2 件とも成功扱いになり、実質 16,000 円の出金を許す。

#### 推奨修正

- 初期方針として、トランザクション内で対象口座行をロックしてから残高判定・更新する方式、または `balance_amount >= amount` を条件にした単一 UPDATE の更新件数で判定する方式のどちらかを選ぶ。
- 振込では複数口座を扱うため、口座 ID 昇順などでロック順序を固定し、デッドロック回避方針を docs に残す。
- `accounts.balance_amount >= 0` の DB 制約を必須候補にする。

#### 次サイクル planner への入力

- 出金・振込の残高競合制御方式を docs scope 候補にする。
- MVP テスト戦略に、同一口座への並行出金・並行振込で残高がマイナスにならないことを具体テストとして追加する。

### Finding 6: 監査ログと取引履歴の transaction 境界が未整理

#### 根拠

- `docs/design-principles.md` は、すべての残高変更に取引履歴を残し、重要操作に監査ログを残すとしている。
- `docs/use-cases.md` は、入金、出金、振込の成功時に監査ログを記録するとしている。
- 残高更新、取引履歴作成、振込依頼更新、成功監査ログを同一 DB transaction に含めるか、業務拒否やシステム障害の失敗ログをどう残すかは未確定である。

#### 影響

- 成功時監査ログを業務 transaction 外で書くと、残高は変わったのに監査ログが欠落する可能性がある。
- 失敗時監査ログを業務 transaction 内だけで書くと、ロールバックにより拒否・障害の証跡が消える可能性がある。
- 監査ログ書き込み失敗時に業務処理を止めるか進めるかが未定だと、実装ごとに事故時の説明責任が変わる。

#### 推奨修正

- 成功、業務拒否、システム障害を分け、監査ログの保存境界を決める。
- 残高変更を伴う成功操作では、残高更新、取引履歴、振込依頼状態、成功監査ログの整合性をどう担保するかを設計する。
- 失敗監査ログは、業務データのロールバック後でも証跡が残る方針を検討し、人間確認事項として残す。

#### 次サイクル planner への入力

- 監査ログの transaction 境界を docs scope 候補にする。
- 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するかを人間確認事項にする。

## 人間確認事項

1. `reversal`、取消、組戻しを MVP 初期の実装対象に含めるか。含めない場合、初期 MVP では「既存取引の訂正・取消は未対応」と明示してよいか。
2. 冪等性キーの一意スコープを、依頼者、振込元口座、操作種別、リクエスト本文 hash のどこまで含めるか。
3. 同一冪等性キーで異なる amount / destination / source が送られた場合、拒否、既存結果返却、監査アラートのどれにするか。
4. 並行出金・並行振込の残高保護方式を、PostgreSQL 行ロック、条件付き UPDATE、または別方式のどれにするか。
5. 監査ログ書き込み失敗時、残高変更を伴う業務処理を失敗させるか、業務処理を優先して補償対象にするか。
6. 失敗した振込依頼をどこまで `transfer_requests` に保存するか。特に権限不足、存在しない振込先、残高不足、システム障害を同じ扱いにしない方針が必要。

## テスト・確認

- `git status --short`: 作業開始時点では未コミット差分なしを確認した。成果物作成後の最終確認では、指定外の `docs/ai/cycles/2026-06-28-001/code-reviewer.md`、`docs/ai/cycles/2026-06-28-001/security-reviewer.md`、`docs/ai/cycles/2026-06-28-005/` も作業ツリーに表示されたが、本 agent はそれらを変更していない。
- 本 agent の書き込み対象は `docs/ai/cycles/2026-06-28-001/banking-reviewer.md` のみ。
- `go test ./...`: 実行を試みたが、この環境では `go` コマンドが見つからず実行できなかった。

## 総評

現時点の実装は health check と HTTP server 設定だけで、金融ドメイン状態を変更しないため、直接的な元帳・残高事故リスクは増えていない。次に価値が高いのは、業務 API や DB schema に進む前に、取引種別の残高方向、`balance_after`、追記型取引履歴、冪等性キー衝突、並行更新時の残高保護、監査ログ境界を小さく設計文書化することである。
