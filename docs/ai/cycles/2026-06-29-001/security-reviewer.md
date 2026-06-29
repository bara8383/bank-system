# security-reviewer: 2026-06-29-001

## レビュー対象

- `docs/ai/cycles/2026-06-29-001/planner.md`
- `docs/ai/cycles/2026-06-29-001/implementer.md`
- 実装差分: `docs/design-principles.md`, `docs/security-notes.md`, `docs/data-model.md`, `docs/test-strategy.md`, `docs/ai/cycles/2026-06-29-001/implementer.md`

## 確認した前提

- `.codex/agents/README.md` と `docs/ai/cycles/README.md` の artifact protocol を確認した。
- `.codex/agents/security-reviewer.toml` の security-reviewer 役割、禁止事項、出力契約を確認した。
- `.agents/skills/banking-security-review/SKILL.md` を参照し、差分レビューを優先した。
- `AGENTS.md`, `README.md`, `docs/START_HERE.md`, `docs/design-principles.md`, `docs/security-notes.md`, `docs/data-model.md`, `docs/use-cases.md`, `docs/test-strategy.md`, `docs/ai/output/human/001-human-review.md` を確認した。
- `git status --short` はレビュー開始時点で未コミット変更なしだった。
- 現行実装は `GET /healthz` の固定応答と HTTP server skeleton のみで、認証、認可、DB、業務 API、監査ログのコード実装はまだ存在しない。

## 総評

今回の差分は docs-only で、前回 security-reviewer が指摘していた「成功監査ログの transaction 境界」「失敗時監査ログの rollback 独立性」「監査ログ書き込み失敗時の fail closed」「監査ログ閲覧ロール」「マスキング方針」を明確化している。残高変更を伴う成功操作では成功監査ログを同じ PostgreSQL transaction に含める案、業務拒否や rollback 後の失敗監査ログを独立 transaction で残す案は、学習用 MVP として安全側の前提に寄っており、accepted scope に概ね適合している。

一方で、監査ログに入れる `ip_address` / `user_agent` / `request_body_hash` の信頼境界と正規化、冪等性キーとの関係、管理者代行時の actor / subject 分離は、今後の認証・認可・監査ログ実装前に詰めないと、証跡汚染、リプレイ・二重送金、説明責任不足につながる余地がある。

## Finding 1: 監査ログの `ip_address` / `user_agent` の信頼境界と正規化ルールが未定義

- 重大度: Medium
- 観点: ログ、秘密情報、監査証跡、入力検証
- 作業仮定のリスク: `user_agent` や proxy 経由の送信元 IP をそのまま「監査に使える値」として保存する前提になると危険。

### 根拠

- `docs/security-notes.md` は監査ログ項目として `ip_address` と `user_agent` を含める方針を追加している。
- 同じ文書は監査ログに password、token、secret、CSRF token、セッション ID、raw request body、過剰な個人情報を保存しない方針も追加している。
- `docs/data-model.md` でも `audit_logs.ip_address` と `audit_logs.user_agent` を属性候補にしているが、最大長、制御文字の扱い、改行・タブ・NUL の除去、ヘッダー値の切り詰め、保存前の正規化、信頼する reverse proxy / `X-Forwarded-For` の範囲は未定義のまま。
- 現在の Go 実装には監査ログ処理がないため、今回の docs が将来の実装仕様として参照される可能性が高い。

### 影響

`User-Agent` はクライアントが任意に送れるため、攻撃者が長大な値、改行を含む値、偽の JSON 断片、token 文字列、個人情報を混ぜた値を送ると、監査ログ汚染やログ注入、保管容量の圧迫、秘密情報混入が起き得る。送信元 IP も、信頼していない proxy header をそのまま採用すると、攻撃者が任意の IP を監査ログに残せる。

攻撃または事故シナリオ:

1. 攻撃者が `User-Agent` に改行と偽の監査イベント風文字列、または token らしき値を含めて振込 API を呼ぶ。
2. 実装者が docs の `user_agent` 項目を raw header 保存として実装する。
3. 監査ログ検索や SIEM 連携時に偽イベントや秘密情報が混入し、調査担当者が actor / 操作結果を誤認する。
4. さらに `X-Forwarded-For` を無条件に信頼している場合、攻撃元 IP の追跡も誤る。

