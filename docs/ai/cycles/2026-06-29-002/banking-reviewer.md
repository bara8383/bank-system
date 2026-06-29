# banking-reviewer: 2026-06-29-002

## レビュー対象

- 依頼ロール: `banking-reviewer`
- cycle: `2026-06-29-002`
- 主な入力 artifact:
  - `AGENTS.md`
  - `.codex/agents/README.md`
  - `docs/ai/cycles/README.md`
  - `README.md`
  - `docs/START_HERE.md`
  - `docs/mvp.md`
  - `docs/domain-model.md`
  - `docs/data-model.md`
  - `docs/design-principles.md`
  - `docs/security-notes.md`
  - `docs/test-strategy.md`
  - `docs/ai/output/human/001-human-review.md`
  - `docs/ai/cycles/2026-06-29-002/planner.md`
  - `docs/ai/cycles/2026-06-29-002/implementer.md`
- 実装差分: `HEAD^..HEAD` (`37b9d4d Add money domain validation`)
  - `README.md`
  - `docs/ai/cycles/2026-06-29-002/implementer.md`
  - `internal/domain/money.go`
  - `internal/domain/money_test.go`

## 確認した前提

- `git status --short` はレビュー開始時点で未コミット変更なし。
- agent 間の直接同期は禁止であるため、同一 cycle の `planner.md` と `implementer.md`、および repo 上の既存 docs / human notes だけを入力として扱った。
- 今回の accepted scope は「金額・残高の最小 domain 型と validation を Go に追加する」であり、DB schema、業務 API、取引履歴、監査ログ、冪等性キー、PostgreSQL 行ロックは非対象。
- 現行実装では、業務 API と DB がまだないため、今回差分だけで実際の口座残高、取引履歴、振込依頼、監査ログを永続化する処理は発生しない。

## 確認コマンド

- `git status --short`: レビュー開始時点で未コミット変更なし。
- `git log --oneline --decorate -n 8`: 最新 commit が `37b9d4d Add money domain validation` であることを確認。
- `git show --stat --patch --find-renames --find-copies HEAD -- README.md internal/domain/money.go internal/domain/money_test.go docs/ai/cycles/2026-06-29-002/implementer.md`: 実装差分を確認。
- `go test ./...`: 成功。
- `git diff --name-only HEAD^ HEAD && git diff --check HEAD^ HEAD`: 差分ファイルと whitespace error なしを確認。
- `rg -n "float32|float64" internal/domain || true`: `internal/domain` に浮動小数点型がないことを確認。

## 総評

今回の差分は、既存 docs の「金額は整数の最小通貨単位」「MVP は JPY のみ」「取引金額は正の整数」「残高は 0 以上」「残高不足時は残高を変えない」という前提を、Go の小さな domain helper と unit test に落としたものである。`Amount` / `Balance` は `int64` を内部値として保持し、`NewAmount` は 0 以下を拒否し、`NewBalance` は負値を拒否している。`AddBalance` は overflow を検出して元残高を返し、`SubtractBalance` は残高不足時に元残高を返すため、後続の入金・出金・振込実装で残高マイナス事故を避けるための下位部品として妥当である。

修正必須の元帳・残高ブロッカーは確認しなかった。一方で、今回の domain helper はまだ「メモリ上の金額・残高計算」に限定される。将来の DB / repository / 業務 API 実装では、DB 制約、取引履歴作成、監査ログ、冪等性、行ロック、口座別取引順序を別途実装しない限り、元帳品質は保証されない。また、Go の exported type として `Amount{}` のゼロ値は外部 package からも作れるため、後続実装では constructor 経由を徹底し、`Amount.Int64()` を直接 DB insert に使う境界の設計に注意が必要である。

## Finding 1: 修正必須の元帳・残高ブロッカーは確認しなかった

### 根拠

