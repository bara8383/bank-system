# security-reviewer: 2026-07-01-001

## レビュー対象

- cycle id: `2026-07-01-001`
- 優先対象: implementer 差分 `81f8a98 Add account status domain helper`
- 参照 artifact:
  - `docs/ai/cycles/2026-07-01-001/planner.md`
  - `docs/ai/cycles/2026-07-01-001/implementer.md`
- 参照した設計文書:
  - `README.md`
  - `docs/START_HERE.md`
  - `docs/domain-model.md`
  - `docs/use-cases.md`
  - `docs/security-notes.md`
  - `docs/design-principles.md`
  - `docs/data-model.md`
  - `docs/test-strategy.md`
  - `docs/mvp.md`

## 作業開始時確認

- `git status --short` で、作業開始時点の未コミット差分がないことを確認した。
- `README.md` で、現在の実装範囲が healthz、金額・残高 validation、口座ステータス validation の domain 土台に限定され、DB 接続、認証、業務 API、監査ログ、冪等性が未実装であることを確認した。
- `AGENTS.md`、`.codex/agents/README.md`、`docs/ai/cycles/README.md` を確認し、security-reviewer は source / README / planner / implementer を変更せず、本ファイルだけを書く前提で作業した。
- docs 配下の設計文書と cycle 成果物を確認し、今回の accepted scope が「口座ステータス domain helper の追加」に限定されていることを確認した。

## Finding 1: 新規 HTTP / DB / 認証境界を追加しておらず、差分による直接的な外部攻撃面の増加はない

- 重大度: Info

### 根拠

- implementer 差分は `internal/domain/account.go`、`internal/domain/account_test.go`、`README.md`、`docs/ai/cycles/2026-07-01-001/implementer.md` に限定されている。
- `internal/domain/account.go` は string-based な `AccountStatus`、`Validate()`、`EnsureAccountCanTransact()`、sentinel error のみを追加している。
- 既存 HTTP route は `/healthz` のみであり、今回差分では route / handler / request parser / response schema を追加していない。
- SQL、DB driver、repository、migration、secret、環境変数、ログ出力、認証 Cookie / token / session は追加されていない。

### 影響

- SQL injection、秘密情報漏えい、認証 bypass、CSRF、ログへの機微情報出力といった外部境界由来の新規リスクは今回差分では増えていない。
- `EnsureAccountCanTransact()` は pure domain helper であり、現時点では単独で資金移動を実行しないため、金融操作の実行面を広げていない。

### 推奨修正

- 今回 cycle 内での修正は不要。
- 後続で業務 API / repository / DB を追加する際は、domain helper 追加だけで安全とみなさず、HTTP 境界、service 境界、DB 制約、監査ログを別途実装・レビューする。

### 次サイクル planner への入力

- 次に業務 API skeleton へ進む場合は、認証・認可 gate、CSRF 方針、外部公開 route の scope、監査ログ方針を同 cycle の accepted scope に含めるか、先に設計文書化すること。

## Finding 2: 口座ステータス gate は認可ではなく業務状態 gate であり、後続実装で owner / role check と混同しないこと

- 重大度: Medium

### 根拠

- `EnsureAccountCanTransact(status AccountStatus) error` は `active` のみ残高変更系操作に進め、`suspended` / `closed` を `ErrAccountNotActive` で拒否する。
- helper の引数は account status のみであり、ログインユーザー、顧客 ID、口座 owner、admin / user role、セッション、権限スコープを受け取らない。
- README と implementer artifact でも、認証・認可は未実装として明示されている。

### 影響

- 後続 service / handler 実装で `EnsureAccountCanTransact()` を「操作権限確認」と誤解すると、active な他人の口座に対する入金・出金・振込を防げない可能性がある。
- 口座状態の安全性は向上するが、認可境界はまだ存在しないため、資金移動 API 実装前に owner / admin 代行範囲 / RBAC を決める必要がある。

### 推奨修正

- 次サイクル以降で use case / service を追加する前に、少なくとも次の gate を分離して設計する。
  - 認証: 誰がログインしているか。
  - 認可: その主体が対象口座を操作できるか。
  - 口座状態: 対象口座が残高変更可能な状態か。
  - 金額・残高 validation: 金額、残高、上限、残高不足を満たすか。
- 関数名や docs では、`EnsureAccountCanTransact()` は「account status check」であり「authorization check」ではないと明記し続ける。

### 次サイクル planner への入力

- 次 cycle では、業務 API 実装より先に `account owner / role authorization` の最小設計を候補に入れること。
- `admin` 代行を MVP で扱う場合、代行操作の audit actor / target / reason を先に設計すること。

## Finding 3: `ErrAccountNotActive` は suspended / closed を同一 error にまとめるため、監査分類・外部応答 mapping を別途設計しないと調査性が不足する

- 重大度: Medium

### 根拠

- `EnsureAccountCanTransact()` は `suspended` と `closed` のどちらにも `ErrAccountNotActive` を返す。
- planner / implementer の作業仮定では、利用者向けメッセージや監査分類で suspended / closed を分けるかは次 cycle 以降の error mapping / audit design で扱うとしている。
- 現行 repo には audit log persistence、`failure_reason` mapping、失敗監査ログの独立 transaction がまだない。