### 推奨修正

次サイクルで、監査ログの入力正規化ルールを docs に追加する。

- `user_agent` は raw header をそのまま信頼せず、最大長、文字集合、制御文字除去、改行除去、切り詰め、空値時の扱いを定義する。
- `ip_address` は `RemoteAddr` と proxy header のどちらを使うか、信頼する proxy がある場合だけ `Forwarded` / `X-Forwarded-For` を採用することを明記する。
- 監査ログへ保存するヘッダー系情報にも「secret/token/password/CSRF/session ID を含めない」方針を適用し、値のマスキングまたは破棄をテスト観点に入れる。
- 大量・長大 header による保存領域圧迫を防ぐため、DB schema 実装前にカラム長と切り詰め方針を決める。

### 次サイクル planner への入力

認証 / 監査ログ実装前の docs-only scope として、「監査ログ属性の正規化・信頼境界・カラム長・マスキングテスト」を採択候補にする。対象は `docs/security-notes.md`, `docs/data-model.md`, `docs/test-strategy.md`。実装に進む場合は middleware / audit logger の単体テストで、改行入り `User-Agent`、長大 header、偽装 `X-Forwarded-For`、token 風文字列を検証対象にする。

## Finding 2: `request_body_hash` が監査ログ候補に留まり、冪等性キーの安全境界とまだ接続されていない

- 重大度: Medium
- 観点: 冪等性、認可、監査証跡、二重送金防止
- 作業仮定のリスク: `request_body_hash` を audit-only の調査項目として追加しただけでは、同一冪等性キーの異内容リクエストや replay を防げない。

### 根拠

- human notes は、冪等性には操作種別、送信元口座、ログインユーザー、リクエスト本文 hash を含めるべきという方向性を示している。
- 今回差分では `docs/security-notes.md` と `docs/data-model.md` に request body hash を raw request body 代替の監査ログ属性として追加した。
- 一方、`docs/data-model.md` の `transfer_requests` には `idempotency_key` はあるが、操作種別、依頼者、送信元口座、request body hash、保存期間、同一キー同一内容 / 異内容の扱いはまだ具体化されていない。
- `docs/use-cases.md` は同じ冪等性キーの成功済み依頼がある場合に既存結果を返すとしているが、human notes と planner / implementer の作業仮定は MVP では重複再送を拒否する方向であり、docs 間に方針差が残っている。

### 影響

将来の振込 API 実装で、`request_body_hash` が audit_logs にだけ保存され、`transfer_requests` の一意性判定や衝突判定に使われない可能性がある。その場合、同じ idempotency key で金額・送信元口座・送信先口座を変えたリクエストが、実装によって「既存結果返却」「拒否」「別処理」のどれになるか曖昧になる。

攻撃または事故シナリオ:

1. 攻撃者または不具合クライアントが、同じ idempotency key で送信元口座や金額だけを変えた振込依頼を再送する。
2. 実装が `idempotency_key` だけで重複判定し、request body hash や actor / source account を比較しない。
3. 既存結果を誤って返す、異内容リクエストを拒否せず処理する、または監査ログ上は同一キーだが transfer request の意味が追跡できない状態になる。
4. 二重送金、架空送金、または利用者への誤った結果返却につながる。

### 推奨修正

次サイクルで、冪等性キーと request body hash の関係を監査ログとは別に `transfer_requests` 側の設計として明記する。

- MVP では重複再送を拒否するのか、既存結果を返すのかを `docs/use-cases.md` と human notes の方針に合わせて統一する。
- `transfer_requests` に保存する比較対象として、少なくとも `requested_by_user_id`, `source_account_id`, `operation_type`, `idempotency_key`, `request_body_hash` を候補化する。
- 同一 key + 同一 hash、同一 key + 異なる hash、異なる actor、異なる source account、処理中再送、成功後再送、失敗後再送の扱いを表にする。
- idempotency key 自体も user input として、最大長、文字種、保存期間、ログ出力時の扱いを定義する。

### 次サイクル planner への入力

