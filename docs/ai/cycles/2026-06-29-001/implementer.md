# implementer: 2026-06-29-001

## 参照した accepted scope

`docs/ai/cycles/2026-06-29-001/planner.md` の accepted scope を参照し、監査ログの記録要否、保存境界、書き込み失敗時、閲覧権限、マスキング、および前 cycle reviewer 指摘の小補修に限定して実装した。

対象ファイルは次に限定されていた。

- `docs/design-principles.md`
- `docs/security-notes.md`
- `docs/data-model.md`
- `docs/test-strategy.md`
- `docs/ai/cycles/2026-06-29-001/implementer.md`

非対象として、Go ソースコード、HTTP handler、server 設定、DB 接続コード、PostgreSQL migration、DB schema、SQL、repository、transaction manager、業務 API、認証/認可実装、Cookie session、CSRF token、冪等性キー詳細、PostgreSQL 行ロック、監査ログ改ざん検知、outbox、非同期補償、運用アラート、README 更新は実施しないことを確認した。

## 変更内容

- `docs/design-principles.md`
  - 残高変更を伴う成功操作では、業務データ更新、取引履歴、振込依頼状態更新、成功監査ログを同じ PostgreSQL データベーストランザクションに含める MVP 初期案を追記した。
  - 業務拒否、DB transaction 途中失敗、監査ログ書き込み失敗を分け、失敗監査ログは業務データ更新の rollback と独立して残す必要があると明記した。
  - 成功監査ログが書けない残高変更や権限変更は MVP では fail closed とし、監査ログ書き込み失敗の運用通知・再送・補償は将来検討として残した。
- `docs/security-notes.md`
  - 監査ログに含める属性として `actor_user_id`、`action_type`、`target_type`、`target_id`、`result`、`failure_reason`、`occurred_at`、`ip_address`、`user_agent` を明記した。
  - `failure_reason` は安全な分類または短い理由にし、request body は raw ではなく hash で扱う設計案にした。
  - password、token、secret、CSRF token、セッションID、raw request body、過剰な個人情報を監査ログに保存しない方針を追記した。
  - MVP では監査ログ照会を `admin` のみに限定し、`operator` ロールは MVP 対象外とした。
- `docs/data-model.md`
  - `audit_logs.failure_reason` の説明を安全な分類または短い理由へ補強した。
  - request body hash 用の `request_body_hash` 候補を追加し、raw request body を保存しない方針を明記した。
  - 失敗時は `target_id` が未確定または未設定になり得ることを補足した。
  - 主な制約案に `transactions.balance_after >= 0` を追加した。
- `docs/test-strategy.md`
  - 成功時に業務データ、取引履歴、成功監査ログが整合して作られるテスト観点を補強した。
  - 残高不足、権限不足、不正金額、存在しない対象などの業務拒否で業務データを変えず失敗監査ログを残す観点を追加した。
  - DB transaction 途中失敗を注入して rollback し、rollback 後に失敗監査ログだけを独立して残すテスト観点を追加した。
  - 監査ログ書き込み失敗時の fail closed、監査ログのマスキング、`admin` 限定/`operator` MVP 対象外、`transactions.balance_after >= 0` の確認観点を追加した。

## scope 適合性

- accepted scope の対象 4 文書と本成果物のみを変更した。
- 監査ログ境界、マスキング、閲覧権限、`transactions.balance_after >= 0`、rollback テスト観点に限定した。
- Go ソースコード、DB schema、migration、repository、transaction manager、業務 API、認証/認可実装には踏み込んでいない。
- `reversal`、冪等性キー詳細、PostgreSQL 行ロック、ロック順序、監査ログ改ざん検知、outbox、非同期補償、運用アラートは今回確定していない。
- `README.md` は、実行方法や現行実装範囲に変更がないため更新していない。

## 実装しなかったこと

- Go ソースコード、HTTP handler、server 設定、DB 接続コードは変更していない。
- PostgreSQL migration、DB schema、SQL、repository、transaction manager は作成していない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装していない。
- 認証、認可、ユーザー登録、パスワードハッシュ、Cookie session、CSRF token、ログアウトは実装していない。
- `reversal`、取消、組戻し、訂正の詳細仕様は作成していない。
- 冪等性キーの複合一意制約、同一キー同一内容/異内容、処理中再送、保存期間は確定していない。
- PostgreSQL 行ロック、ロック順序、デッドロック回避、分離レベルは確定していない。
- 監査ログの改ざん検知、outbox、非同期補償、運用アラート、外部 SIEM 連携は実装・確定していない。
- `README.md` と過去 cycle `2026-06-28-001` から `2026-06-28-005` の成果物は変更していない。

## テスト結果

- `go test ./...`: 成功。
  - `ok  	bank-system/cmd/server	(cached)`
  - `ok  	bank-system/internal/httpapi	(cached)`
- `git diff -- docs/design-principles.md docs/security-notes.md docs/data-model.md docs/test-strategy.md docs/ai/cycles/2026-06-29-001/implementer.md`: 差分を確認し、accepted scope 内の docs-only 変更であることを確認した。
- `rg -n "監査ログ|audit_logs|failure_reason|balance_after|rollback|ロールバック|password|token|secret|CSRF|operator" docs/design-principles.md docs/security-notes.md docs/data-model.md docs/test-strategy.md`: 監査ログ境界、マスキング、制約、テスト観点が反映されていることを確認した。

## 作業仮定

- human notes の「MVPとしては失敗させる」は、成功監査ログが書けない残高変更や権限変更を成功扱いにしない fail closed 方針として扱った。
- human notes の「独立して残す」は、業務拒否および DB transaction rollback 後の失敗監査ログを、業務データ更新と独立したデータベーストランザクションで残す方針として扱った。
- `request_body_hash` は raw request body を保存しないための将来候補属性であり、実際の hash algorithm、対象項目、保存期間は今回確定していない。
- `operator` は MVP 対象外とし、監査ログ照会は `admin` のみに限定する設計案で進めた。
- 成功監査ログを業務データ更新と同じ PostgreSQL データベーストランザクションに含める案は、MVP 初期の学習用設計として記録した。outbox や非同期補償が必要な場合は別 scope で見直す。
