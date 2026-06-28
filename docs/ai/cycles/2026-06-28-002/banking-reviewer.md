# banking-reviewer: 2026-06-28-002

## レビュー対象

- 役割: `banking-reviewer` として、銀行ドメイン、元帳、残高、取引履歴、金融事故リスクの観点でレビューした。
- 優先対象: 同一 cycle の実装差分。直近 commit `261cc5e Add minimal Go REST API skeleton` の差分を中心に確認した。
- 確認した入力:
  - `AGENTS.md`
  - `docs/ai/cycles/README.md`
  - `docs/ai/cycles/2026-06-28-002/planner.md`
  - `docs/ai/cycles/2026-06-28-002/implementer.md`
  - 実装ファイル: `README.md`, `go.mod`, `cmd/server/main.go`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
  - 関連 docs: `docs/START_HERE.md`, `docs/mvp.md`, `docs/domain-model.md`, `docs/data-model.md`, `docs/design-principles.md`, `docs/use-cases.md`, `docs/test-strategy.md`, `docs/security-notes.md`
  - `git status --short`, `git log --oneline -5`, `git diff`, `git diff --cached`, `git show --stat --oneline --find-renames HEAD`, `git diff --stat HEAD~1..HEAD`
- human notes: `docs/ai/output/human/` は存在しないため、追加の人間メモは確認できなかった。

## 実装差分の要約

- `go.mod` が追加され、module 名は `bank-system`、Go version は `1.24` とされた。
- `cmd/server/main.go` に、標準ライブラリ `net/http` の HTTP server 起動処理が追加された。
- `internal/httpapi/router.go` に、`GET /healthz` のみを提供する router / handler が追加された。
- `internal/httpapi/router_test.go` に、`/healthz` の成功、未対応 method 拒否、固定レスポンスの情報露出抑止を確認するテストが追加された。
- `README.md` に、現在の実装範囲、起動方法、テスト方法、未実装機能、学習用であり本番金融システムではない旨が記録された。
- 残高、口座、取引履歴、振込、冪等性、監査ログ、DB transaction、PostgreSQL schema は実装されていない。

## 総合評価

今回の実装は、同一 cycle の accepted scope どおり、金融業務 API に入る前の最小 Go REST API 土台と `/healthz` に限定されている。銀行ドメイン/元帳観点では、残高や取引履歴を操作するコードが追加されていないため、二重送金、残高不整合、取引履歴欠落、監査ログ欠落などの直接的な金融事故リスクはこの差分では新規に発生していない。

また、`README.md` と `implementer.md` が、顧客、口座、入出金、振込、DB、認証認可、監査ログ、冪等性キー処理を未実装として明示しているため、今回の skeleton が金融仕様を暗黙に確定してしまうリスクも低い。

## Findings

### Finding 1: 今回差分に銀行ドメイン/元帳上のブロッカーはない

- 重要度: Info
- 種別: 実装差分レビュー

#### 根拠

- `GET /healthz` は固定 JSON `{"status":"ok"}` を返すだけで、口座、残高、取引、振込、監査ログ、冪等性キーを参照・更新しない。
- router は `/healthz` のみを登録しており、業務 API は追加されていない。
- `cmd/server/main.go` は HTTP server 起動だけを行い、DB 接続、残高更新、取引作成、監査ログ作成を行わない。
- `README.md` は、現時点で最小 REST API server と health check のみ実装済みであり、DB 接続、認証、業務 API が未導入であることを明示している。
- `implementer.md` は、DB、認証、顧客、口座、入出金、振込、残高、取引履歴、監査ログ、冪等性キー処理を scope 外・未実装として記録している。

#### 影響

- 今回差分だけでは、残高非負制約違反、取引履歴との不整合、振込の片側成功、二重送金、監査証跡欠落などの金融事故は発生しない。
- 一方で、MVP に必要な元帳・残高・取引履歴・監査ログの安全性はまだ実装で担保されていない。これは今回 scope 外として明示されているため、欠陥ではなく次 cycle 以降の設計・実装課題である。

#### 推奨修正

- 今回差分に対する修正は不要。
- 次に入出金・振込・DB schema に進む前に、元帳と残高更新の仕様を docs で明確化することを推奨する。

#### 次サイクル planner への入力

- 業務 API を追加する前に、少なくとも以下を accepted scope 候補として検討する。
  1. `transaction_type` ごとの残高増減方向の確定。
  2. `accounts.balance_amount` と `transactions.balance_after` の整合性ルール。
  3. 残高更新と取引履歴作成を同一 DB transaction に入れる境界。
  4. 振込時の出金取引・入金取引・振込依頼更新を同一 transaction で扱う方針。
  5. 出金・振込の競合更新対策として、PostgreSQL 行ロックまたは条件付き UPDATE のどちらを採るか。
  6. 冪等性キーの一意スコープと、同一キー異内容リクエストの扱い。
  7. 監査ログを業務 transaction と同一境界に含めるか、失敗時にどう扱うか。

## 事故シナリオ観点

今回差分では業務データを持たないため、以下の事故シナリオは発火しない。

| 事故シナリオ | 今回差分での状態 | コメント |
| --- | --- | --- |
| 二重送金 | 該当なし | 振込 API と冪等性処理が未実装。 |
| 残高マイナス | 該当なし | 残高カラム、出金、振込が未実装。 |
| 取引履歴欠落 | 該当なし | 取引作成処理が未実装。 |
| 片側だけ成功する振込 | 該当なし | DB transaction と振込処理が未実装。 |
| 監査ログ欠落 | 今回 scope では業務操作なし | health check は重要業務操作ではないため、監査ログ未実装は今回の金融事故リスクではない。 |
| 金融仕様の暗黙確定 | 低い | README と implementer 成果物で未実装範囲が明記されている。 |

## 人間確認事項

次 cycle 以降で残高・取引履歴・振込・監査ログに入る前に、人間が確認すべき事項は以下。

1. MVP の `transaction_type` を `deposit`, `withdrawal`, `transfer_debit`, `transfer_credit`, `reversal` とするか。
2. `reversal` を MVP で実装しない場合でも、取消・訂正の将来方針だけ先に docs へ定義するか。
3. 残高競合制御を PostgreSQL の行ロックで行うか、条件付き UPDATE で行うか。
4. 冪等性キーの一意スコープを `requested_by_user_id`、`source_account_id`、または複合条件のどれにするか。
5. 同一冪等性キーで異なる amount / destination が送られた場合、拒否、既存結果返却、監査アラートのどれにするか。
6. 監査ログ書き込み失敗時、業務 transaction を失敗させるか、業務処理を優先して別途復旧対象にするか。

## 結論

- 今回の実装差分は、銀行ドメイン仕様を暗黙に決めず、残高・元帳・取引履歴に触れない最小 skeleton として妥当。
- banking-reviewer として、この差分に対する blocker / major finding はない。
- 次 cycle で業務 API や DB schema に進む場合は、先に元帳・残高方向・transaction 境界・冪等性・監査ログ境界を設計 scope として切り出すことを強く推奨する。
