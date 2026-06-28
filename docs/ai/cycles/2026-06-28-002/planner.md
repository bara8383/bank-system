# planner: 2026-06-28-002

## repo現状

### 作業開始時に確認したもの

- `git status --short`: 作業開始時点では未コミット変更なし。
- `AGENTS.md`: 学習用の銀行・金融システムであり、Go + REST + PostgreSQL を前提とする。ただし他の技術は未定義。作業ルールとして「小さく実装する」「実装前に既存コードを確認する」「設計判断は docs/ に記録する」「README を最新状態に保つ」が示されている。
- README: `README.md` / `README` / `README.*` は存在しない。repo root の入口がないため、次の実装 scope では README 作成を許可する必要がある。
- `docs/START_HERE.md`: 最初のゴールは、顧客登録、口座作成、入金、出金、振込、残高・取引履歴照会、監査ログを安全に扱えるミニバンキングシステムであることを確認。
- `docs/mvp.md`: MVP はユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録を含む。MVP 完了条件として、残高更新、取引履歴、監査ログ、異常系拒否、振込原子性、冪等性、主要テストが必要。
- `docs/domain-model.md`: 顧客、ログインユーザー、口座、残高、取引、振込依頼、監査ログ、認証、認可、トランザクションなどの用語を確認。
- `docs/data-model.md`: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs` の初期テーブル候補と、金額整数、残高非負、冪等性キー一意制約案を確認。
- `docs/use-cases.md`: UC-001 から UC-008 までの正常系・異常系を確認。特に入出金・振込では、口座状態、権限、正の整数金額、残高不足、冪等性、ロールバックが重要。
- `docs/design-principles.md`: 残高非負、金額整数、取引履歴、監査ログ、認証認可、原子性、二重実行防止、状態遷移、調査可能なエラーを確認。
- `docs/security-notes.md`: パスワード保存、セッション/トークン、認可、監査、秘密情報、入力検証、今後の対策を確認。
- `docs/test-strategy.md`: 金額計算、残高更新、取引履歴、監査ログ、異常系、振込原子性、冪等性、認証認可を重点的にテストする方針を確認。
- `docs/ai/cycles/README.md`: planner が accepted scope を作成し、implementer は同一 cycle の accepted scope を参照する運用を確認。
- `docs/ai/output/README.md`: human notes の置き場を確認。現時点で `docs/ai/output/human/` は存在せず、追加の人間メモはない。
- `docs/decision-logs/2026-06-27-subagent-parallel-cycle.md`: planner / implementer / reviewer の cycle artifact protocol が採択済みであることを確認。
- 過去 cycle `2026-06-28-001`: `planner.md`、`implementer.md`、`code-reviewer.md`、`security-reviewer.md`、`banking-reviewer.md` を確認。
- 既存コード: Go ソース、`go.mod`、`go.sum`、SQL、migration、Docker、Makefile、CI 設定、テストコードは見つからない。
- TODO/FIXME: 明示的な `TODO` / `FIXME` は見つからない。未定義・未実装の事項は、既存 docs と cycle 001 reviewer 出力に多数記録されている。

### 実装済み

- アプリケーションコード、DB schema、migration、REST API、テストはまだ存在しない。
- 設計ドキュメント群と AI cycle 運用ドキュメントは存在する。
- cycle 001 の implementer は、同一 cycle の accepted scope を確認できなかったとして `blocked: accepted scope not found` を記録しており、コード実装は行っていない。

### 設計済みだが未実装

- MVP 業務機能: ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- 初期データモデル候補: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs`。
- 重要原則: 金額整数、残高非負、残高変更と取引履歴の整合性、監査ログ、認証認可、振込の原子性、冪等性、状態遷移。
- cycle 001 planner では「最小 Go REST API の土台」が accepted scope として採択されたが、並列実行タイミングのため implementer には反映されていない。

### 未設計または具体化不足

