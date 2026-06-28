# banking-reviewer: 2026-06-28-005

## レビュー対象

- 対象 cycle: `2026-06-28-005`
- 役割: `banking-reviewer`
- 対応 skill: `.agents/skills/banking-ledger-review`
- レビュー種別: repo-wide review

## 確認した入力

- `git status --short`: 作業開始時点では未コミット変更なし。
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
- `docs/security-notes.md`
- `docs/test-strategy.md`
- `docs/ai/output/human/`: ディレクトリなし。追加 human notes は確認できなかった。
- 同一 cycle `docs/ai/cycles/2026-06-28-005/`: 作業開始時点で Markdown 成果物なし。
- 過去 cycle の planner / implementer / reviewer 成果物、特に `2026-06-28-004`
- 現在の Go 実装: `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go`

## 実装差分の有無

- `git status --short` と `git diff --stat` は空で、実装差分は確認できなかった。
- 同一 cycle の他 agent 成果物も作業開始時点では存在しなかった。
- したがって、今回は差分レビューではなく repo-wide review を行う。

## Finding

### Finding 1: 現在の実装は業務データを扱わず、直接の元帳・残高事故リスクは発火しない

#### 根拠

- `README.md` は、現在の実装範囲を Go 標準ライブラリだけの最小 REST API server と `GET /healthz` に限定し、DB 接続、認証、業務 API は未導入としている。
- `internal/httpapi/router.go` は `/healthz` の固定 JSON 応答のみを提供している。
- `cmd/server/main.go` は HTTP server 起動と timeout 設定だけを扱い、口座、残高、取引履歴、振込依頼、監査ログを扱わない。
- `README.md` の未実装一覧には、入金、出金、振込、残高照会、取引履歴照会、PostgreSQL、transaction 処理、監査ログ、冪等性キー処理が明記されている。

#### 影響

- 現時点では、二重送金、残高マイナス、片側だけ成功する振込、取引履歴欠落、履歴改変といった金融事故は実装上は発生しない。
- 一方で、MVP の中核である残高更新、取引履歴、振込、監査ログ、冪等性はまだ設計を実装可能な粒度まで落とし切れていない。

#### 推奨修正

- 現在の Go 実装に対する修正は不要。
- 業務 API または DB schema に進む前に、元帳・残高更新・取引履歴・冪等性・監査ログ境界を docs scope として先に具体化する。

#### 次サイクル planner への入力

- 次 cycle では、実装より先に `docs/design-principles.md`、`docs/data-model.md`、`docs/test-strategy.md` の元帳・残高ルール具体化を accepted scope 候補にする。

### Finding 2: cycle 004 で採択された元帳・残高方向の docs 具体化が現行 docs に未反映

#### 根拠

- `docs/ai/cycles/2026-06-28-004/planner.md` は、`transaction_type` ごとの残高増減方向、`balance_after` の意味、成功時 transaction 境界を docs に具体化する scope を採択していた。
- 一方、`docs/ai/cycles/2026-06-28-004/implementer.md` は、実行時点で accepted scope が見つからなかったため `blocked: accepted scope not found` と記録している。
- 現行 `docs/data-model.md` は `transaction_type` に `deposit`, `withdrawal`, `transfer_debit`, `transfer_credit`, `reversal` を列挙しているが、それぞれの残高増減方向表はない。
- 同じく `transactions.balance_after` は「取引後残高」とだけ書かれており、「対象口座に当該取引を適用した直後の残高」か、振込全体後の残高かが明文化されていない。
- `docs/design-principles.md` は同一トランザクション方針を掲げるが、入金、出金、振込それぞれで同一 DB transaction に含めるべき更新単位はまだ抽象的である。

#### 影響

