# banking-reviewer: 2026-06-28-004

## レビュー対象

- 対象 cycle: `2026-06-28-004`
- 役割: `banking-reviewer`
- 対応 skill: `.agents/skills/banking-ledger-review`
- レビュー種別: repo-wide review

## 確認した入力

- `git status --short`
- `.codex/agents/README.md`
- `docs/ai/cycles/README.md`
- `AGENTS.md`
- `README.md`
- `docs/START_HERE.md`
- `docs/design-principles.md`
- `docs/domain-model.md`
- `docs/data-model.md`
- `docs/use-cases.md`
- `docs/mvp.md`
- `docs/test-strategy.md`
- 過去 cycle の `banking-reviewer.md`
- `docs/ai/cycles/2026-06-28-003/planner.md`
- `docs/ai/cycles/2026-06-28-003/implementer.md`
- `docs/ai/cycles/2026-06-28-004/implementer.md`
- 現在の Go 実装: `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go`

## 実装差分の有無

- `git status --short` では `docs/ai/cycles/2026-06-28-004/` が未追跡として表示された。
- 同一 cycle の `implementer.md` は `blocked: accepted scope not found` を記録しており、ソースコード、README、設計文書、DB schema、migration は変更していない。
- `git diff --stat` は空で、実装差分は確認できなかった。
- したがって、今回は差分レビューではなく repo-wide review を行う。

## Finding

### Finding 1: 実装はまだ業務データを扱わず、現時点の元帳・残高事故リスクは発火しない

#### 根拠

- `README.md` は、現在の実装範囲を Go 標準ライブラリのみの最小 REST API server と `GET /healthz` に限定している。
- `cmd/server/main.go` は HTTP server の起動、listen address、timeout 設定だけを扱い、DB 接続、口座、残高、取引履歴、監査ログを扱わない。
- `internal/httpapi/router.go` は `/healthz` の固定 JSON 応答だけを提供している。
- `docs/ai/cycles/2026-06-28-004/implementer.md` は accepted scope 不在のため blocked とし、実装しなかったことを記録している。

#### 影響

- 今回の状態では、二重送金、残高マイナス、片側だけ成功する振込、取引履歴欠落、取消時の履歴改変といった金融事故は発火しない。
- 一方で、MVP の中核である入金、出金、振込、残高照会、取引履歴照会、監査ログ、冪等性処理は未実装であり、設計の具体化が次の事故予防策になる。

#### 推奨修正

- 今回の実装に対する修正は不要。
- 次 cycle の planner は、業務 API や DB schema に進む前に、元帳・残高更新・取引履歴・冪等性・監査ログ境界を小さな docs scope として採択することを検討する。

#### 次サイクル planner への入力

- `transaction_type` ごとの残高増減方向を確定する。
- `balance_after` の意味を「口座ごとの当該取引反映後残高」として明文化するか検討する。
- `accounts.balance_amount` と `transactions` の整合性検証ルールを docs に追加する。

### Finding 2: 元帳モデルは追記型方針を持つが、取引種別と取消の意味がまだ実装可能な粒度ではない

#### 根拠

- `docs/design-principles.md` は、残高変更には必ず取引履歴を作成し、取消が必要な場合は取消取引を追加するとしている。
- `docs/data-model.md` は `transactions.transaction_type` に `deposit`, `withdrawal`, `transfer_debit`, `transfer_credit`, `reversal` を候補として挙げている。
- しかし、`reversal` が元取引の逆方向を表すのか、訂正・組戻し・取消をどう区別するのか、MVP で実装対象に含めるのかは未確定である。

#### 影響

- 取消や訂正を既存取引の更新・削除で表現してしまうと、取引履歴の不可逆性と監査可能性が損なわれる。
- `reversal` の意味が曖昧なまま実装されると、明細表示、残高再計算、事故調査で、どの取引がどの取引を打ち消したのか説明しにくくなる。

#### 事故シナリオ

1. 誤った入金を取り消すため、既存の入金取引を削除または金額更新する。
2. `accounts.balance_amount` は手作業で修正されるが、元の入金が存在した事実が取引履歴から消える。
3. 利用者問い合わせや監査時に、なぜ残高が変化したかを時系列で説明できない。

#### 推奨修正

- MVP で取消を実装しない場合でも、「履歴は削除・更新せず、将来の取消は追記取引で表現する」ことを docs に明記する。
- `reversal` を使う場合は、取消対象 `related_transaction_id`、残高方向、取消可能な元取引種別、二重取消防止を最小限定義する。

#### 次サイクル planner への入力

- `reversal` を MVP 実装対象に含めるか、人間確認事項として分離する。
- 少なくとも取引履歴の不可変性と、取消は追記で扱う方針を `docs/data-model.md` または専用設計文書に追加する候補を作る。

### Finding 3: 冪等性キーのスコープと同一キー異内容時の扱いが未確定

#### 根拠

- `docs/design-principles.md` は、同じ振込依頼が複数回処理されないよう、冪等性キーで同じキーの処理結果を再利用するとしている。
- `docs/data-model.md` は `transfer_requests.idempotency_key` を依頼者または振込元口座の範囲で一意にする案を示している。
- 同じ冪等性キーで、金額、振込元、振込先、通貨が異なるリクエストが来た場合の扱いは未確定である。

#### 影響