- Go module 名、package layout、HTTP server 起動方法、エラー分類、設定管理。
- PostgreSQL migration ツール、DB 接続方法、transaction manager、repository 境界、ローカル DB 起動方法。
- 認証方式、パスワードハッシュ方式、セッション/トークン、CSRF、ログアウト、レート制限。
- RBAC の権限表、管理者・運用担当者の責務分離。
- `transaction_type` ごとの残高増減方向、`balance_after` の検証ルール。
- 冪等性キーの一意スコープ、リクエスト同一性検証、同一キー異内容時の扱い、保存期間。
- 振込ステータス遷移、失敗した振込依頼の保存境界、処理結果不明時の扱い。
- 並行出金・並行振込時の残高保護方式、ロック順序、デッドロック回避方針。
- 監査ログの transaction 境界、失敗時ログ、書き込み失敗時の業務処理継続可否、マスキング規則。
- API 入力検証、検索制限、エラー応答形式、ログ出力規則、データ分類。

### docs/実装不一致

- docs は MVP と設計方針を示しているが、実装が存在しないため、現在は「設計のみ存在する」状態。
- README が存在せず、AGENTS.md の「README を最新状態に保つ」と、初回参加者向けの repo root 導線が満たされていない。
- cycle 001 planner の accepted scope は未実装であり、cycle 001 implementer の blocked 記録と状態がずれている。ただし、これは並列実行上のタイミング差であり、既存ファイルを revert する必要はない。

### レビュー未反映

- cycle 001 code-reviewer: 最初の縦切り、Go/PostgreSQL スケルトン、DB 制約、transaction boundary、レイヤード構成、実行可能なテスト基盤が未整備。
- cycle 001 security-reviewer: 認証、認可、監査ログ、並行実行、冪等性、データ分類、API 入力検証、シークレット管理が未具体化。
- cycle 001 banking-reviewer: 元帳・残高方向、冪等性、振込状態遷移、競合更新、監査ログ境界が未具体化。

## 入力レビュー

### human notes

- `docs/ai/output/human/` は存在しないため、追加の human notes はない。

### cycle 001 implementer

- `docs/ai/cycles/2026-06-28-001/implementer.md` は、accepted scope が確認できなかったため実装を blocked として終了している。
- そのため、cycle 001 planner で採択された「最小 Go REST API の土台」はまだ実装されていない。
- 今回は同じ失敗を避けるため、accepted scope を本ファイル内で明確にし、実装対象・非対象・テスト方針を細かく区切る。

### cycle 001 code-reviewer

- 実装コードがないため repo-wide review が行われた。
- 次の planner への入力として、`go.mod`、アプリケーション起動点、PostgreSQL migration の置き場所、テスト実行方法を含む最小スケルトンが候補化されている。
- ただし、振込・冪等性・認証本実装を同一 scope に含めないことが推奨されている。
- package layout、エラー分類、テスト境界を小さく決める必要がある。

### cycle 001 security-reviewer

- 認証方式、RBAC、監査ログ、並行実行、冪等性、データ分類、API 入力検証、シークレット管理の未確定事項が整理された。
- 最初の実装が認証や残高更新に踏み込む場合は、先にセキュリティ仕様を固定する必要がある。
- 一方、ヘルスチェック中心の最小 REST 土台であれば、認証方式や金融仕様を確定せずに進められる。ただし、秘密情報・環境情報をレスポンスに出さないことは必須。

### cycle 001 banking-reviewer

