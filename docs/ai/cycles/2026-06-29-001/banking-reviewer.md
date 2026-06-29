# banking-reviewer: 2026-06-29-001

## レビュー対象

- 入力 artifact:
  - `.codex/agents/README.md`
  - `docs/ai/cycles/README.md`
  - `docs/ai/cycles/2026-06-29-001/planner.md`
  - `docs/ai/cycles/2026-06-29-001/implementer.md`
  - `docs/ai/output/human/001-human-review.md`
- 参照した金融品質観点:
  - `.agents/skills/banking-ledger-review/SKILL.md`
  - `docs/START_HERE.md`
  - `docs/mvp.md`
  - `docs/domain-model.md`
  - `docs/data-model.md`
  - `docs/design-principles.md`
  - `docs/security-notes.md`
  - `docs/test-strategy.md`
- 実装差分: `HEAD^..HEAD` の docs-only 差分。
  - `docs/design-principles.md`
  - `docs/security-notes.md`
  - `docs/data-model.md`
  - `docs/test-strategy.md`
  - `docs/ai/cycles/2026-06-29-001/implementer.md`

今回の差分は、監査ログの記録要否、成功/失敗時の保存境界、監査ログ書き込み失敗時の fail closed、マスキング、閲覧権限、`transactions.balance_after >= 0`、rollback テスト観点を docs に整理するものだった。Go ソースコード、DB schema、migration、業務 API、認証/認可実装、冪等性キー詳細、PostgreSQL 行ロックは変更されていない。

## Finding 1: 修正必須の元帳・残高ブロッカーは確認しなかった

### 根拠

- planner の accepted scope は、監査ログ境界、失敗時扱い、閲覧権限、マスキング方針、および `transactions.balance_after >= 0` と rollback テスト観点の小補修に限定している。
- implementer は対象を `docs/design-principles.md`, `docs/security-notes.md`, `docs/data-model.md`, `docs/test-strategy.md`, `docs/ai/cycles/2026-06-29-001/implementer.md` に限定し、Go ソースコード、DB schema、migration、業務 API、認証/認可実装には踏み込んでいないと記録している。
- 実装差分では、残高変更を伴う成功操作の成功監査ログを同じ PostgreSQL データベーストランザクションに含める案、業務拒否や DB transaction 途中失敗の失敗監査ログを独立したデータベーストランザクションで残す案、成功監査ログが書けない残高変更や権限変更を fail closed にする案が docs 化された。
- `docs/data-model.md` には `transactions.balance_after` を 0 以上にする制約案が追加され、`docs/test-strategy.md` には成功時の業務データ・取引履歴・成功監査ログの整合、業務拒否時の失敗監査ログ、途中失敗 rollback、監査ログ書き込み失敗時 fail closed の確認観点が追加された。

### 影響

今回の docs-only 差分は、前 cycle で指摘された「失敗時監査ログが未確定に見える」「`balance_after >= 0` の制約案が不足」「rollback テスト観点が不足」というリスクを低減している。現時点では実際の口座残高、取引履歴、振込依頼、監査ログを更新するコードが追加されていないため、差分単体で二重送金、残高マイナス、片側振込成功、取引履歴欠落を発生させる直接リスクは増えていない。

### 推奨修正

この finding について、今回差分内での修正必須事項はない。次に DB schema や業務 API へ進む前に、今回 docs 化された transaction 境界を実装可能な粒度へ分解し、テーブル制約、repository 境界、テスト注入点に落とし込むことを推奨する。

### 次サイクル planner への入力

次 cycle で業務 API や migration に進む場合は、先に次のいずれかを小 scope として採択するのが安全である。

1. 振込依頼状態遷移と冪等性キー衝突時の扱い。
2. PostgreSQL 行ロックと 2 口座振込時のロック順序。
3. 口座別取引順序、`balance_after` 連続性、定期 reconciliation の検証方針。

## Finding 2: 失敗監査ログを独立保存する方針は改善だが、書き込み失敗時の事故シナリオは残る

### 根拠