- `internal/domain/money.go` は `Amount` を `int64` の最小通貨単位として保持し、`NewAmount` で 0 以下の取引金額を `ErrAmountMustBePositive` として拒否している。
- `Balance` は `int64` の最小通貨単位として保持され、`NewBalance` は負の残高を `ErrBalanceMustBeNonNegative` として拒否している。
- `AddBalance` は amount が 0 以下の場合に元残高を返し、さらに `math.MaxInt64` を超える加算を `ErrBalanceOverflow` として拒否する。
- `SubtractBalance` は amount が 0 以下の場合と残高不足の場合に元残高を返し、残高をマイナスにしない API になっている。
- `internal/domain/money_test.go` は、正の取引金額、0 円・負の取引金額拒否、0 円・正の残高、負の残高拒否、加算、残高内減算、残高不足時の元残高維持、overflow 時の元残高維持を確認している。
- `go test ./...` は成功し、`rg -n "float32|float64" internal/domain` でも浮動小数点型の利用は確認されなかった。

### 影響

今回の差分は、入金・出金・振込実装前の最小 domain 土台として、次の金融事故リスクを下げている。

- 0 円または負の取引金額を残高変更に使うリスク。
- 残高不足の減算で残高が負になるリスク。
- `int64` overflow により大きな入金後の残高が負値や不正値に反転するリスク。
- handler / repository ごとに金額・残高 validation を重複実装して、将来の業務処理間で不整合が出るリスク。

現時点では DB / 業務 API / 取引履歴が未実装のため、今回差分から直接の二重送金、取引履歴欠落、片側振込成功、監査ログ欠落は発生しない。

### 推奨修正

今回差分内での修正必須事項はない。後続で入金・出金・振込 API に進むときは、今回追加された domain helper をサービス層または use case 層の入口で必ず使い、HTTP request の raw `int64` を直接 DB update に流さないことを実装規約として明確にする。

### 次サイクル planner への入力

次 cycle で code-changing scope を採択する場合は、DB や業務 API に進む前に、次のいずれかを小さく docs 化または設計補強するのが安全である。

1. domain helper を呼ぶ層と DB 制約の責務分担。
2. 振込依頼状態遷移と冪等性キー衝突時の扱い。
3. PostgreSQL 行ロックと 2 口座振込時のロック順序。
4. 口座別取引順序、`balance_after` 連続性、reconciliation 方針。

## Finding 2: `Amount{}` のゼロ値を作れるため、後続実装で constructor bypass のリスクが残る

### 根拠

- `Amount` は exported type であり、内部 `value` field は unexported だが、Go では外部 package からも `domain.Amount{}` のゼロ値を作れる。
- `NewAmount` は 0 以下を拒否するが、`Amount{}` を直接作った場合は constructor の validation を通らない。
- `AddBalance` / `SubtractBalance` は `amount.value <= 0` を再検証しているため、今回追加された helper 経由ではゼロ値 amount による残高変更は拒否される。
- ただし、将来の repository や transaction writer が `Amount` を受け取り、`Amount.Int64()` を直接 `transactions.amount` や `transfer_requests.amount` に保存する設計になると、ゼロ値 amount が DB 境界まで到達する余地がある。

### 影響

今回差分の範囲では金融事故には直結しないが、後続実装で `Amount` を「常に正の取引金額」と誤解して扱うと、0 円取引の取引履歴作成、0 円振込依頼、冪等性キーだけ消費する不正リクエスト、監査ログだけ残る無意味な取引などが発生し得る。DB 側に `amount > 0` 制約を置く予定があるため最終防衛線は作れるが、API / service / repository 境界での扱いを決めないと、テストが抜けた経路で domain constructor を bypass するリスクが残る。

### 推奨修正

次に domain 型を業務 API や repository に接続する前に、次のいずれかを設計・実装方針として明示する。

- `Amount` は必ず `NewAmount` で作るという規約を use case / repository 境界に書く。
- `Amount` を受け取る公開関数では、今回の `AddBalance` / `SubtractBalance` と同様に再 validation する。
- DB insert / update 前にも `amount > 0` を service または repository で確認し、DB constraint と二重に守る。
- 必要なら `Amount` に `Valid()` 相当の検証メソッドを追加し、`Int64()` を直接使う前の検証をテストしやすくする。

### 次サイクル planner への入力

「domain 型の境界利用ルール」を、DB / 業務 API 実装前の小さな設計補足として扱う。特に `transactions.amount`, `transfer_requests.amount`, API request amount の変換点で、raw `int64` から `Amount` へ変換し、失敗時は業務データを更新せず失敗監査ログへ接続する方針を後続 scope に含める。