- 取引種別ごとの符号解釈が handler、repository、テスト、集計処理に分散すると、`accounts.balance_amount` と `transactions.balance_after` の不一致を検出しにくくなる。
- 振込では、振込元の `transfer_debit` と振込先の `transfer_credit` が別々の口座に対する取引であることが曖昧なまま実装されると、明細表示や残高再計算で誤った集計をする可能性がある。
- `reversal` の方向や利用条件が未確定のまま schema や API に入ると、取消・訂正を履歴更新や削除で表現してしまうリスクが残る。

#### 事故シナリオ

1. `transactions.amount` を常に正の整数で保存するが、`withdrawal` と `transfer_debit` の減算ルールが共通化されていない。
2. 残高再計算やテストでは全取引を加算してしまい、実際の `accounts.balance_amount` と履歴合計が一致しない。
3. 問い合わせや監査時に、どの取引で残高が増減したかを説明できない。

#### 推奨修正

- `docs/data-model.md` に `transaction_type` ごとの残高方向表を追加する。
  - `deposit`: 対象口座の残高を増やす。
  - `withdrawal`: 対象口座の残高を減らす。
  - `transfer_debit`: 振込元口座の残高を減らす。
  - `transfer_credit`: 振込先口座の残高を増やす。
  - `reversal`: MVP 初期で実装対象に含めるか未確定として分離する。
- `balance_after` は、その取引を対象口座へ適用した直後の残高であり、0 以上であると明記する。
- `docs/design-principles.md` に、成功した残高変更では残高更新と取引履歴作成が同一 DB transaction に入ることを、入金・出金・振込別に書く。

#### 次サイクル planner への入力

- cycle 004 の accepted scope を再採択するか、同等の docs-only scope として再提示する。
- `reversal`、取消、組戻しの詳細は人間確認事項として残し、初期 MVP では確定済みルールに混ぜない。

### Finding 3: 冪等性キーの一意スコープと同一キー異内容時の扱いが未確定

#### 根拠

- `docs/design-principles.md` は、振込依頼に冪等性キーを持たせ、同じキーの処理結果を再利用するとしている。
- `docs/data-model.md` は、`transfer_requests.idempotency_key` を依頼者または振込元口座の範囲で一意にする案に留めている。
- `docs/use-cases.md` は、同じ冪等性キーの成功済み依頼がある場合は既存結果を返すとしているが、同じキーで金額、振込元、振込先、通貨が異なる場合の扱いは明記していない。

#### 影響

- 同一キーを異なる内容に再利用した場合に既存結果を返すだけだと、利用者が意図しない振込結果を正常応答として受け取る可能性がある。
- 一意制約違反だけで処理すると、利用者向けエラー、失敗監査ログ、失敗した振込依頼を保存するかどうかが実装ごとにばらつく。
- 二重送金防止と事故調査の両方に必要な「同じ依頼とは何か」が曖昧になる。

#### 推奨修正

- 「同じスコープ、同じ冪等性キー、同じリクエスト内容なら同じ結果を返す」と定義する。
- 「同じキーだがリクエスト内容が異なる場合は冪等性キー衝突として拒否し、監査対象にする」と定義する。
- 比較対象として、操作種別、振込元、振込先、金額、通貨、または request hash を保存する方針を検討する。

#### 次サイクル planner への入力

- 冪等性キーの一意スコープを、`requested_by_user_id + idempotency_key`、`source_account_id + idempotency_key`、操作種別込みの複合条件などから選ぶ人間確認事項として残す。
- 同一キー異内容時の拒否仕様と監査記録を docs scope 候補にする。

### Finding 4: 並行出金・並行振込時の残高保護方式が未確定

#### 根拠

- `docs/design-principles.md` は、残高をマイナスにしないこと、振込を原子的に扱うことを定めている。
- `docs/test-strategy.md` は、将来追加するテストとして並行実行時の残高競合テストを挙げている。
- PostgreSQL を前提としているが、行ロック、条件付き UPDATE、分離レベル、楽観ロック、複数口座のロック順序は未定義である。

#### 影響

