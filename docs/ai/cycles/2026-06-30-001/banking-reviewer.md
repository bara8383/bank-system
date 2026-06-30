# banking-reviewer: 2026-06-30-001

## 確認した入力

- 作業開始時に `git status --short` を確認し、未コミット変更がないことを確認した。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md` を確認し、reviewer は実装差分優先・書き込み先を本ファイルのみに限定する方針に従った。
- `README.md`、`docs/START_HERE.md`、`docs/*.md`、`docs/ai/output/human/001-human-review.md`、`docs/ai/cycles/2026-06-30-001/planner.md`、`docs/ai/cycles/2026-06-30-001/implementer.md` を確認した。
- 実装差分は `d104ada Add domain boundary validation helpers` を対象に確認した。

## Finding 1: blocking なし。`Amount.Validate()` / `Balance.Validate()` は今回 scope の境界再検証目的を満たしている

### 根拠

- `Amount.Validate()` は `value <= 0` を `ErrAmountMustBePositive` として拒否するため、`Amount{}` や 0 円取引が service / repository / DB insert 境界で混入するリスクを低減できている。
- `Balance.Validate()` は `value < 0` を `ErrBalanceMustBeNonNegative` として拒否し、`Balance{}` は 0 円残高として valid に扱っている。
- `NewAmount` / `NewBalance` は作成した値に対して `Validate()` を呼ぶ構造になっており、constructor と境界再検証 helper の rule が分岐していない。
- 追加 test は `Amount{}` 拒否、正の `Amount` 許可、0/正の `Balance` 許可、負の `Balance` 拒否を確認している。
- README は業務 API / DB / 認証 / 監査 / 冪等性が未実装である前提を維持しつつ、domain helper の境界再検証用途を説明している。

### 影響

- 後続の入金・出金・振込・repository 実装で、constructor を経由しない値を保存前または計算前に明示的に検査できる。
- 0 円取引の混入、負の残高の永続化、DB 制約前のアプリケーション境界漏れを減らす小さな防御層として妥当。
- 今回差分は元帳、取引履歴、冪等性、状態遷移を新規実装していないため、既存の未実装領域に新たな金融事故経路を追加していない。

### 推奨修正

- 今回 cycle 内での修正は不要。
- 後続の service / repository / DB insert 実装では、`Amount.Validate()` と `Balance.Validate()` を「保存・計算境界で呼ぶ」規約を test と docs に接続する。

### 次サイクル planner への入力

- DB schema / repository に進む前に、`Amount.Validate()` / `Balance.Validate()` と PostgreSQL `CHECK (amount > 0)`、`CHECK (balance_amount >= 0)`、`CHECK (balance_after >= 0)` の責務分担を設計 scope 化する。
- 入金・出金・振込 service の最初の実装では、domain helper の validation、DB transaction、取引履歴、成功監査ログ、失敗監査ログを一体で test する scope を優先する。

## Finding 2: `AddBalance` / `SubtractBalance` は `amount` のみ再検証し、入力 `balance` の不変条件は呼び出し側境界に依存している

### 根拠

- 実装差分では `AddBalance` / `SubtractBalance` が `amount.Validate()` を呼ぶようになった一方、引数 `balance` に対して `balance.Validate()` は呼んでいない。
- `Balance{value: -1}` のような値は domain package 内や将来の repository 復元処理で作成でき、`Balance.Validate()` はそれを検出できるが、計算 helper 自体は不正な元残高を拒否する最終防衛線にはなっていない。
- planner / implementer の scope は「境界再検証 helper 追加」であり、既存 helper の戻り値互換性維持も条件だったため、今回差分としては許容範囲。ただし銀行ドメイン上は、負残高を起点にした加算・減算が呼び出し側 validation 漏れで進む余地が残る。

### 影響

- 将来 DB から復元した残高、テスト helper、migration、repository mock などが constructor を経由せずに負の `Balance` を生成した場合、service 側が `Balance.Validate()` を呼び忘れると、負残高を起点にした残高更新や `ErrInsufficientBalance` で負残高を返し続ける処理になり得る。
- 元帳・取引履歴実装時に `balance_after` を計算 helper の戻り値から記録する設計にすると、不正な開始残高を早期検出できず、残高と取引履歴の不整合調査が難しくなる。
- 現時点では業務 API / DB / repository が未実装のため即時事故ではないが、DB 接続前に呼び出し規約または helper 側防御のどちらを採るか決める必要がある。

### 推奨修正

- 次 cycle 以降で、`AddBalance` / `SubtractBalance` が `balance.Validate()` も実行して invalid balance を拒否するべきか、または service / repository 境界で必ず `Balance.Validate()` 済みの値だけを渡す規約にするかを設計判断として明文化する。
- helper 側で `balance.Validate()` を追加する場合は、エラー時に元 balance を返す既存挙動と sentinel error を維持し、負の開始残高を使った加算・減算 test を追加する。
- 規約で対応する場合でも、repository から復元した `Balance` と `transactions.balance_after` を service 層で検証する test を必須にする。

### 次サイクル planner への入力

- 小さな code-changing scope 候補: `AddBalance` / `SubtractBalance` に `balance.Validate()` を追加し、負の開始残高を拒否する unit test を追加する。
- docs scope 候補: `docs/design-principles.md` または `docs/test-strategy.md` に、DB 復元値・service 入力値・DB insert 値の validation 境界を明記する。
- DB 実装候補に進む場合は、`accounts.balance_amount >= 0`、`transactions.amount > 0`、`transactions.balance_after >= 0` の DB 制約とアプリケーション validation の二重化を accepted scope に含める。

## Finding 3: 元帳・取引履歴・冪等性・状態遷移の実装はまだ増えておらず、既存の高優先未実装リスクは継続

### 根拠

- README と implementer artifact は、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、PostgreSQL、監査ログ、冪等性キー処理が未実装であることを明示している。
- docs では、残高変更と取引履歴・成功監査ログを同じ PostgreSQL transaction に含める方針、失敗監査ログを独立して残す方針、振込の冪等性・行ロック・`balance_after` の整合性が重要事項として整理されている。
- 今回差分は domain validation helper に限定され、元帳テーブル、取引履歴、振込依頼状態、冪等性キー、行ロックは追加していない。

### 影響

- 今回差分単体では金融事故リスクを増やしていないが、MVP の本質である「残高更新と取引履歴の一致」「二重送金防止」「失敗時監査」は未実装のまま残る。
- 次に業務 API だけを先行追加すると、DB transaction、行ロック、冪等性、監査ログが不足した endpoint になり、二重実行・片側更新・履歴欠落のリスクが高い。

### 推奨修正

- 次の実装は、業務 API の前に DB 制約 / transaction / repository 境界、またはそれらの詳細設計を先に扱う。
- 振込より先に、入金または出金の単一口座残高変更で「残高更新・取引履歴・成功監査ログの同一 transaction」「失敗監査ログの独立保存」「`balance_after` 一致」を検証する最小 slice を作る。
- 冪等性は振込実装前に、操作種別、送信元口座、ログインユーザー、request body hash を含むキー設計と同一キー再送時の MVP 挙動を docs / tests に固定する。

### 次サイクル planner への入力

- 優先候補 1: DB schema 前設計として、`accounts.balance_amount`、`transactions.amount`、`transactions.balance_after`、`transfer_requests` 状態、監査ログ failure_reason、行ロック順序を docs 化する。
- 優先候補 2: domain helper の次の小変更として、負の開始 `Balance` を計算 helper で拒否する defensive validation を追加する。
- 優先候補 3: 入金または出金の repository/service skeleton を作る場合は、業務 API ではなく transaction 境界・履歴・監査・rollback test を最小単位にする。