## Finding 3: domain helper だけでは元帳・取引履歴の完全性はまだ保証されない

### 根拠

- planner / implementer は、HTTP route、handler、request / response schema、DB schema、repository、transaction manager、取引履歴、監査ログ、冪等性キー、PostgreSQL 行ロックを今回の非対象としている。
- `docs/design-principles.md` は、残高変更に成功した場合、口座残高更新と取引履歴作成を同じ DB transaction に含める方針を持っている。
- `docs/data-model.md` は、`accounts.balance_amount >= 0`, `transactions.amount > 0`, `transactions.balance_after >= 0` の制約案を持ち、`balance_after` は対象口座の更新後残高と一致させるとしている。
- `docs/test-strategy.md` は、取引履歴の増減と現在残高の整合、振込の 2 取引、途中失敗時 rollback、二重送信防止を今後のテスト対象としている。
- 今回の `internal/domain/money.go` は純粋な値計算 helper であり、残高更新と取引履歴作成の atomicity、`balance_after` の記録、口座別順序、監査ログ、冪等性、並行更新制御はまだ実装していない。

### 影響

今回の差分は有用な下位部品だが、これだけで「銀行元帳として正しい」とは言えない。将来、DB 実装時に今回の helper だけを使って `accounts.balance_amount` を更新し、`transactions` への追記や `balance_after` の一致確認を忘れると、現在残高は変わったが取引履歴で説明できない状態が起こり得る。また、行ロックなしで同時出金・同時振込を処理すると、各リクエスト内では `SubtractBalance` が成功しても、DB 上では lost update や残高不足見逃しが起こる可能性がある。

### 推奨修正

DB / 業務 API 実装へ進む前に、次の仕様を小さく docs 化することを推奨する。

- 残高変更 service は、`Amount` validation、残高計算、`accounts.balance_amount` 更新、`transactions` 追記、`balance_after` 記録、成功監査ログを同じ DB transaction に含める。
- 出金・振込元口座では、残高確認と更新前に PostgreSQL 行ロックを取得する。
- 2 口座振込では、内部 `account_id` 昇順などの固定順で両口座をロックする。
- `transactions.balance_after` は更新後 `accounts.balance_amount` と同一値にし、口座別の残高連続性をテストする。
- 業務拒否や途中失敗では、口座残高と取引履歴を変更せず、失敗監査ログを独立して残す。

### 次サイクル planner への入力

次 cycle は、今回の domain helper を前提にしてもすぐ業務 API へ進まず、次のどちらかを優先候補にすることを推奨する。

1. `docs/data-model.md` / `docs/test-strategy.md` に、口座別取引順序、`balance_after` 連続性、DB constraint、reconciliation の検証方針を追加する docs-only scope。
2. `docs/design-principles.md` / `docs/use-cases.md` に、PostgreSQL 行ロック、振込時ロック順序、デッドロック時の扱い、冪等性キー状態遷移との接続を追加する docs-only scope。

## 事故シナリオメモ

今回差分で直接発生する事故ではないが、後続実装時に注意すべきシナリオを残す。

1. **constructor bypass**: `Amount{}` を作り、`Int64()` の 0 を直接 DB に保存して 0 円取引が残る。
2. **履歴欠落**: `AddBalance` / `SubtractBalance` の結果だけで `accounts.balance_amount` を更新し、`transactions` 追記と `balance_after` 記録を忘れる。
3. **並行出金**: 同一口座に対する 2 つの出金が行ロックなしで同時に現在残高を読み、それぞれ `SubtractBalance` に成功して、DB commit 順で lost update または残高不足見逃しが起こる。
4. **振込片側成功**: 振込元減算と振込先加算、2 件の取引履歴、振込依頼状態、成功監査ログを同一 DB transaction に入れず、一部だけが commit される。

## 結論

今回の実装差分は accepted scope に適合しており、元帳・残高観点で修正必須のブロッカーは確認しなかった。`internal/domain` の金額・残高 helper は、業務 API / DB 実装前に導入する下位部品として妥当である。次 cycle では、この helper を実際の元帳更新へ接続する前に、冪等性、行ロック、取引履歴の `balance_after` 連続性、DB transaction 境界を docs / test 方針へ落とすことを推奨する。
