# banking-reviewer: 2026-07-01-001

## レビュー範囲

- 実装差分レビューを優先し、直近 commit `81f8a98 Add account status domain helper` の変更範囲を確認した。
- 参照した cycle 成果物は `docs/ai/cycles/2026-07-01-001/planner.md` と `docs/ai/cycles/2026-07-01-001/implementer.md` のみ。agent 間の直接同期は行っていない。
- 確認対象は、口座ステータス helper が残高、元帳、取引履歴、冪等性、状態遷移、銀行ドメイン、金融事故リスクに与える影響。
- 今回の実装は pure domain helper であり、DB schema、残高更新 service、元帳/取引履歴永続化、監査ログ、冪等性キーはまだ未実装のまま。

## Finding 1: blocking なし。残高変更 gate としての口座ステータス判定は accepted scope に合致している

### 根拠

- `AccountStatus` は MVP の `active` / `suspended` / `closed` を明示し、docs 上の「有効」「停止中」「解約済み」への対応をコメントで示している。
- `AccountStatus.Validate()` は `active` / `suspended` / `closed` のみ valid とし、空文字・未知 status を `ErrInvalidAccountStatus` として拒否する。
- `EnsureAccountCanTransact` はまず status validation を行い、`active` だけを残高変更可能、`suspended` / `closed` を `ErrAccountNotActive` として拒否している。
- unit test は supported status、空文字・未知 status、active 許可、suspended / closed 拒否、unknown status と inactive account の error 分離を確認している。
- README は、口座作成 API / DB / 認証 / 監査ログ / 冪等性が未実装のままであることを維持しつつ、口座ステータス validation helper の追加を反映している。

### 影響

- 停止中・解約済み口座に対する入金、出金、振込などの残高変更操作を後続 service で一貫して拒否する土台ができた。
- unknown status を「停止中/解約済み」と同じ業務拒否に潰さないため、将来 DB mapper 不備、データ破損、想定外 enum 値を監査・運用上別分類にしやすい。
- 現時点では helper が導入されただけなので、残高、元帳、取引履歴、監査ログ、冪等性の整合性を実際に保証する runtime 経路はまだ存在しない。

### 推奨修正

- 今回差分に対する修正は不要。
- 後続で入金・出金・振込 service を追加する際は、金額 validation と残高演算の前、かつ DB transaction 内の対象口座 lock 後に `EnsureAccountCanTransact` 相当の口座状態 gate を必ず通す設計にする。
- DB から読み出した status は repository / mapper 境界でも `Validate()` し、未知 status を通常の業務拒否ではなくデータ不整合系 error として監査分類できるようにする。

### 次サイクル planner への入力

- 次 cycle で業務 API へ進む前に、口座状態 error を API response と audit `failure_reason` にどう mapping するかを設計候補に入れる。
- `ErrInvalidAccountStatus` はデータ不整合・実装不備寄り、`ErrAccountNotActive` は業務拒否寄りとして分ける前提を planner の入力にする。

## Finding 2: 状態遷移の許可表は未定義のため、将来の口座作成・停止・解約で事故シナリオが残る

### 根拠

- 今回の accepted scope は「現在の口座 status が残高変更可能か」を判定する helper に限定されており、口座状態そのものの遷移 helper は実装対象外だった。
- `active` / `suspended` / `closed` の値は定義されたが、`suspended -> active` の再開可否、`closed -> active` の禁止、残高あり `closed` の可否、解約済み口座への入金拒否後の返金/組戻し扱いは未定義。
- README と implementer 成果物でも、口座作成、永続 entity、DB schema、監査ログ、reversal / 取消は未実装と明記されている。

### 影響

- 将来、口座管理 API や admin 操作を追加する際に、状態遷移の許可/禁止が handler や service ごとに分散すると、解約済み口座を誤って復活させる、残高あり口座を解約する、停止中口座への振込を一部経路だけ許可する、といった金融事故につながる。
- 状態遷移が監査ログと結びつかない場合、なぜ残高変更が拒否されたか、誰がいつ停止/解約したかを後から追跡しにくくなる。

### 推奨修正

- 次に口座 lifecycle を扱う cycle では、状態遷移表を docs に明記してから domain helper に落とす。
- 最小案として、`active -> suspended`、`suspended -> active`、`active/suspended -> closed` を候補にし、`closed -> active/suspended` は禁止する案を検討する。
- `closed` への遷移条件として、残高 0、未完了 transfer request なし、必要な取引履歴・監査ログが永続化済みであることを検討する。

### 次サイクル planner への入力

- 口座ステータス値の validation が入ったため、次は「残高変更可否」ではなく「口座状態遷移の設計」を独立候補として扱える。
- ただし DB / 監査ログなしに遷移 API だけを作ると追跡不能な状態変更が発生するため、まず docs の状態遷移表または audit mapping の小 scope が適切。

## Finding 3: 元帳・取引履歴・冪等性の保証は今回差分では増えていないため、業務 API 追加前の優先課題として残る

### 根拠

- 今回の実装は `internal/domain` の status helper と test、README 更新に限定されている。
- 入金、出金、振込、残高照会、取引履歴照会、PostgreSQL 接続、DB schema、transaction 処理、監査ログ、冪等性キー処理は README 上も未実装のまま。
- `EnsureAccountCanTransact` は残高変更可否の前提条件を表すだけで、残高更新と取引履歴 insert を同一 transaction にする保証、二重実行防止、元帳の追跡性はまだ提供しない。

### 影響

- 今後 service / repository を追加する時に、口座状態 gate だけを実装済み安全策として過信すると、取引履歴なしの残高更新、残高更新なしの取引履歴、二重入金/二重出金、振込の片側成功などの金融事故リスクが残る。
- 特に振込では、from/to 口座双方の状態確認、lock 順序、同一 DB transaction、冪等性キー重複拒否、監査ログが揃わないと、今回の status helper だけでは事故を防げない。

### 推奨修正

- 業務 API 実装前に、少なくとも次のどちらかを次 cycle の小 scope として採択する。
  - domain error から API response / audit `failure_reason` への mapping 表を作る。
  - 口座・取引履歴・監査ログ・冪等性キーの DB 制約と transaction 境界を docs に具体化する。
- 入金・出金・振込 service 実装時は、`EnsureAccountCanTransact`、`Amount.Validate()`、`Balance.Validate()`、残高演算、transaction history insert、audit log insert、idempotency key check を一連の不変条件として扱う。

### 次サイクル planner への入力

- 今回の差分は口座状態 gate として妥当で blocking はないが、次に業務 API を直接増やすより、監査分類・DB transaction 境界・冪等性キー重複拒否のいずれかを先に固定する方が金融事故リスクを下げやすい。
- human notes では「MVP の冪等性キー重複は既存結果返却ではなく拒否でよい」とされているため、冪等性キー設計を小さく docs 化する候補は次 cycle に適している。

## 作業メモ

- 作業開始時に `git status --short` を確認し、表示は空だった。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、README、docs 配下の設計文書、human notes、同一 cycle の planner / implementer 成果物を確認した。
- ソースコード、README、planner.md、implementer.md は変更していない。書き込みはこの `docs/ai/cycles/2026-07-01-001/banking-reviewer.md` のみに限定した。