- 単純な「残高読取、残高不足判定、残高更新」の実装では、同一口座への同時出金や同時振込で lost update が起き得る。
- 残高非負という最重要原則を破る実装になり、学習用であっても金融事故シナリオとして危険な土台になる。

#### 事故シナリオ

1. 口座 A の残高が 10,000 円。
2. 8,000 円の振込依頼が同時に 2 件処理される。
3. 両方が更新前残高 10,000 円を読んで成功可能と判定する。
4. 排他制御がなければ、2 件とも成功扱いになり、実質 16,000 円の出金を許す。

#### 推奨修正

- 初期方針として、トランザクション内で対象口座行をロックしてから残高判定・更新する方式、または `balance_amount >= amount` を条件にした単一 UPDATE の更新件数で判定する方式のどちらかを選ぶ。
- 振込で 2 口座を扱う場合は、口座 ID などでロック順序を固定し、デッドロック回避方針を docs に残す。

#### 次サイクル planner への入力

- 出金・振込の残高競合制御方式を docs scope 候補にする。
- `docs/test-strategy.md` に、同一口座への並行出金・並行振込で残高がマイナスにならないことを具体テストとして追加する。

### Finding 5: 監査ログの成功・失敗時境界が未整理

#### 根拠

- `docs/design-principles.md` と `docs/mvp.md` は、重要操作には監査ログを残すとしている。
- `docs/data-model.md` は `audit_logs` に actor、action、target、result、failure_reason、IP、User-Agent を持たせる案を示している。
- 残高更新、取引履歴、振込依頼更新、成功監査ログを同一 DB transaction に含めるか、業務拒否やシステム障害の失敗ログをどう残すかは未確定である。

#### 影響

- 成功時監査ログを業務 transaction 外で書くと、残高は変わったが監査ログが欠落する状態が起き得る。
- 失敗時監査ログを業務 transaction 内だけで書くと、rollback により拒否や障害の証跡が消える可能性がある。
- 監査ログ書き込み失敗時に業務処理を止めるか進めるかが未定だと、実装ごとに説明責任の水準が変わる。

#### 推奨修正

- 成功、業務拒否、システム障害を分けて、監査ログの保存境界を定義する。
- 残高変更を伴う成功操作では、残高更新、取引履歴、振込依頼状態、成功監査ログの整合性をどう担保するかを設計する。
- 失敗監査ログは、業務 transaction の rollback と独立して残す必要があるかを人間確認事項にする。

#### 次サイクル planner への入力

- 監査ログの transaction 境界を docs scope 候補にする。
- 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するかを人間確認事項にする。

## 人間確認事項

1. `reversal`、取消、組戻しを MVP 初期の実装対象に含めるか。含めない場合、初期 MVP では「既存取引の訂正・取消は未対応」と明示してよいか。
2. 並行出金・並行振込時の残高保護を、PostgreSQL の行ロック、条件付き UPDATE、または別方式のどれで学習・実装するか。
3. 冪等性キーの一意スコープを、操作種別、送信元口座、ログインユーザー、リクエスト本文 hash のどこまで含めるか。
4. 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するか。
5. 失敗した振込依頼をどこまで `transfer_requests` に保存するか。特に権限不足、存在しない振込先、残高不足、システム障害を同じ扱いにしない方針が必要。

## テスト・確認

- `git status --short`: 作業開始時点では未コミット変更なし。
- `git diff --stat`: 実装差分なし。
- `go test ./...`: 実行を試みたが、この環境では `go` コマンドが見つからず実行できなかった。

## 総評

現時点の実装は health check と HTTP server 設定だけで、金融ドメイン状態を変更しないため、直接的な元帳・残高事故リスクは増えていない。次に価値が高いのは、cycle 004 で採択されたが未反映の「元帳・残高方向・成功時 transaction 境界」の docs 具体化を再度小さく採択し、DB schema や業務 API 実装の前提を揃えることである。