「振込依頼状態遷移と冪等性キー衝突時の扱い」を security / banking 合同観点の docs-only scope として採択候補にする。対象は `docs/use-cases.md`, `docs/data-model.md`, `docs/security-notes.md`, `docs/test-strategy.md`。特に `docs/use-cases.md` の「既存結果を返す」と human notes の「MVP は拒否」の差分を解消し、request body hash を audit_logs だけでなく transfer request の安全な重複判定に使うかを決める。

## Finding 3: 管理者代行操作の actor / subject / 権限判断の分離が監査ログ項目にまだ表現されていない

- 重大度: Medium
- 観点: 認可、権限境界、監査証跡、否認防止
- 作業仮定のリスク: `admin` が顧客操作を代行可能という前提を置く場合、`actor_user_id` と対象 ID だけでは「誰のために」「どの権限で」実行したかを後から説明しにくい。

### 根拠

- human notes は `admin` をシステム管理者で代行可能、`operator` は MVP 対象外としている。
- `docs/security-notes.md` は MVP の監査ログ照会を `admin` のみに限定し、`operator` を MVP 対象外にした。
- `docs/data-model.md` の監査ログ属性は `actor_user_id`, `action_type`, `target_type`, `target_id`, `result`, `failure_reason`, `request_body_hash`, `ip_address`, `user_agent` だが、代行対象顧客、利用したロール、認可判断結果、代行理由、承認 ID のような項目は未定義。
- 既存の `docs/use-cases.md` では管理者が顧客登録、口座作成、監査ログ確認を行うが、入金・出金・振込を管理者が代行するかどうか、代行する場合の監査粒度はまだ整理されていない。

### 影響

管理者代行操作が今後 MVP に入る場合、監査ログ上は「admin user が account を操作した」ことしか残らず、顧客本人操作なのか、管理者による業務代行なのか、権限変更後の操作なのか、どの理由で許可されたのかを区別しにくくなる。内部不正、誤操作、権限誤設定の調査で説明責任が弱くなる。

攻撃または事故シナリオ:

1. 管理者アカウントが侵害され、顧客口座に対する入金・出金・振込・口座状態変更を実行する。
2. 監査ログには `actor_user_id=admin` と対象口座だけが残る。
3. 代行対象顧客、代行理由、許可されたロール、認可判断の根拠がないため、正当な業務代行と不正操作を後から分類しづらい。

### 推奨修正

次サイクル以降の RBAC / 認証設計で、管理者代行操作を MVP に含めるかを先に決める。含める場合は監査ログ項目に次の候補を追加するか検討する。

- `actor_user_id`: 実際に認証されたユーザー。
- `actor_role`: 操作時点のロール。
- `subject_customer_id` または `on_behalf_of_customer_id`: 代行対象。
- `authorization_result` / `authorization_reason`: 認可判断の分類。
- `operation_reason_code`: 管理者操作理由。自由記述にする場合は PII / secret 混入対策を付ける。

### 次サイクル planner への入力

「Cookie + CSRF と MVP RBAC」scope を作る際に、admin の代行範囲、顧客本人操作との監査ログ上の区別、operator を MVP に含めない場合の将来拡張点をセットで扱う。監査ログ schema / テスト観点には、admin 代行操作が actor と subject を分離して残ること、顧客本人操作と管理者操作を検索で区別できることを入れる。

## 問題なしと判断した点

- 成功した残高変更の監査ログを業務データ更新と同じ PostgreSQL transaction に含める方針は、証跡欠落を避ける観点では安全側であり、前回 Finding の主要部分を解消している。
- 業務拒否や DB transaction 途中失敗の失敗監査ログを、業務データ rollback と独立して残す方針は、失敗操作の追跡性を高めている。
- 監査ログ書き込み失敗時に残高変更や権限変更を成功扱いにしない fail closed 方針は、MVP 初期の学習用設計として妥当。
- 監査ログ照会を MVP では `admin` のみに限定し、`operator` を対象外にした点は、権限境界を小さく保っている。
- Go ソース、SQL、DB 接続、業務 API の差分はなく、今回差分から直接の SQL injection、認証 bypass、秘密情報漏えいのコード回帰は確認していない。
