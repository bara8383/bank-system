# ミニバンキングシステム 初期データモデル案

このドキュメントは、MVPで必要になるテーブル候補と主な属性を整理します。実際の実装では使用するフレームワークやデータベースに合わせて型・制約・インデックスを具体化します。

## 前提

- MVPではJPYのみを扱う。
- 金額は整数で保存する。
- 1口座につき1顧客として扱う。
- 外部銀行接続は扱わず、システム内部の口座間振込のみを扱う。
- 取引履歴と監査ログは削除せず、追記型を基本にする。

## テーブル候補

### users

ログインと権限管理のためのテーブルです。

| カラム | 概要 |
| --- | --- |
| id | 内部ユーザーID |
| login_id | ログインID。一意にする。 |
| password_hash | ハッシュ化済みパスワード |
| role | customer、admin、operatorなど |
| status | active、disabledなど |
| last_login_at | 最終ログイン日時 |
| created_at | 作成日時 |
| updated_at | 更新日時 |

### customers

銀行サービスの利用者としての顧客を表します。

| カラム | 概要 |
| --- | --- |
| id | 顧客ID |
| user_id | 顧客本人のログインユーザーID。管理者は紐づかない場合がある。 |
| name | 氏名または名称 |
| email | メールアドレス |
| phone_number | 電話番号 |
| address | 住所 |
| status | active、suspended、closedなど |
| created_at | 登録日時 |
| updated_at | 更新日時 |

### accounts

顧客が保有する口座を表します。

| カラム | 概要 |
| --- | --- |
| id | 内部口座ID |
| account_number | 表示・検索用の口座番号。一意にする。 |
| customer_id | 口座所有者の顧客ID |
| account_type | ordinaryなど。MVPでは普通預金のみ。 |
| currency | JPY |
| balance_amount | 現在残高。整数で保存する。 |
| status | active、suspended、closedなど |
| opened_at | 開設日時 |
| created_at | 作成日時 |
| updated_at | 更新日時 |

### transactions

口座残高を変化させた取引の履歴です。

| カラム | 概要 |
| --- | --- |
| id | 取引ID |
| account_id | 対象口座ID |
| transaction_type | deposit、withdrawal、transfer_debit、transfer_credit、reversalなど |
| amount | 取引金額。正の整数で保存する。 |
| balance_after | 取引後残高 |
| currency | JPY |
| related_transaction_id | 振込の相手側取引や取消元取引への参照 |
| transfer_request_id | 関連する振込依頼ID |
| description | 摘要 |
| occurred_at | 取引発生日時 |
| created_at | 作成日時 |

### transfer_requests

利用者から受け付けた振込依頼と処理状態を表します。

| カラム | 概要 |
| --- | --- |
| id | 振込依頼ID |
| requested_by_user_id | 依頼者ユーザーID |
| source_account_id | 振込元口座ID |
| destination_account_id | 振込先口座ID |
| amount | 振込金額 |
| currency | JPY |
| idempotency_key | 二重送金防止用キー |
| status | accepted、processing、succeeded、failed、cancelledなど |
| failure_reason | 失敗理由 |
| requested_at | 受付日時 |
| processed_at | 処理完了日時 |
| created_at | 作成日時 |
| updated_at | 更新日時 |

### audit_logs

重要操作の追跡に使う監査ログです。

| カラム | 概要 |
| --- | --- |
| id | 監査ログID |
| actor_user_id | 操作したユーザーID |
| action_type | login、create_customer、create_account、deposit、withdrawal、transferなど |
| target_type | customer、account、transaction、transfer_requestなど |
| target_id | 対象ID |
| result | success、failure |
| failure_reason | 失敗理由 |
| ip_address | 操作元IPアドレス |
| user_agent | 操作元User-Agent |
| occurred_at | 操作日時 |
| created_at | 作成日時 |

## 主な制約案

- `users.login_id` は一意にする。
- `accounts.account_number` は一意にする。
- `accounts.balance_amount` は0以上にする。
- `transactions.amount` は0より大きい値にする。
- `transfer_requests.amount` は0より大きい値にする。
- `transfer_requests.idempotency_key` は依頼者または振込元口座の範囲で一意にする。
- `transactions.account_id`、`accounts.customer_id` などの外部キーを設定する。

## 残高と取引履歴の考え方

MVPでは、現在残高を `accounts.balance_amount` に保持し、残高変更の根拠を `transactions` に追記します。これにより残高照会を高速にしつつ、取引履歴から残高更新の理由を確認できます。

将来的には、取引履歴の合計と現在残高が一致するかを定期的に検証する仕組みを追加します。
