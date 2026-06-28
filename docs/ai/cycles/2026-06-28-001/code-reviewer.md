# code-reviewer レビュー結果: 2026-06-28-001

## レビュー種別

- 種別: repo 全体レビュー
- 理由: `git status --short` で作業ツリーに実装差分がなく、`docs/ai/cycles/2026-06-28-001/` 配下にも同一 cycle の implementer 成果物がまだ存在しないため。
- 対象: 現時点のリポジトリ全体。実装コードはまだ存在せず、設計ドキュメントと agent/cycle 運用定義を中心に確認した。

## 前提と制約

- このプロジェクトは学習用ミニバンキングシステムであり、本番金融システム相当の完全性は要求しない。
- ソースコード変更は禁止のため、本レビューでは実装修正を行わず、次サイクル planner への入力として課題を整理する。
- 現時点では Go / PostgreSQL の実装、migration、テストコードが存在しないため、コードレベルのバグ検出ではなく、実装開始前の設計・保守性リスクを中心に評価する。

## Finding 1: 実装の最初の縦切りが未定義で、全機能を同時に始めるリスクがある

- 重大度: Medium

### 根拠

- MVP はユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録までを含んでいる。
- `docs/START_HERE.md` には実装順の候補があるが、次に実装する最小単位・完了条件・対象外を cycle 単位で固定する成果物はまだ存在しない。
- 現在のリポジトリには Go module、API エントリポイント、DB migration、テストがまだない。

### 影響

- implementer が MVP 全体を一度に実装しようとすると、認証、口座、取引、監査、冪等性、DB トランザクションが同時に絡み、レビュー可能な差分が大きくなる。
- 学習用プロジェクトの「小さく実装する」方針に反し、トランザクション境界や責務分離の失敗を早期に発見しにくくなる。

### 推奨修正

- 次 cycle の planner は、最初の accepted scope を「プロジェクト土台 + 顧客/口座の最小読み書き」または「入金だけの縦切り」など、1つの業務フローに限定する。
- accepted scope には、実装する API、DB table、migration、テスト、実装しない機能を明記する。
- 最初から振込まで入れず、DB トランザクションと監査ログの基本形を小さい処理で確立してから拡張する。

### 次サイクル planner への入力

- `go.mod`、アプリケーション起動点、PostgreSQL migration の置き場所、テスト実行方法を含む最小スケルトンを accepted scope 候補にする。
- ただし、振込・冪等性・認証本実装まで同一 scope に含めない。

## Finding 2: データモデル案はあるが、PostgreSQL 制約・インデックス・トランザクション時のロック方針が未具体化

- 重大度: High

### 根拠

- `docs/data-model.md` は `accounts.balance_amount >= 0`、`transactions.amount > 0`、`transfer_requests.idempotency_key` の一意性などを制約案として挙げている。
- 一方で、PostgreSQL の具体的な `CHECK`、`UNIQUE`、`FOREIGN KEY`、検索 index、残高更新時の `SELECT ... FOR UPDATE` または条件付き `UPDATE` の方針は未定義である。
- `docs/design-principles.md` は振込の原子性を求めているが、実装時の repository / transaction manager / isolation level の設計はまだない。

### 影響

- アプリケーション層だけで残高非負や冪等性を守る実装になると、並行実行時に残高がマイナスになる、同じ振込が二重処理される、取引履歴と残高が不整合になるリスクが残る。
- DB 制約が後付けになると、既存データ移行やテスト修正の負担が増える。

### 推奨修正

- 初回 migration 作成時点で、最低限次を DB 制約として入れる。
  - `accounts.account_number` の一意制約。
  - `accounts.balance_amount >= 0` の `CHECK` 制約。
  - `transactions.amount > 0` の `CHECK` 制約。
  - `transfer_requests.amount > 0` の `CHECK` 制約。
  - `transfer_requests` の冪等性キーに対するスコープ付き一意制約。
- 残高更新は、Go の service 層から DB transaction を開始し、repository に transaction context を渡す形にする。
- 出金・振込の残高競合について、口座行ロックまたは条件付き更新のどちらを採るかを設計判断として docs に残す。

### 次サイクル planner への入力

- accepted scope に migration の具体制約を含める。
- 並行出金をまだ実装しない場合でも、残高更新 API の設計で transaction boundary を後から差し替えられるようにする。

## Finding 3: 監査ログが「業務処理と同一トランザクションか、失敗時も残すか」の方針未決定

- 重大度: Medium

### 根拠

- 設計原則とセキュリティメモは重要操作の成功・失敗を監査ログに残す方針を示している。
- 入金、出金、振込などのユースケースでも成功・失敗ログが言及されている。
- しかし、失敗監査ログを業務 DB transaction の内側で書くか、ロールバック後に別 transaction で書くか、書き込み失敗時に業務処理を失敗させるかは未定義である。