### 影響

- 後続でこの sentinel error だけを監査ログに記録すると、停止中口座への試行と解約済み口座への試行を後から区別しにくい。
- セキュリティ運用上、停止中口座への反復試行、解約済み口座へのアクセス試行、データ破損による unknown status を分けて検知・調査したい場面がある。
- 一方で、外部 API response では詳細を出しすぎると account existence / status enumeration の材料になり得るため、内部監査分類と外部メッセージは分離が必要。

### 推奨修正

- 次 cycle で domain error から API response / audit `failure_reason` への mapping 表を作る。
- 内部監査では少なくとも次を区別する。
  - `account_status_suspended`
  - `account_status_closed`
  - `account_status_invalid`
- 外部応答では、認可失敗や口座状態詳細の露出を抑える方針を別途決める。

### 次サイクル planner への入力

- `docs/security-notes.md` または cycle artifact に、domain error / external response / audit failure reason の対応表を accepted scope として採択することを推奨する。
- その際、unknown status は利用者起因ではなくデータ破損・mapper 不備寄りとして扱い、アラート対象にするか検討すること。

## Finding 4: 入力検証 helper は追加されたが、DB 制約・transaction・行ロックが未実装のため、将来の永続化境界で再検証が必要

- 重大度: Medium

### 根拠

- `AccountStatus.Validate()` は `active` / `suspended` / `closed` 以外を `ErrInvalidAccountStatus` として拒否する。
- 現行実装には PostgreSQL schema、CHECK 制約、repository、transaction manager、行ロックがない。
- README でも PostgreSQL 接続、DB schema、migration、transaction 処理は未実装として記載されている。

### 影響

- domain helper は有効な第一防衛線だが、将来 DB や repository を追加した時に DB 側の制約がないと、不正 status が永続化される可能性が残る。
- API / service / repository / DB のどこで再検証するかが曖昧なままだと、constructor を経由しない値や migration 不備による invalid status が残高変更処理に混入する恐れがある。

### 推奨修正

- DB schema 追加時に `accounts.status` の許容値を DB constraint で固定する。
- repository の読み書き境界で `AccountStatus.Validate()` を再実行し、DB 由来の破損 status を通常の停止口座とは別扱いで監査する。
- 入出金・振込 service では、transaction 内で account row を lock してから status と balance を検証する方針を設計する。

### 次サイクル planner への入力

- PostgreSQL schema / migration cycle では、status CHECK 制約、balance non-negative 制約、transaction rows、audit logs を同時に検討すること。
- 行ロック採用方針に沿い、2 口座振込時の lock 順序を docs に明記すること。

## Finding 5: ログ・秘密情報・SQL injection の観点では今回差分に直接の問題は確認できない

- 重大度: Info

### 根拠

- 新規コードは `errors.New(...)` による固定 error と switch validation のみで、ユーザー入力や secret をログ出力していない。
- SQL 文字列、DB 接続情報、外部ライブラリ、環境変数読み取りは追加されていない。
- test も domain unit test に限定され、機微情報や外部接続を扱っていない。

### 影響

- 今回差分単体では secret leakage、log injection、SQL injection の具体的な経路は見当たらない。
- ただし将来 handler で status 値や error をそのままログ・レスポンス化する場合、制御文字、長大入力、詳細な内部状態露出への対策が必要になる。

### 推奨修正

- 今回 cycle 内での修正は不要。
- 将来の handler / audit 実装では、外部入力値の最大長、正規化、監査ログの構造化、外部応答メッセージの固定化を設計する。

### 次サイクル planner への入力

- 監査ログ設計 cycle では、`ip_address`、`user_agent`、request body hash、failure reason、actor / target、correlation id の最大長と信頼境界を含めること。

## 総合評価

今回の差分は accepted scope に収まっており、認証・認可・HTTP・DB・ログ・secret・SQL の新規攻撃面は追加していない。口座ステータス validation は、停止中 / 解約済み口座の残高変更を拒否するための有効な domain 土台である。

ただし、この helper は認可ではなく業務状態 gate である。次 cycle 以降で業務 API や永続化に進む前に、owner / role authorization、監査 failure reason mapping、DB constraint、transaction 内 row lock、外部応答と内部監査分類の分離を planner 入力として扱うべきである。

## 実行した確認

- ✅ `git status --short`
- ✅ `sed -n '1,220p' .codex/agents/README.md`
- ✅ `sed -n '1,240p' docs/ai/cycles/README.md`
- ✅ `sed -n '1,200p' AGENTS.md`
- ✅ `sed -n '1,180p' README.md`
- ✅ `sed -n '1,180p' docs/START_HERE.md docs/domain-model.md docs/use-cases.md docs/security-notes.md docs/design-principles.md docs/data-model.md docs/test-strategy.md docs/mvp.md`
- ✅ `sed -n '1,240p' docs/ai/cycles/2026-07-01-001/planner.md docs/ai/cycles/2026-07-01-001/implementer.md`
- ✅ `git show --stat --oneline --name-only HEAD`
- ✅ `sed -n '1,180p' internal/domain/account.go internal/domain/account_test.go`