- 元帳、残高方向、冪等性、振込状態遷移、競合更新、監査ログ境界について、実装前に具体化すべき金融事故リスクが示された。
- これらは入出金・振込・DB schema に入る前の重要論点であり、今後の accepted scope 候補にする。
- 今回採択する土台実装では残高・取引履歴・振込・監査ログには触れず、金融仕様を暗黙に確定しない。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. 最小 Go REST API 土台と README を作る | AGENTS.md は Go + REST、DB は PostgreSQL を前提とする。cycle 001 planner も同内容を採択したが未実装。code-reviewer も最小スケルトンとテスト基盤を候補化している。 | `go.mod`、起動可能な HTTP server、ヘルスチェック、テスト、README がない。 | 後続の業務機能を小さく追加・レビューするための最小の実行単位が必要。 | code-reviewer: package 構成、テスト容易性、過剰設計回避。security-reviewer: 情報露出なし。banking-reviewer: 金融仕様を暗黙に確定しない。 | DB、認証、顧客、口座、入出金、振込、監査ログは実装しない。README は現状とテストコマンドに限定する。 |
| B. PostgreSQL migration と初期 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。code-reviewer は DB 制約の具体化を推奨。 | migration ツール、ID 型、制約、index、DB 起動方法が未定義。 | 残高非負、冪等性、取引履歴整合性の土台になる。 | banking-reviewer: 残高方向、競合更新、冪等性。security-reviewer: 個人情報・権限・シークレット。 | schema は後戻りしにくく、残高・冪等性・監査境界が未整理のため今回採択しない。 |
| C. 元帳・残高方向・transaction 方針を docs に追加する | banking-reviewer が `transaction_type` と残高増減方向、整合性検証、競合更新を重要入力としている。 | 取引種別の符号、`balance_after`、ロック方式、デッドロック回避が未定義。 | 入出金・振込の前に金融事故リスクを下げる。 | banking-reviewer: 元帳整合性。code-reviewer: transaction boundary。 | 実装基盤がまだなく、今回はまず実行可能な skeleton を優先。次サイクル以降で高優先。 |
| D. 認証・RBAC・セッション管理方針を docs に追加する | security-reviewer が認証方式・認可モデルの未確定を High としている。 | パスワードハッシュ、セッション/トークン、CSRF、ロール権限表が未定義。 | MVP の全業務 API は認証認可に依存する。 | security-reviewer: 認証強度、水平権限不備、管理者権限過大。 | 安全上重要な仕様で人間確認が必要。今回の土台実装では認証を実装しない。 |
| E. 監査ログ・データ分類・API エラー標準を docs に追加する | security-reviewer と banking-reviewer が監査ログ境界、マスキング、入力検証、エラー応答を指摘。 | 成功/失敗ログ、機微情報マスキング、検索制限、利用者向け/運用向けエラー分離が未定義。 | 監査性・調査可能性・情報漏えい防止に必要。 | security-reviewer: 秘密情報と個人情報保護。banking-reviewer: 失敗時証跡。 | 監査ログ書き込み失敗時の業務扱いは人間確認事項。今回採択しない。 |
| F. README のみ作成する | README が存在せず、AGENTS.md は README 最新化を求めている。 | repo root の入口、現状、テストコマンド案がない。 | 学習用 repo のオンボーディングに有用。 | code-reviewer: docs と実装の一致。 | README 単独より、最小 Go skeleton と同時に「実際に動くコマンド」を載せる方が価値が高い。A に含める。 |

## 採択

### 採択: A. 最小 Go REST API 土台と README を作る

