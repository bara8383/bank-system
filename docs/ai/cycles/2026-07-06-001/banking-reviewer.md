# banking-reviewer: 2026-07-06-001

## レビュー前提

- 役割: `banking-reviewer` として、残高、元帳、取引履歴、冪等性、状態遷移、金融事故リスクを確認した。
- 指示確認: `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md`、`.agents/skills/banking-ledger-review/SKILL.md`、同 skill の品質ルーブリックを確認した。
- agent 間の直接同期は行わず、同一 cycle の `planner.md` と `implementer.md`、実装差分、既存 docs を入力として扱った。
- 実装差分優先で、`HEAD^..HEAD` の変更を確認した。対象差分は `README.md`、`docs/security-notes.md`、`internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`docs/ai/cycles/2026-07-06-001/implementer.md`。
- 本レビューではソースコード、設計文書、README は変更せず、この Markdown 成果物のみを作成した。

## 参照した入力

- cycle artifact:
  - `docs/ai/cycles/2026-07-06-001/planner.md`
  - `docs/ai/cycles/2026-07-06-001/implementer.md`
- 実装差分:
  - `git diff --name-status HEAD^..HEAD`
  - `git diff --find-renames HEAD^..HEAD -- README.md docs/security-notes.md internal/domain/failure_reason.go internal/domain/failure_reason_test.go docs/ai/cycles/2026-07-06-001/implementer.md`
- 既存設計・品質入力:
  - `docs/START_HERE.md`
  - `docs/design-principles.md`
  - `docs/domain-model.md`
  - `docs/data-model.md`
  - `docs/use-cases.md`
  - `docs/mvp.md`
  - `docs/security-notes.md`
  - `docs/test-strategy.md`
  - `docs/ai/output/human/001-human-review.md`
  - `docs/ai/output/human/002-human-review.md`
- 既存 domain 実装:
  - `internal/domain/money.go`
  - `internal/domain/account.go`
  - `internal/domain/transaction.go`

## Finding

### BR-2026-07-06-001-01: Blocking finding なし — safe failure category helper は残高・元帳・取引履歴の整合性を悪化させていない

- 重要度: Info / Blocking なし
- 種別: 実装差分レビュー
- 対象: `internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`README.md`、`docs/security-notes.md`

#### 根拠

- 今回の code 差分は domain sentinel error を固定の `FailureReason` に写像する pure Go helper とテストの追加であり、`Amount`、`Balance`、`AddBalance`、`SubtractBalance`、`ApplyTransaction` の残高計算や既存 sentinel error の意味は変更していない。
- `FailureReasonFromError` は `errors.Is` により既知の domain error のみを固定分類へ写像し、未知 error や `nil` では空文字と `false` を返すため、raw `Error()` 文字列を監査 `failure_reason` や将来の API 応答へ流す挙動になっていない。
- `FailureReason.Validate()` は定義済み constants だけを許可し、空文字、未知値、既存 sentinel error の raw message、secret 風の自由入力値を拒否するテストが追加されている。
- README と security notes は、helper を「将来の利用者応答・監査ログ・安全な構造化ログで使う分類候補」と位置付けつつ、HTTP error response、監査ログ永続化、DB schema が未実装であることを明記している。
- 既存設計原則は、残高変更成功時に口座残高・取引履歴・成功監査ログを同一 PostgreSQL transaction に含め、失敗監査ログは独立 transaction で残す方針を定めている。今回の helper はその前段の失敗理由分類であり、残高更新や取引履歴永続化を単独で行わない。
- `go test ./...` と `git diff --check HEAD^..HEAD` は成功した。

#### 影響

- 残高:
  - 今回差分は残高計算 helper を変更していないため、入金・出金・振込の残高増減方向、残高不足時に元残高を返す挙動、overflow 検出には直接影響しない。
  - `ErrBalanceMustBeNonNegative` が `invalid_balance_state` に分類され、将来の service / repository / DB 境界で「利用者入力不備」と「内部・永続化境界の残高状態異常」を分けて扱う土台になる。
- 元帳・取引履歴:
  - helper は transaction row、`balance_after`、関連取引、追記型履歴を作成しないため、元帳完成と誤認してはいけない。
  - 一方で、残高不足、invalid amount、invalid transaction type などの失敗分類が固定されたことで、将来の失敗監査ログや transfer request の `failure_reason` に raw error ではなく安全な分類値を保存しやすくなる。
- 冪等性:
  - 冪等性キー処理は未実装のままであり、今回差分は二重入金・二重出金・二重送金を防止しない。
  - ただし、将来の冪等性重複拒否にも同様の safe failure category が必要になるため、分類 helper の方針は後続設計と整合する。
- 状態遷移:
  - `ErrAccountNotActive` と `ErrInvalidAccountStatus` の分類が分離されており、停止中・解約済み口座への残高変更拒否と、不正な status 値の拒否を将来の監査分類で区別できる。
  - suspended / closed の詳細を `account_not_active` にまとめる作業仮定は、外部応答で口座状態詳細を出しすぎない安全側の判断として妥当。
- 金融事故リスク:
  - 今回差分は資金移動処理を追加していないため、直接の二重送金、片側だけ成功する振込、残高不整合、取引履歴欠落の新規事故経路は増えていない。
  - むしろ raw request body、password、token、secret、CSRF token、session ID、未加工の自由入力値を failure reason として保存・返却しない方針を code/docs/test で補強しており、監査ログ実装前の安全性が改善している。