- `docs/design-principles.md` は、業務拒否や DB transaction 途中失敗では業務データを更新しない、または rollback した上で、失敗監査ログを独立したデータベーストランザクションで残す方針に整理された。
- 同じ文書では、監査ログ書き込み失敗時について、成功監査ログが書けない残高変更や権限変更は MVP では fail closed としつつ、監査ログ自体の書き込み失敗をどう運用通知、再送、補償するかは将来検討としている。
- `docs/test-strategy.md` は、監査ログ書き込み失敗時に対象業務処理が成功扱いにならないことを確認する観点を追加しているが、失敗監査ログそのものが書けない場合の運用検知、再記録、重複防止、利用者への結果表現は未定義である。

### 影響

金融事故シナリオとして、DB 障害や監査ログ保存経路の障害が発生した場合、業務処理は fail closed で止まるが、止めた事実自体の監査ログも残らない可能性がある。これは残高不整合を直接発生させるよりは安全側だが、利用者から見た失敗、オペレーター調査、障害時の時系列復元が難しくなる。特にリトライが重なると、「実際には処理していないが失敗証跡もない」操作が増え、後続の冪等性設計や問い合わせ対応に影響する。

### 推奨修正

次 cycle 以降で、監査ログ保存失敗時の最小運用方針を docs に追加する。MVP では大きな outbox や外部 SIEM までは不要でも、少なくとも次を設計案として分けるとよい。

- 成功監査ログが書けない場合は業務処理を成功扱いにしない、という現方針を維持する。
- 失敗監査ログ自体が書けない場合の利用者向けエラー、サーバーログへの最小安全記録、運用通知、再試行対象化の有無を決める。
- 監査ログ保存失敗を再送する場合、同じ失敗証跡を二重記録しないための request id または event id を設計する。

### 次サイクル planner への入力

「監査ログ保存失敗時の運用・再送・重複防止」を単独 scope にするか、後続の API エラー応答/ログ標準 scope に含めることを検討する。DB schema 実装前に、`audit_logs` に request id / event id 相当を持たせるかどうかも候補に入れる。

## Finding 3: `request_body_hash` は監査ログに入ったが、冪等性・状態遷移との接続は未解決

### 根拠

- human notes は、冪等性の意味を持たせるために、操作種別、送信元口座、ログインユーザー、リクエスト本文 hash を含める方向を示している。
- 今回差分では、`docs/security-notes.md` と `docs/data-model.md` に raw request body を保存せず `request_body_hash` を使う設計案が追加された。
- 一方で、planner の非対象により、冪等性キーの複合一意制約、同一キー同一内容/異内容、処理中再送、保存期間は今回確定していない。
- `docs/data-model.md` の `transfer_requests` は `idempotency_key` を持つが、同一性検証に使う request hash、操作種別、冪等性 scope、衝突時に `transfer_requests` を作るか監査ログだけにするかはまだ未定義である。

### 影響

事故シナリオとして、同じ `idempotency_key` で金額や振込先が違う依頼が来た場合、将来の実装者が「一意制約で拒否するだけ」「既存結果を返す」「新規失敗依頼を残す」などを個別判断すると、二重送金防止と事故調査の両方が不安定になる。監査ログに request hash があっても、振込依頼側の状態遷移や一意制約と接続されていなければ、元帳と監査ログを突き合わせて「この再送は同一依頼か、衝突か」を機械的に説明しにくい。

### 推奨修正

次 cycle で `transfer_requests` の状態遷移と冪等性キー詳細を docs 化する。最低限、次を表で定義することを推奨する。

- 冪等性 scope: 例として `requested_by_user_id + source_account_id + action_type + idempotency_key` を候補にし、human notes の意図と照合する。
- request hash: どの正規化済み入力から hash を作るか、`transfer_requests` に保持するか、`audit_logs` のみに保持するか。
- 同一キー同一内容: MVP では拒否するのか、既存結果を返すのか。
- 同一キー異内容: 冪等性キー衝突として拒否し、失敗監査ログに何を残すか。
- `accepted` / `processing` / `succeeded` / `failed` の許可遷移、処理中クラッシュ時の扱い。

### 次サイクル planner への入力

「振込依頼状態遷移 + 冪等性キー衝突」を次 cycle の高優先 banking/security scope として採択候補にする。これは DB migration や振込 API より前に決めるべきである。

## Finding 4: `balance_after >= 0` は追加されたが、口座別の残高連続性・順序保証はまだ不足している

### 根拠