### 影響

- 業務 transaction 内に監査ログを書くだけだと、業務処理をロールバックしたとき失敗ログも消える可能性がある。
- 逆に監査ログの永続化失敗で業務処理を止める設計にすると、可用性と監査完全性のトレードオフが実装ごとにばらつく。
- 後から監査方針を変えると、service 層のエラーハンドリングとテスト全体に影響する。

### 推奨修正

- 学習用 MVP として、まずは「成功した残高変更の取引履歴は同一 transaction 内で必須」「失敗監査ログはロールバック後に可能な限り別 transaction で記録」など、明示的な暫定方針を決める。
- 監査ログ書き込み失敗時の扱いを、人間確認事項として分離する。
- service 層の戻り値に、利用者向けエラーと監査用 failure reason を分ける設計を検討する。

### 次サイクル planner への入力

- 監査ログを初回実装に含める場合、成功ログ・失敗ログ・ロールバック時の扱いを accepted scope に明記する。
- 監査ログを次回以降に回す場合でも、service interface に actor / request metadata を渡せる余地を残す。

## Finding 4: Go のレイヤード構成・エラー分類・テスト境界がまだ決まっていない

- 重大度: Medium

### 根拠

- `.codex/agents/code-reviewer.toml` は Go/PostgreSQL、トランザクション境界、レイヤード設計、テスト容易性をレビュー対象にしている。
- `docs/test-strategy.md` は単体・結合・セキュリティ・データ整合性テストの観点を整理している。
- ただし、Go package 構成、domain/service/repository/http handler の責務、エラー型、テストで DB を使う範囲はまだ決まっていない。

### 影響

- handler が直接 SQL と業務ルールを持つ実装になると、認可・残高更新・監査ログの責務が混ざり、金融系で重要な異常系テストが書きにくくなる。
- エラー分類が曖昧だと、残高不足・権限不足・入力不正・DB 障害を API レスポンスや監査ログで安全に出し分けにくい。

### 推奨修正

- 最初の実装前に、最小の package 方針を決める。
  - 例: `cmd/api`、`internal/domain`、`internal/application`、`internal/adapter/http`、`internal/adapter/postgres`。
- 業務ルールは service/application 層に置き、HTTP handler は認証済み actor の取り出し、入力検証、レスポンス変換に寄せる。
- 予期される業務エラーと内部エラーを Go の型または sentinel error で分類し、テスト対象にする。

### 次サイクル planner への入力

- accepted scope に package layout とエラー分類の最小ルールを含める。
- 最初の機能は unit test と PostgreSQL integration test のどちらで検証するかを明記する。

## Finding 5: テスト戦略は十分だが、実行可能なテスト基盤がまだない

- 重大度: Medium

### 根拠

- `docs/test-strategy.md` は金額計算、残高更新、取引履歴、監査ログ、振込原子性、冪等性、認証認可を重要テスト対象として整理している。
- 現時点で `go test ./...` を実行できる Go module やテストファイルは存在しない。
- PostgreSQL を使う結合テストの起動方法や test database の管理方法も未定義である。

### 影響

- 実装が先行すると、残高競合や冪等性の不具合を手動確認に頼る期間が長くなる。
- DB transaction を伴う処理は単体テストだけでは品質を保証しにくく、結合テストの導入が遅れるほどリファクタリングコストが上がる。

### 推奨修正

- 最初の Go 実装と同時に `go test ./...` が通る状態を作る。
- PostgreSQL integration test は、初回から全機能を対象にせず、migration 適用と 1 つの残高変更フローに絞る。
- CI 未導入でも、README または docs にローカルテストコマンドを記録する。

### 次サイクル planner への入力

- accepted scope に「実行可能なテストコマンド」を必ず含める。
- DB が必要なテストを入れる場合は、Docker Compose などの開発用 DB 起動方法も scope に含めるか、明示的に次回以降へ回す。

## 人間確認事項

1. MVP 初回実装は「認証なしの内部 API で口座・入金の土台を作る」のか、「認証基盤を先に作る」のか。セキュリティ観点では認証が重要だが、学習の縦切りとしては口座・入金から始める選択もあり得る。
2. 監査ログ書き込みに失敗した場合、業務処理を失敗させるか、業務処理を優先して別途アラート扱いにするか。
3. PostgreSQL integration test のために Docker Compose を導入してよいか。

## 総評

現時点のドキュメントは、金融システム学習に必要な用語、MVP 範囲、設計原則、データモデル、セキュリティ、テスト観点を広く整理できている。一方で、実装コードがまだないため、次に重要なのは MVP 全体を一括実装することではなく、Go/PostgreSQL の最小スケルトン、DB 制約、transaction boundary、テストコマンドを小さい accepted scope に落とすことである。