- 同一キーを異なる内容に再利用したときに既存結果を返すだけだと、利用者が意図しない振込結果を正常応答として受け取る可能性がある。
- 一意制約違反だけで扱うと、利用者向けエラー、監査ログ、失敗した振込依頼を保存するかどうかがばらつき、二重送金防止と事故調査の両方が弱くなる。

#### 推奨修正

- 「同じスコープ、同じ冪等性キー、同じリクエスト内容なら同じ結果を返す」と定義する。
- 「同じキーだが内容が異なる場合は冪等性キー衝突として拒否し、監査対象にする」と定義する。
- 比較対象として、振込元、振込先、金額、通貨、操作種別、または request hash を保存する方針を検討する。

#### 次サイクル planner への入力

- 冪等性キーの一意スコープを `requested_by_user_id + idempotency_key`、`source_account_id + idempotency_key`、または別案から選ぶ人間確認事項として残す。
- 同一キー異内容時の拒否仕様と監査記録を docs scope 候補にする。

### Finding 4: 並行出金・並行振込時の残高保護方式が未確定

#### 根拠

- `docs/design-principles.md` は、残高をマイナスにしないこと、振込を原子的に扱うことを基本原則にしている。
- `docs/test-strategy.md` は、将来追加するテストとして並行実行時の残高競合テストを挙げている。
- PostgreSQL を前提にする一方、行ロック、条件付き UPDATE、分離レベル、楽観ロック、ロック順序はまだ決まっていない。

#### 影響

- 単純な「残高読取、残高不足判定、更新」の実装では、同一口座への同時出金や同時振込で lost update が起き得る。
- 学習用であっても、残高非負という最重要原則を破る実装になり、金融事故シナリオの説明教材としても危険な形になる。

#### 事故シナリオ

1. 口座 A の残高が 10,000 円。
2. 8,000 円の振込依頼が同時に 2 件処理される。
3. 両方が更新前残高 10,000 円を読んで成功可能と判定する。
4. 排他制御がなければ、2 件とも成功扱いになり、実質 16,000 円の出金を許す。

#### 推奨修正

- 初期方針として、トランザクション内で対象口座行をロックしてから残高判定・更新する方式、または `balance_amount >= amount` を条件にした単一 UPDATE の更新件数で判定する方式のどちらかを選ぶ。
- 振込で複数口座を扱う場合は、口座 ID などでロック順序を固定し、デッドロック回避方針を docs に残す。

#### 次サイクル planner への入力

- 出金・振込の残高競合制御方式を docs scope 候補にする。
- MVP テスト戦略に、同一口座への並行出金・並行振込で残高がマイナスにならないことを具体テストとして追加する。

### Finding 5: 監査ログの成功・失敗時境界が未整理

#### 根拠

- `docs/design-principles.md` と `docs/mvp.md` は、重要操作には監査ログを残すとしている。
- `docs/data-model.md` は `audit_logs` に actor、action、target、result、failure_reason、IP、User-Agent を持たせる案を示している。
- 残高更新、取引履歴、振込依頼更新、成功監査ログを同一 DB transaction に含めるか、業務拒否やシステム障害の失敗ログをどう残すかは未確定である。

#### 影響

- 成功時監査ログを業務トランザクション外で書くと、残高は変わったのに監査ログが欠落する可能性がある。
- 失敗時監査ログを業務トランザクション内だけで書くと、ロールバックにより拒否・障害の証跡が消える可能性がある。
- 監査ログ書き込み失敗時に業務処理を止めるか進めるかが未定だと、実装ごとに事故時の説明責任が変わる。

#### 推奨修正

- 成功、業務拒否、システム障害を分け、監査ログの保存境界を決める。
- 残高変更を伴う成功操作では、残高更新、取引履歴、振込依頼状態、成功監査ログの整合性をどう担保するかを設計する。
- 失敗監査ログは、ロールバック後でも証跡が残る方針を検討し、人間確認事項として残す。

#### 次サイクル planner への入力

- 監査ログの transaction 境界を docs scope 候補にする。
- 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するかを人間確認事項にする。

## 人間確認事項

1. MVP で `reversal` を実装対象に含めるか、初期は「取消未実装だが履歴は不可変」と明記するだけにするか。
2. 冪等性キーの一意スコープを、依頼者、振込元口座、操作種別、リクエスト本文 hash のどこまで含めるか。
3. 並行出金・並行振込の残高保護方式を、PostgreSQL 行ロック、条件付き UPDATE、または別方式のどれにするか。
4. 監査ログ書き込み失敗時、残高変更を伴う業務処理を失敗させるか、業務処理を優先して補償対象にするか。
5. 失敗した振込依頼をどこまで `transfer_requests` に保存するか。特に権限不足、存在しない振込先、残高不足、システム障害を同じ扱いにしない方針が必要。

## テスト・確認

- `git status --short`: 実装差分はなく、cycle 004 成果物ディレクトリのみ未追跡であることを確認した。
- `git diff --stat`: 空であることを確認した。
- `go test ./...`: 実行を試みたが、この環境では `go` コマンドが見つからず実行できなかった。

## 総評

現時点の実装は health check と HTTP server 設定だけで、金融ドメイン状態を変更しないため、直接的な元帳・残高事故リスクは増えていない。次に価値が高いのは、業務 API や DB schema に進む前に、取引種別の残高方向、追記型取引履歴、冪等性キー衝突、並行更新時の残高保護、監査ログ境界を小さく設計文書化することである。