- 今回差分で `docs/data-model.md` の主な制約案に `transactions.balance_after` は 0 以上にすることが追加された。
- 既存 docs では、`balance_after` は対象口座に取引を適用した直後の残高で、対象口座の更新後残高と一致させると説明されている。
- `docs/test-strategy.md` には、`transactions.balance_after` が 0 未満にならないこと、取引履歴の増減と現在残高が矛盾しないことがある。
- ただし、同一口座内で取引をどの順序キーで並べるか、前回 `balance_after` と今回 `transaction_type` / `amount` から今回 `balance_after` を再計算する検証、同時刻取引や振込の debit / credit 関連付けをどう検証するかはまだ具体化されていない。

### 影響

`balance_after >= 0` と最新残高一致だけでは、途中の取引履歴欠落、二重記録、順序逆転を検出しにくい。たとえば入金 10,000 円、出金 3,000 円、振込出金 2,000 円の順序が壊れた場合でも、最終残高だけ合わせる実装になると、元帳として「どの時点でいくらだったか」を説明できない。これは顧客明細、事故調査、将来の reconciliation で問題になる。

### 推奨修正

次 cycle で、口座別取引順序と残高連続性の検証方針を docs に追加する。

- 口座ごとに取引を確定順で並べるための候補キーを検討する。例: `occurred_at` だけでなく、DB sequence / transaction id / created_at などの tie-breaker。
- `previous_balance_after + 増減方向 * amount = current_balance_after` を検証する観点を `docs/test-strategy.md` に追加する。
- 振込では `transfer_debit` と `transfer_credit` の関連付け、金額一致、通貨一致、同一 `transfer_request_id` の存在を検証する。
- 定期 reconciliation は将来検討でもよいが、DB schema に進む前に必要な列候補だけは洗い出す。

### 次サイクル planner への入力

「元帳順序・残高連続性・reconciliation 方針」を docs-only scope として検討する。冪等性/状態遷移 scope と並ぶ高優先候補だが、同一 cycle に入れると大きくなるため、どちらか一方に絞ることを推奨する。

## Finding 5: 並行更新制御は未確定のままで、DB 実装前の必須入力として残る

### 根拠

- human notes は、PostgreSQL の行ロックで悲観ロックを使う方向を示している。
- planner は、PostgreSQL 行ロック、ロック順序、デッドロック回避、分離レベルを今回の非対象にしている。
- `docs/design-principles.md` は、残高をマイナスにしないこと、振込元と振込先の更新を同じデータベーストランザクションに入れることを定義しているが、並行出金や 2 口座振込時のロック対象と取得順序はまだ未定義である。

### 影響

将来 DB 実装へ進んだとき、同一口座への同時出金・同時振込で lost update が起きると、どちらの処理も残高不足でないと判断してしまい、最終的に残高がマイナスになる事故が起こり得る。2 口座振込で振込元・振込先をリクエスト順にロックすると、A→B と B→A が同時に走ったときにデッドロックし、処理中断やリトライ時の状態遷移不整合につながる。

### 推奨修正

DB schema / repository / transaction manager 実装前に、PostgreSQL 行ロックの設計方針を docs 化する。

- 出金・振込元口座の残高確認と更新は同一 transaction 内でロックする。
- 2 口座振込ではロック順序を固定する。例: 内部 `account_id` 昇順で両口座をロックしてから残高確認・更新する。
- 残高照会がロック不要か、read committed の通常 read でよいかを明記する。
- デッドロック検出時の扱い、リトライするか失敗監査ログを残して拒否するかを状態遷移/冪等性設計と接続する。

### 次サイクル planner への入力

「PostgreSQL 行ロックによる残高保護方針」を DB 実装前の採択候補として残す。冪等性キーと状態遷移の docs 化が先に必要であれば、次々 cycle の候補として明示する。

## 総評

今回の差分は、前 cycle で banking-reviewer が指摘した失敗証跡の曖昧さをかなり解消している。成功時は残高更新・取引履歴・振込依頼状態・成功監査ログを同じ PostgreSQL transaction に含め、業務拒否や rollback 後の失敗監査ログは独立保存する、という整理は、学習用 MVP として安全側で一貫している。

一方で、次に実装へ進むには、まだ二重送金防止、処理中再送、同一キー異内容、口座別取引順序、並行残高更新のロック順序が未確定である。次 cycle planner は、DB schema や業務 API の前に、少なくとも「振込依頼状態遷移 + 冪等性キー衝突」または「PostgreSQL 行ロック + 残高保護」のどちらかを小さく採択することを推奨する。