- 理由: 現在は実装コードがなく、cycle 001 の accepted scope も未実装である。MVP の金融機能に入る前に、Go + REST の最小起動単位、テスト実行基盤、repo root の README を用意することが、最も小さく、後戻りリスクが低い。
- code-reviewer 入力への対応: `go.mod`、アプリケーション起動点、handler test、`go test ./...` を作ることで、実行可能な最小スケルトンとテスト基盤を作る。
- security-reviewer 入力への対応: `/healthz` は秘密情報、環境変数、DB 接続情報、内部パス、stack trace を返さない。認証方式・セッション方式は確定しない。
- banking-reviewer 入力への対応: 残高、取引履歴、振込、監査ログ、冪等性、DB transaction には触れず、金融仕様を暗黙に確定しない。
- README への対応: AGENTS.md の「README を最新状態に保つ」に従い、実装後の README にはプロジェクト概要、現状、起動方法、テスト方法、未実装の金融機能を明記する。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| F. README のみ作成する | README は必要だが、実装がない状態で README だけを作っても検証可能な進捗になりにくい。A に含めて、実際の起動・テストコマンドと同期させる。 | 実装変更が許可されない docs-only cycle の場合。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| B. PostgreSQL migration と初期 schema を作る | DB schema は後戻りしにくい。元帳方向、冪等性スコープ、監査境界、認証との関係が未整理のまま固定しない。 | migration ツール、ID 型、口座番号採番、個人情報の保存項目、冪等性キーの一意スコープ。 | Go REST 土台後、docs で DB 方針を具体化してから小さい migration scope にする。 |
| C. 元帳・残高方向・transaction 方針を docs に追加する | 入出金・振込前には必須だが、今回の healthz skeleton は金融データに触れないため先送り可能。 | `reversal` を MVP 前に定義するか、残高競合制御を行ロックか条件付き UPDATE のどちらにするか。 | 次 cycle 以降で高優先候補にする。 |
| D. 認証・RBAC・セッション管理方針を docs に追加する | 認証方式は安全上重要で、人間確認なしに最終確定しない。今回の実装は認証不要の healthz のみに限定する。 | Cookie session か Bearer token か、パスワードハッシュ方式、CSRF、管理者作成方法、運用担当者ロールの有無。 | 業務 API 実装前に docs scope として採択する。 |
| E. 監査ログ・データ分類・API エラー標準を docs に追加する | 監査ログは重要だが、今回の skeleton では監査対象業務操作がない。 | 監査ログ書き込み失敗時に業務処理を止めるか、失敗ログを業務 DB と同じ DB に置くか。 | 認証・業務 API または DB schema の前に docs scope として採択する。 |

## accepted scope

### 目的

- 学習用ミニバンキングシステムの最初の実装として、業務機能に入る前の最小 Go REST API 土台を作る。
- 後続で顧客、口座、入出金、振込、取引履歴、監査ログを追加できるよう、起動可能・テスト可能な最小構成に限定する。
- README を作成し、repo の現状、起動方法、テスト方法、未実装範囲を明示する。
- 金融仕様、DB schema、認証方式、監査ログ方式は確定しない。

### 対象ファイル/領域

- Go module / アプリケーション起動に必要な最小ファイル。
  - 例: `go.mod`
  - 例: `cmd/server/main.go` または同等の小さな entrypoint
  - 例: `internal/http`、`internal/server`、`internal/app` など、標準ライブラリで handler をテストしやすくする最小 package
- 最小テスト。
  - 例: `/healthz` handler の unit test
- README。
  - 例: `README.md`
- 実装結果を記録する同一 cycle の implementer 成果物。
  - `docs/ai/cycles/2026-06-28-002/implementer.md`

### 実装対象

1. `go mod init` 相当で Go module を作る。
   - module 名は repo 名に合わせたローカルで妥当な名前に留める。
   - 外部依存は追加しない。標準ライブラリのみを使う。
2. 標準ライブラリ `net/http` で起動できる最小 HTTP server を作る。
   - `/healthz` エンドポイントを 1 つ提供する。
   - HTTP method は `GET` を基本にし、それ以外の method の扱いは標準的な `405 Method Not Allowed` または明示的な小さい実装にする。
   - レスポンスは固定の安全な JSON または plain text とする。
   - レスポンスに環境変数、DB 接続情報、秘密情報、内部ファイルパス、stack trace、ホスト固有情報を含めない。
3. server / handler をテストしやすいように分離する。
   - `main` に handler をすべて直書きしない。
   - handler 生成関数、router 生成関数、またはそれに相当する小さな関数を作り、`httptest` で直接検証できるようにする。
4. 最小テストを追加する。
   - `/healthz` が成功ステータスを返すこと。
   - レスポンス body が想定どおりであること。
   - レスポンスが秘密情報や不要な詳細を含まないことを、少なくとも固定値の確認で担保する。