#### 推奨修正

- 今回差分に対する blocking 修正は不要。
- 次に入出金・振込 service へ進む前に、今回追加された `FailureReason` を使う境界を明確化すること。
  - domain error から API error response、audit `failure_reason`、transfer request `failure_reason`、構造化ログ分類への写像責務を service / adapter のどこに置くかを決める。
  - `FailureReasonFromError` が `false` を返す未知 error を、利用者応答・監査ログ・運用ログでどう扱うかを決める。未知 error の raw message を外部応答や監査 `failure_reason` に保存しない方針は維持する。
- `FailureReason` を安全な分類として使う場合でも、残高変更の成功判定、取引履歴作成、監査ログ作成、DB transaction commit の順序をこの helper に寄せず、service / repository / transaction manager 側で明示すること。

## 観点別確認

| 観点 | 確認結果 | 残リスク |
| --- | --- | --- |
| 残高 | 残高計算 code は変更なし。failure reason helper は残高を更新しない。 | 金額上限、残高上限、日次上限は未確定。DB 制約も未実装。 |
| 元帳 | transaction row や ledger persistence は追加されていないため、元帳完成とは扱わない。 | `transactions.balance_after` と `accounts.balance_amount` の同一 transaction 更新は未実装。 |
| 取引履歴 | 取引履歴作成は未実装。今回差分は失敗分類の土台に限定。 | 入金・出金・振込 service 実装時に履歴欠落を防ぐ transaction 境界が必要。 |
| 冪等性 | 冪等性キー処理は未実装。 | human note の「操作種別、送信元口座、ログインユーザー、request body hash」を含む一意範囲と、MVP で重複時拒否する failure reason が未決定。 |
| 状態遷移 | invalid status と not active の分類分離は妥当。 | 口座 lifecycle の状態遷移表、残高あり解約、未完了振込依頼がある口座の解約可否は未確定。 |
| 金融事故リスク | 今回差分で直接の資金移動事故経路は増えていない。 | service / DB / audit / idempotency が未実装のため、業務 API を追加する前に gate 順序と transaction 境界が必要。 |

## 事故シナリオメモ

- シナリオ: 将来の handler が unknown error の `err.Error()` をそのまま `failure_reason` として保存する。
  - 今回の helper 自体はこれを行わないが、caller が `ok == false` を無視して raw message を代替値にすると、DB 接続文字列、個人情報、自由入力値、内部状態が監査ログや API response に混入する可能性がある。
  - 対策: unknown error 用の safe category を導入するか、audit/log adapter で raw error と public/audit category を分離する。
- シナリオ: 入出金 service が `FailureReasonFromError` の分類だけを返して、残高更新・取引履歴・監査ログの transaction 境界を実装済みと誤解する。
  - 今回 helper は分類のみで、元帳・履歴・冪等性・監査保存を提供しない。
  - 対策: service 実装前に gate 順序、DB transaction 境界、成功/失敗監査ログ境界を docs または accepted scope に固定する。

## 次サイクル planner への入力

1. 入出金・振込 service 実装前に、認証、CSRF、owner / role authorization、口座存在確認、口座 status gate、金額 validation、取引種別 validation、冪等性確認、DB row lock、残高計算、取引履歴作成、成功監査ログ、失敗監査ログの順序を設計 scope として採択すること。
2. `FailureReason` の利用境界を次 cycle で明確化すること。特に、API response 用 code、audit `failure_reason`、transfer request `failure_reason`、structured log category を同一分類にするか、用途別に薄い adapter を置くかを決めること。
3. unknown error の取り扱いを決めること。raw error message を保存・返却しない原則を保ちつつ、運用調査に必要な correlation ID、safe internal category、詳細ログの保管場所を分ける案が必要。
4. human note に従い、冪等性キーは操作種別、送信元口座、ログインユーザー、request body hash を含む一意範囲として設計し、MVP では重複時に既存結果返却ではなく拒否する方針を accepted scope 化すること。
5. PostgreSQL 行ロックによる悲観ロック方針を、入出金・振込ごとの lock 対象と順序に落とすこと。特に 2 口座振込では deadlock 回避のための lock 順序を明示すること。
6. 口座 lifecycle の状態遷移表を作り、`active` / `suspended` / `closed` の許可操作、残高あり解約、未完了振込依頼がある口座の解約可否を後続実装前に明確化すること。
7. 金額上限、残高上限、日次上限は引き続き未確定であり、業務 API 接続前に少なくとも学習用の暫定上限を作業仮定として置くかを検討すること。

## 実行した確認

- `git status --short`: 作業開始時は未コミット変更なし。
- `rg --files -uu -g 'AGENTS.md'`: repo 内の AGENTS scope は root のみであることを確認。
- `git diff --name-status HEAD^..HEAD`: 実装差分対象を確認。
- `git diff --find-renames HEAD^..HEAD -- README.md docs/security-notes.md internal/domain/failure_reason.go internal/domain/failure_reason_test.go docs/ai/cycles/2026-07-06-001/implementer.md`: 実装差分内容を確認。
- `go test ./...`: 成功。
- `git diff --check HEAD^..HEAD`: 成功。
