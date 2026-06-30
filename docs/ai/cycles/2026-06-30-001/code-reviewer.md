# code-reviewer: 2026-06-30-001

## 確認した入力

- 作業開始時に `git status --short` を確認し、未コミット変更はなかった。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md` を確認し、reviewer は source / README を変更せず、本ファイルだけに書き込む方針を確認した。
- README と `docs/START_HERE.md`、`docs/domain-model.md`、`docs/data-model.md`、`docs/design-principles.md`、`docs/security-notes.md`、`docs/test-strategy.md`、`docs/use-cases.md`、`docs/ai/output/human/001-human-review.md`、同 cycle の `planner.md` / `implementer.md` を確認した。
- 実装差分として commit `d104ada Add domain boundary validation helpers` を優先確認した。対象は `README.md`、`internal/domain/money.go`、`internal/domain/money_test.go`、`docs/ai/cycles/2026-06-30-001/implementer.md`。

## Finding 1: blocking なし。accepted scope の境界再検証 helper は小さく整合している

### 根拠

- `Amount.Validate()` は `value <= 0` を `ErrAmountMustBePositive` として拒否し、`NewAmount` は同じ method を経由している。
- `Balance.Validate()` は `value < 0` を `ErrBalanceMustBeNonNegative` として拒否し、`NewBalance` は同じ method を経由している。
- `AddBalance` / `SubtractBalance` は不正 `Amount` の判定を `Amount.Validate()` へ寄せ、エラー時に元 `Balance` を返す既存挙動を維持している。
- 追加 test は planner の重点確認観点である `Amount{}` 拒否、正の `Amount` 許可、`Balance{}` / 正の `Balance` 許可、負の `Balance` 拒否を直接検証している。
- README は DB、認証、業務 API が未実装である点を維持しつつ、境界再検証用 `Validate()` method の存在だけを追記している。

### 影響

- Go domain package の公開 API と test は accepted scope に収まり、PostgreSQL / transaction / HTTP / 認証 / 監査 / 冪等性の未確定領域を先取りしていない。
- constructor bypass で作られた不正 `Amount` / `Balance` を service、repository、DB insert 境界で明示的に再検証するための足場ができた。
- 既存 sentinel error を再利用しているため、将来の handler / usecase 層で domain error mapping を作る際にも互換性を保ちやすい。

### 推奨修正

- 今 cycle 内での修正は不要。
- 次に service / repository / DB insert 境界を作るときは、`Int64()` で永続化する直前または transaction usecase の入口で `Validate()` を呼ぶ規約を docs または test に固定する。

### 次サイクル planner への入力

- DB schema / repository 着手前に、domain `Validate()` と PostgreSQL `CHECK` 制約の対応関係を設計 scope にする。
- 例: `amount > 0`、`balance >= 0`、`transactions.balance_after >= 0`、必要なら `currency = 'JPY'` のような DB 制約案を docs 化する。

## Finding 2: 残高演算 helper は入力 `Balance` 自体を再検証しないため、将来の境界責務を明確にする必要がある

### 根拠

- 今回追加された `Balance.Validate()` は負の残高を検出できる。
- 一方、`AddBalance(balance, amount)` / `SubtractBalance(balance, amount)` は `amount.Validate()` のみを呼び、引数 `balance` に対しては `balance.Validate()` を呼ばない。
- 現在の package 外からは `Balance.value` が非公開であり、通常は `NewBalance` 経由で不変条件を満たす値が渡る。ただし同一 package の test / 将来の mapper / repository 実装では、constructor を経由しない値を扱う可能性がある。
- planner / implementer は `Validate()` を service / repository / persistence 境界で使う補助として定義しており、今回の accepted scope では `AddBalance` / `SubtractBalance` に `Balance` 再検証を必須としていない。

### 影響

- 現時点では blocking ではない。実装差分は planner scope と矛盾せず、既存挙動も壊していない。
- ただし将来、DB から読み取った残高や test fixture で作った `Balance` を演算 helper に渡す設計にすると、演算 helper が不正な開始残高を検出する責務を持つのか、呼び出し元境界が検出する責務を持つのかが曖昧になり得る。
- 特に repository / transaction manager 導入時にこの責務が未定義だと、負の残高を「残高不足」や別の演算結果として扱ってしまう test 漏れにつながる可能性がある。

### 推奨修正

- 今 cycle での code 修正は不要。次 cycle 以降で、残高演算 helper の責務を次のどちらかに明文化する。
  1. `AddBalance` / `SubtractBalance` の先頭で `balance.Validate()` も呼び、不正開始残高を `ErrBalanceMustBeNonNegative` として返す。
  2. 演算 helper は「valid な `Balance` を受け取る」前提に限定し、service / repository / DB read 境界で `Balance.Validate()` を必ず呼ぶ test / docs を追加する。
- 金融事故リスクを下げる観点では、DB read mapper または usecase transaction 入口で `Balance.Validate()` を呼ぶ test を優先して追加するのがよい。

### 次サイクル planner への入力

- repository / transaction 実装前の小 scope として、「DB から復元した `Balance` / `Amount` を domain 境界で `Validate()` する mapper test」または「演算 helper に開始 `Balance` validation を追加する scope」を検討する。
- PostgreSQL `CHECK (balance >= 0)` と Go 側 `Balance.Validate()` の二重防衛を同じ設計判断として記録する。

## Finding 3: テスト範囲は現在の domain scope には十分だが、DB / transaction 設計の検証は未着手

### 根拠

- `go test ./...` は成功した。
- `rg -n "float32|float64" internal/domain` は一致なしだった。`rg` は一致なしの場合 exit code 1 になるため、これは想定どおりの結果として扱う。
- 今回の差分は domain helper と README / cycle artifact のみであり、PostgreSQL 接続、SQL、migration、transaction manager、repository は追加されていない。

### 影響

- 現時点の code-changing scope に対する unit test は妥当。
- 一方、金融システムとして重要な「残高更新 + 取引履歴 + 成功監査ログを同一 DB transaction で commit / rollback する」検証は、まだ code が存在しないため未検証のまま残る。

### 推奨修正

- 今 cycle での修正は不要。
- DB 導入 cycle では、domain helper 単体 test に加えて、repository / transaction usecase の commit / rollback test、残高不足時に残高と取引履歴が変わらない test、成功監査ログと業務更新の transaction 境界 test を追加する。

### 次サイクル planner への入力

- 次の設計 scope 候補として、PostgreSQL schema / transaction manager に入る前に test strategy を具体化する。
- 最低限、`accounts.balance` と `transactions.balance_after` の `CHECK` 制約、行ロック順序、deadlock 方針、transaction rollback test 方針を docs に固定する。

## 実行した確認

- ✅ `git status --short`
- ✅ `go test ./...`
- ✅ `rg -n "float32|float64" internal/domain`（一致なし。浮動小数点利用なし）
- ✅ `git diff d104ada^ d104ada -- README.md internal/domain/money.go internal/domain/money_test.go`