5. `go test ./...` が実行できる状態にする。
6. `README.md` を作成する。
   - プロジェクトが学習用ミニバンキングシステムであること。
   - 現在実装済みなのは最小 server と health check のみであること。
   - `go test ./...` と server 起動コマンドを記載すること。
   - 顧客、口座、入出金、振込、DB、認証認可、監査ログは未実装であること。
   - 実際の金融機関向け本番システムではないこと。
7. `docs/ai/cycles/2026-06-28-002/implementer.md` に、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

### 実装しないこと

- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログの業務 API は実装しない。
- PostgreSQL 接続、DB schema、migration、repository、transaction 処理は実装しない。
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション、トークン、CSRF、ログアウトは実装しない。
- 口座番号採番、冪等性キー処理、残高更新、取引履歴、監査ログの詳細仕様は確定しない。
- Docker、CI、lint、外部フレームワーク、外部ライブラリ、OpenAPI 仕様は導入しない。
- cycle 001 の成果物は編集しない。

### テスト方針

- `go test ./...` を実行する。
- handler の unit test では `net/http/httptest` を使う。
- DB、外部ネットワーク、外部サービス、Docker を必要とするテストは作らない。
- server の手動起動確認をする場合は、長時間常駐プロセスを残さない。
- README に記載したコマンドと実際のテストコマンドを一致させる。

### レビューで重点確認してほしい観点

- code-reviewer:
  - Go module / package 構成が過剰でなく、後続の業務機能追加に耐えるか。
  - `main` と handler が分離され、`httptest` で検証できるか。
  - 標準ライブラリ中心で不要な依存を増やしていないか。
  - README の起動・テスト手順が実装と一致しているか。
- security-reviewer:
  - `/healthz` が秘密情報、環境情報、内部パス、stack trace を返していないか。
  - 将来の認証導入を妨げる公開 API 設計になっていないか。
  - README が本番金融システムとして誤解される表現になっていないか。
- banking-reviewer:
  - 今回の実装が金融ドメイン仕様を暗黙に確定していないか。
  - 残高、取引履歴、監査ログ、冪等性、DB transaction 境界が未実装であることが明確か。
  - 次 cycle で元帳・残高方向・冪等性・監査境界を扱う必要が見落とされていないか。

## 実装しないこと

- planner として、ソースコード、DB schema、認証方式、金融仕様の実装・最終決定は行わない。
- 本ファイル `docs/ai/cycles/2026-06-28-002/planner.md` 以外へ書き込まない。
- cycle 001 の成果物を修正しない。
- ユーザー作業や他 agent 作業を revert しない。
- 保留事項や人間確認事項を accepted scope に混ぜない。

## 人間確認事項

1. Go module 名は何にするか。未指定の場合、implementer は repo 名ベースの暫定名を使う可能性がある。
2. 次に DB を扱う cycle では、migration ツールを採用するか。候補は手書き SQL、`golang-migrate`、`goose` など。
3. MVP 初期の認証方式を Cookie session にするか、Bearer token にするか。パスワードハッシュ方式、CSRF、トークン失効、ログアウトも合わせて決める必要がある。
4. 管理者が顧客の入金・出金・振込を代行できるか。運用担当者ロールを MVP に含めるか。
5. 冪等性キーの一意スコープを `requested_by_user_id` 基準、`source_account_id` 基準、または `requested_by_user_id + source_account_id` 基準のどれにするか。
6. 監査ログ書き込みに失敗した場合、残高変更などの業務処理を失敗させるか、業務処理を優先して別途アラート・再記録対象にするか。
7. 出金・振込時の残高競合制御は、PostgreSQL の行ロックを使うか、条件付き UPDATE を使うか。
8. 取消・訂正を MVP で扱わない場合でも、将来のために `reversal` の意味と残高方向だけ先に定義するか。
