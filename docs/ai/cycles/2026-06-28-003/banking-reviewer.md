# banking-reviewer: 2026-06-28-003

## レビュー対象

- 対象 cycle: `2026-06-28-003`
- 参照した成果物:
  - `AGENTS.md`
  - `.codex/agents/README.md`
  - `docs/ai/cycles/README.md`
  - `docs/ai/cycles/2026-06-28-003/planner.md`
  - `docs/ai/cycles/2026-06-28-003/implementer.md`
- 参照した未コミット実装差分:
  - `README.md`
  - `cmd/server/main.go`
  - `cmd/server/main_test.go`
  - `docs/ai/cycles/2026-06-28-003/implementer.md`
- レビュー観点: 元帳、残高、取引履歴、ドメイン、金融事故リスク。

## Finding

### Finding 1: 今回の実装差分は金融ドメイン状態を変更しておらず、元帳・残高・取引履歴上の直接的な事故リスクは増やしていない

#### 根拠

- planner の accepted scope は HTTP server hardening と最小設定管理に限定され、残高、元帳、取引履歴、振込、冪等性、監査ログ、DB transaction の仕様を暗黙に確定しないことを banking-reviewer の重点確認観点としている。
- 実装差分は `cmd/server/main.go` の listen address 設定、環境変数 `BANK_SYSTEM_HTTP_ADDR`、`http.Server` timeout、server 構築関数の追加に集中している。
- `cmd/server/main_test.go` は server 設定と timeout の unit test であり、口座、残高、取引、振込、監査ログ、DB transaction を扱っていない。
- `README.md` は現状の実装範囲として `GET /healthz` のみ、DB 接続・認証・業務 API は未導入と明記している。
- `docs/ai/cycles/2026-06-28-003/implementer.md` も、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ、PostgreSQL、冪等性キー処理を実装していないと記録している。

#### 影響

- 今回差分だけを見る限り、二重送金、残高マイナス、片側だけ成功する振込、取引履歴欠落、元帳と残高の不一致、取消・組戻しの誤処理といった金融事故シナリオは発火しない。
- `/healthz` の固定レスポンスは金融ドメイン情報を返さないため、残高や取引履歴の漏えい・誤表示につながる変更ではない。
- ただし、HTTP server の入口が整ったことで、次 cycle 以降に業務 API を追加しやすくなる。業務 API 追加時に元帳・残高・監査境界を未確定のまま進めると、今回ではなく次以降の差分で金融事故リスクが顕在化する。

#### 推奨修正

- 今回差分に対する banking-reviewer 観点の必須修正はない。
- 現状のまま、`/healthz` に残高、口座数、DB 接続先、内部設定、環境変数、取引件数などの業務・運用情報を追加しない方針を維持する。
- 次に業務 API または DB schema へ進む前に、元帳・残高方向・transaction 境界・冪等性・監査ログ境界を docs scope として先に扱う。

#### 次サイクル planner への入力

- 次 cycle の高優先候補として、`transaction_type` ごとの残高増減方向、`balance_after` の意味、残高非負制約、残高変更と取引履歴作成の同一 DB transaction 境界を設計文書に具体化することを推奨する。
- 業務 API 実装に進む場合は、先に「入金」「出金」「振込」のうち 1 つだけを選び、元帳記録・残高更新・監査ログ・冪等性の最小単位を明示した accepted scope にする。
- health endpoint を将来 readiness endpoint に拡張する場合でも、残高件数、取引件数、顧客件数、口座状態の分布など、金融ドメイン情報を公開レスポンスに含めない制約を planner に残す。

### Finding 2: README と implementer 成果物は未実装範囲を明示しており、金融仕様の暗黙確定は避けられている

#### 根拠

- `README.md` は未実装機能として、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、PostgreSQL、transaction 処理、認証、監査ログ、冪等性キー処理を列挙している。
- `README.md` は金融ドメイン仕様、DB schema、認証方式、監査ログ方式を今後の cycle で扱うと明記している。
- `docs/ai/cycles/2026-06-28-003/implementer.md` は `transaction_type`、`reversal`、残高競合制御、監査ログ境界を確定していないと記録している。

#### 影響

- 現時点の利用者・次 agent が、今回の HTTP server hardening を業務 API や元帳仕様の実装済み状態と誤解するリスクは低い。
- 金融ドメインで後戻りしにくい判断、特に残高モデル、取引履歴モデル、取消・組戻し、冪等性キー一意スコープが、今回差分でなし崩しに決まっていない。

#### 推奨修正

- 今回差分に対する必須修正はない。
- 次 cycle で README を更新する場合も、実装済みの範囲と設計済みだが未実装の範囲を分けて記載する。
- 金融仕様を docs に追加する際は、実装済み事実ではなく「設計案」「人間確認事項」「accepted scope」を明確に分離する。

#### 次サイクル planner への入力

- 次 cycle では、README の未実装一覧を維持したまま、金融ドメイン設計 docs の具体化を優先候補にする。
- 特に `docs/data-model.md` と `docs/design-principles.md` に対して、残高更新と取引履歴作成が必ず同一 transaction で行われること、履歴は原則追記型で改変しないこと、失敗時の監査記録方針は人間確認事項として残すことを検討する。

## 事故シナリオ確認

| 事故シナリオ | 今回差分での状態 | コメント |
| --- | --- | --- |
| 二重送金 | 発火しない | 振込 API、冪等性キー、DB 書き込みが未実装。 |
| 残高マイナス | 発火しない | 残高フィールド、出金処理、振込処理が未実装。 |
| 元帳・残高不一致 | 発火しない | 取引履歴・残高更新処理が未実装。 |
| 片側だけ成功する振込 | 発火しない | 口座間資金移動処理が未実装。 |
| 取引履歴欠落 | 発火しない | 取引履歴作成処理が未実装。 |
| 公開 health endpoint からの金融情報漏えい | 現状は低リスク | `/healthz` は固定 JSON のみ。今後 readiness 化する場合は注意が必要。 |

## 人間確認事項

1. 次に金融ドメインへ進む前に、`transaction_type` と残高増減方向をどの粒度で定義するか。
2. `reversal`、取消、組戻しを MVP 前に設計対象へ含めるか、初期 MVP では「履歴追記のみで取消未対応」と明示するか。
3. 並行出金・並行振込時の残高保護を、PostgreSQL の行ロック、条件付き UPDATE、または別方式のどれで学習・実装するか。
4. 冪等性キーの一意スコープを、操作種別、送信元口座、ログインユーザー、リクエスト本文 hash のどこまで含めるか。
5. 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するか。

## テスト・確認

- `git status --short`: 未コミット実装差分を確認した。
- `git diff -- README.md cmd/server/main.go cmd/server/main_test.go docs/ai/cycles/2026-06-28-003/implementer.md`: 実装差分を確認した。
- `go test ./...`: 成功。
