# implementer: 2026-06-29-001

## 参照した accepted scope

`docs/ai/cycles/2026-06-29-001/planner.md` の accepted scope「A. 元帳・残高方向・成功時 transaction 境界を docs に具体化する」を参照した。

対象ファイルは次に限定されていた。

- `docs/design-principles.md`
- `docs/data-model.md`
- `docs/test-strategy.md`
- `docs/ai/cycles/2026-06-29-001/implementer.md`

accepted scope の非対象として、Go ソースコード、HTTP handler、server 設定、DB 接続コード、PostgreSQL migration、DB schema、SQL、repository、transaction manager、業務 API、認証認可、`README.md`、cycle 001 から 005 の成果物は変更しないことを確認した。

## 変更内容

- `docs/design-principles.md`
  - 残高変更成功時は、口座残高更新と取引履歴作成を同一データベーストランザクションに含める方針を追記した。
  - 入金、出金、振込成功時の最小トランザクション境界を表で追記した。
  - 金額は正の整数、MVP の通貨は JPY、残高は 0 未満にしないことを追記した。
  - 失敗時監査ログ、監査ログ書き込み失敗時の業務処理、並行更新制御方式は未確定として分離した。
- `docs/data-model.md`
  - `transactions.transaction_type` ごとの残高増減方向表を追記した。
  - `balance_after` は、その取引を対象口座へ適用した直後の口座残高であり、0 以上かつ更新後残高と一致することを追記した。
  - 1 件の振込では `transfer_debit` と `transfer_credit` を別行で持ち、それぞれ対象口座ごとの `balance_after` を持つことを追記した。
  - 取引履歴は追記型を基本とし、`reversal` の詳細は未確定として残した。
- `docs/test-strategy.md`
  - 将来の入金、出金、振込実装で確認する、残高変更と取引履歴のトランザクション整合性テスト観点を追記した。

## scope 適合性

- accepted scope の対象 3 文書と本成果物のみを変更した。
- 残高方向、`balance_after`、成功時の同一データベーストランザクション境界、将来テスト観点に限定した。
- `reversal`、並行更新方式、冪等性キー詳細、監査ログ書き込み失敗時の扱い、認証認可は確定していない。
- 金額を浮動小数点で扱う記述は追加していない。

## 実装しなかったこと

- Go ソースコード、HTTP handler、server 設定、DB 接続コードは変更していない。
- PostgreSQL migration、DB schema、SQL、repository、transaction manager は作成していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログの業務 API は実装していない。
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウトは実装・確定していない。
- `README.md` と cycle 001 から 005 の成果物は変更していない。

## テスト結果

- `go test ./...`: 未実行。環境に `go` コマンドがなく、`/bin/bash: line 1: go: command not found` で失敗した。
- `git diff -- docs/design-principles.md docs/data-model.md docs/test-strategy.md`: 差分を確認し、accepted scope 内の docs-only 変更であることを確認した。
- `rg -n "浮動小数点|float|double|reversal|balance_after|transaction_type|データベーストランザクション" docs/design-principles.md docs/data-model.md docs/test-strategy.md`: 追記箇所と未確定事項の表現を確認した。

## 未確認事項

- MVP で `reversal` を実装対象に含めるか、初期は未実装として履歴不可変方針だけを維持するか。
- 並行出金・並行振込時の残高保護方式を、PostgreSQL の行ロック、条件付き UPDATE、または別方式のどれにするか。
- 冪等性キーの一意スコープ、同一キー異内容時の扱い、保存期間。
- 監査ログ書き込み失敗時に業務処理を失敗させるか、別経路で補償するか。
- 失敗時監査ログを業務トランザクションの rollback と独立して残す必要があるか。
- 業務 API 追加前の認証方式、RBAC、管理者・運用担当者の権限境界。
