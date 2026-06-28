# planner: 2026-06-28-001

## repo現状

### 作業開始時に確認したもの

- `git status --short`: 作業開始時点では未コミット変更なし。
- README: `README.md` / `README*` は存在しない。AGENTS.md は README 最新化を求めているが、今回のユーザー指示により出力先は本ファイルのみに限定されるため README 作成・更新は行わない。
- `AGENTS.md`: 学習用の銀行・金融システムであり、Go + REST + PostgreSQL を前提とする。ただし他の技術は未定義。
- `docs/START_HERE.md`: ミニバンキングシステムの段階的な進め方と、最初のゴールとして顧客登録、口座作成、入金、出金、振込、残高・取引履歴照会、監査ログを扱う方針を確認。
- `docs/mvp.md`: MVP 機能、異常系、対象外機能、完了条件を確認。
- `docs/domain-model.md`: 顧客、ログインユーザー、口座、残高、取引、振込依頼、監査ログなどの用語定義を確認。
- `docs/data-model.md`: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs` の初期テーブル候補と制約案を確認。
- `docs/use-cases.md`: UC-001 から UC-008 までの正常系・異常系を確認。
- `docs/design-principles.md`: 金額整数、残高非負、取引履歴、監査ログ、認証認可、原子性、冪等性、状態遷移を確認。
- `docs/security-notes.md`: 認証、認可、監査、秘密情報、入力検証、今後検討対策を確認。
- `docs/test-strategy.md`: 金額・残高更新、取引履歴、監査ログ、異常系、原子性、冪等性、認証認可を重視する方針を確認。
- `docs/ai/cycles/README.md`: cycle artifact protocol と並列ルールを確認。
- `docs/ai/output/README.md`: human notes の置き場を確認。現時点で `docs/ai/output/human/` は存在しない。
- 既存 cycle: `docs/ai/cycles/README.md` のみで、過去 cycle の planner / implementer / reviewer 出力は未確認。前 cycle のレビュー未反映事項は存在しない扱い。
- 既存コード: Go ソース、`go.mod`、migration、SQL、API handler、service/usecase、repository、テストは見つからない。実装はまだ開始前。
- TODO/FIXME: 明示的な TODO/FIXME は見つからない。`AGENTS.md` と `docs/START_HERE.md` に、技術スタック詳細や実装順が未定義・未着手であることが示されている。

### 実装済み

- 実装済みのアプリケーションコード、DB schema、migration、テストはなし。
- AI subagent / cycle 運用に関するドキュメントと agent 定義は存在する。

### 設計済みだが未実装

- MVP の業務機能: ユーザー登録、認証、顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ記録。
- 初期データモデル候補: `users`、`customers`、`accounts`、`transactions`、`transfer_requests`、`audit_logs`。
- 重要な金融原則: 金額整数、残高非負、残高変更と取引履歴の整合性、監査ログ、認証認可、振込の原子性、冪等性。

### 未設計または具体化不足

- Go プロジェクト構成、モジュール名、依存ライブラリ、REST API のエンドポイント詳細。
- DB migration ツール、PostgreSQL 接続方法、トランザクション境界の実装方式。
- 認証方式、セッション / トークン方式、パスワードハッシュ方式、ロール詳細。
- 口座番号採番ルール、監査ログ保持方針、取消 / 組戻し、並行更新ロック方針。
- エラーコード、レスポンス形式、ログ形式、設定管理、ローカル起動方法。

### docs/実装不一致

- docs は MVP と設計方針を示しているが、実装が存在しないため、現時点では「不一致」というより「設計のみ存在する」状態。
- README が存在しないため、AGENTS.md の「README を最新状態に保つ」と docs の入口が repo root で案内されていない点は不足。ただし今回の出力先制約により README 更新は accepted scope に含めない。

### レビュー未反映

- 過去 cycle の reviewer 出力が存在しないため、レビュー未反映事項はなし。

## 入力レビュー

- human notes: `docs/ai/output/human/` が存在しないため入力なし。
- reviewer 出力: 過去 cycle の `code-reviewer.md`、`security-reviewer.md`、`banking-reviewer.md` は存在しないため入力なし。
- decision log: `docs/decision-logs/2026-06-27-subagent-parallel-cycle.md` により、planner が accepted scope を作り、implementer は同一 cycle の accepted scope のみを実装する運用が採択済み。

## 改善候補

| 候補 | repo 上の根拠 | 現在の不足 | MVP に入れる理由 | reviewer 観点 | 実装時の注意 |
| --- | --- | --- | --- | --- | --- |
| A. 最小 Go REST API の土台を作る | AGENTS.md は BE: Go, REST、DB: PostgreSQL を前提とする。docs は Phase 6 の最初に技術スタック決定を挙げている。 | `go.mod`、起動可能な HTTP サーバー、ヘルスチェック、テストがない。 | 業務機能実装前に小さく動く単位とテスト実行基盤が必要。 | code-reviewer: 構成、テスト容易性、依存最小化。security-reviewer: 不要な情報露出がないか。 | 金融仕様や DB schema を確定しない。ヘルスチェック程度に留める。 |
| B. DB migration と初期 schema を作る | `docs/data-model.md` にテーブル候補と制約案がある。 | 実 DB schema、migration、制約、index がない。 | 残高・取引履歴・冪等性の土台になる。 | banking-reviewer: 残高非負、取引履歴、冪等性制約。code-reviewer: migration 管理。security-reviewer: 個人情報・秘密情報。 | schema 確定は後戻りしにくいため、口座番号や認証方式など人間確認が必要。今回採択しない。 |
| C. 認証方式を決めてユーザー登録を実装する | `docs/mvp.md` はユーザー登録と認証を MVP に含める。 | 認証方式、ハッシュ方式、セッション管理が未決定。 | 以後の認可境界に必要。 | security-reviewer: パスワード保存、トークン、ログ漏洩。code-reviewer: middleware 設計。 | 認証方式は安全上重要な仕様なので人間確認なしに確定しない。今回採択しない。 |
| D. README を作成して docs 入口と現状を示す | AGENTS.md は README 最新化を求めるが README がない。 | repo root の入口がない。 | 実装前のオンボーディングに有用。 | code-reviewer: docs と実装の一致。 | ユーザー指示で出力先は本ファイルのみのため今回実施不可。 |
| E. 同一 cycle の reviewer が repo-wide review を行う | cycle protocol は reviewer 出力を次 cycle の入力にする。 | 初回 reviewer 出力がない。 | 実装前に設計リスクを洗い出せる。 | 各 reviewer: 専門観点の初回レビュー。 | planner は reviewer ファイルを作らない。別 agent に任せる。 |

## 採択

### 採択: A. 最小 Go REST API の土台を作る

- 理由: 現在は実装が存在せず、MVP の業務機能に入る前に、Go + REST の最小起動単位とテスト実行基盤を作るのが最も小さく、安全で、後続作業の足場になるため。
- 金融ドメイン影響: 直接の残高・取引履歴・監査ログ・冪等性には触れない。金融仕様を確定しないため、後戻りリスクが小さい。
- セキュリティ影響: 認証方式や秘密情報管理は実装しない。ヘルスチェックは内部状態や環境変数を漏らさないこと。
- DB 影響: PostgreSQL 接続、migration、schema は実装しない。DB トランザクション境界は次以降の scope で扱う。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| D. README を作成して docs 入口と現状を示す | 有用だが、今回のユーザー指示で出力先は `docs/ai/cycles/2026-06-28-001/planner.md` のみに限定されているため。 | 次 cycle 以降で出力先制約がなく、README 更新が許可された場合。 |

## 保留

| 候補 | 保留理由 | 人間確認事項 | 次のアクション |
| --- | --- | --- | --- |
| B. DB migration と初期 schema を作る | schema、制約、採番、認証との関連は後戻りしにくい。実装基盤なしに先に固定すると変更コストが高い。 | migration ツール、ID 型、口座番号採番、冪等性キーの一意範囲、個人情報の最小項目。 | Go REST API 土台とテスト基盤後に、schema 案を小さく accepted scope 化する。 |
| C. 認証方式を決めてユーザー登録を実装する | 認証方式とパスワードハッシュ方式は安全上重要で、人間確認なしに最終確定しない。 | セッション方式かトークン方式か、パスワードハッシュ方式、ロール初期値、管理者作成方法。 | 人間確認または設計ドキュメント更新後に scope 化する。 |
| E. 同一 cycle の reviewer が repo-wide review を行う | planner の出力先ではなく reviewer agent の責務。 | なし。 | code-reviewer / security-reviewer / banking-reviewer が同一 cycle に出力する。 |

## accepted scope

### 目的

- 学習用ミニバンキングシステムの最初の実装として、業務機能に入る前の最小 Go REST API 土台を作る。
- 後続で顧客、口座、入出金、振込、監査ログを追加できるよう、起動可能・テスト可能な最小構成に限定する。
- 金融仕様、DB schema、認証方式は確定しない。

### 対象ファイル/領域

- Go module / アプリケーション起動に必要な最小ファイル。
  - 例: `go.mod`
  - 例: `cmd/server/main.go`
  - 例: `internal/http` または同等の最小 HTTP handler パッケージ
- 最小テスト。
  - 例: HTTP handler の unit test / `go test ./...` が通る構成
- 実装結果を記録する同一 cycle の implementer 成果物。
  - `docs/ai/cycles/2026-06-28-001/implementer.md`

### 実装対象

1. `go mod init` 相当で Go module を作る。
   - module 名は repo 名に合わせたローカルで妥当な名前に留める。
   - 外部依存は追加しないか、標準ライブラリのみを優先する。
2. 標準ライブラリ `net/http` で起動できる最小 HTTP server を作る。
   - `/healthz` などのヘルスチェックエンドポイントを 1 つ提供する。
   - レスポンスは固定の安全な JSON または plain text とし、環境変数、DB 接続情報、秘密情報、内部パスを出さない。
3. server / handler をテストしやすいように分離する。
   - `main` にすべてを直書きせず、handler 生成関数などを別パッケージまたは別関数に分ける。
4. 最小テストを追加する。
   - `/healthz` が成功ステータスを返すこと。
   - レスポンスが想定どおりで、機密情報や不要な詳細を含まないこと。
5. `go test ./...` が実行できる状態にする。
6. `docs/ai/cycles/2026-06-28-001/implementer.md` に、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、未確認事項を記録する。

### 実装しないこと

- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログの業務 API は実装しない。
- PostgreSQL 接続、DB schema、migration、repository、transaction 処理は実装しない。
- 認証、認可、ユーザー登録、パスワードハッシュ、セッション / トークンは実装しない。
- 口座番号採番、冪等性キー処理、残高更新、取引履歴、監査ログの詳細仕様は確定しない。
- README 作成・更新は今回の accepted scope に含めない。今回のユーザー指示により planner の書き込み先が本ファイルに限定されているため、implementer が README を変更するかは別途ユーザー許可がある場合のみ。
- Docker、CI、lint、外部依存導入、フレームワーク導入は行わない。

### テスト方針

- `go test ./...` を実行する。
- handler の unit test では `httptest` を使い、HTTP status と body を確認する。
- 起動確認が必要な場合でも、長時間常駐する server を残さない。
- DB や外部ネットワークを必要とするテストは作らない。

### レビューで重点確認してほしい観点

- code-reviewer:
  - Go module / package 構成が過剰でなく、次の業務機能追加に耐えるか。
  - `main` と handler が分離され、テスト容易性があるか。
  - 標準ライブラリ中心で不要な依存を増やしていないか。
- security-reviewer:
  - `/healthz` が秘密情報、環境情報、内部パス、stack trace を返していないか。
  - 将来の認証導入を妨げる公開 API 設計になっていないか。
- banking-reviewer:
  - 今回の実装が金融ドメイン仕様を暗黙に確定していないか。
  - 残高、取引履歴、監査ログ、冪等性、DB トランザクション境界に未実装であることが明確か。

## 実装しないこと

- planner として実装、ソースコード変更、DB schema 確定、認証方式確定、金融仕様の最終決定は行わない。
- 本ファイル以外への書き込みは行わない。
- 他 agent の同時作業や未コミット変更を revert しない。
- 保留事項を accepted scope に混ぜない。

## 人間確認事項

1. Go module 名は何にするか。未指定の場合、implementer は repo 名ベースの暫定名を使う可能性がある。
2. README が存在しないため、次 cycle 以降で README 作成を許可するか。
3. DB migration ツールを採用するか。採用する場合、`golang-migrate`、`goose`、手書き SQL などの候補から選ぶ必要がある。
4. 認証方式をどうするか。セッション Cookie、JWT、その他の方式は安全上重要なため人間確認が必要。
5. 口座番号採番ルール、ID 型、冪等性キーの一意範囲、監査ログ保持期間、取消 / 組戻しの扱いは後戻りしにくいため、実装前に別途設計判断が必要。
