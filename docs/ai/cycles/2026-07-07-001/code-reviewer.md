# code-reviewer: 2026-07-07-001

## レビュー範囲

- `.codex/agents/README.md` と `docs/ai/cycles/README.md` に従い、同 cycle の `planner.md` / `implementer.md` と現在の実装差分を入力としてレビューした。
- 作業開始時に `git status --short` を確認した。表示は空で、未コミット変更は確認しなかった。
- `AGENTS.md`、`README.md`、`docs/START_HERE.md`、主要な `docs/*.md`、`docs/decision-logs/*.md`、`docs/memo/*.md`、`docs/ai/output/human/*.md`、同 cycle 成果物を確認した。
- repo-local skill `.agents/skills/banking-code-review/SKILL.md` を参照し、Go / PostgreSQL / transaction 設計 / 保守性 / テスト容易性の観点で、実装差分を優先して確認した。
- レビュー対象の implementer 差分は commit `16c1ec8 Add safe failure fallback reason` の次の変更:
  - `internal/domain/failure_reason.go`
  - `internal/domain/failure_reason_test.go`
  - `README.md`
  - `docs/security-notes.md`
  - `docs/ai/cycles/2026-07-07-001/implementer.md`

## Finding

### 重大度: なし（Blocking / Major / Minor finding なし）

今回の implementer 差分について、code-reviewer 観点で修正必須の問題は確認しなかった。

## 根拠

- `FailureReasonInternalError = "internal_error"` が追加され、`FailureReason.Validate()` の allow-list にも含まれているため、監査ログ / safe structured log 用の固定分類として validation 境界を通過できる。
- 既存 `FailureReasonFromError(err error) (FailureReason, bool)` は、既知 domain sentinel error を固定分類へ写像し、未知 error と `nil` を `"", false` にする semantics を維持している。
- 新規 `SafeFailureReasonFromError(err error) (FailureReason, bool)` は、既知 domain sentinel error では既存 helper と同じ分類を返し、`nil` は failure として分類せず、未知 non-nil error は raw `err.Error()` ではなく `FailureReasonInternalError` へ寄せている。
- helper comment は、用途を audit `failure_reason` / safe structured logs に限定し、public API response body / HTTP status code の最終仕様ではないこと、raw request body や secrets を保存・露出しないことを明記している。
- unit test は supported reason validation、unsafe / unknown reason rejection、既存 `FailureReasonFromError` の未知 error / `nil` 非分類、`SafeFailureReasonFromError` の既知・wrapped・known+unknown joined・`nil`・未知 non-nil error の挙動を確認している。
- README と `docs/security-notes.md` は、`SafeFailureReasonFromError` を監査ログ / safe structured log 用 fallback と説明し、HTTP error response / status code、監査ログ永続化、DB schema は未実装であることを維持している。
- `go test ./...` と `git diff --check` は成功した。
- `rg -n "err\.Error\(\)|internal_error|FailureReasonInternalError|SafeFailureReasonFromError|password|token|secret|CSRF|session|request body" internal/domain README.md docs/security-notes.md` で関連箇所を確認し、unknown error の raw message を failure category として返す実装や、secret / raw request body を保存・返却する説明にはなっていないことを確認した。

## 影響

- 今回の差分は pure domain helper と docs / tests に閉じており、HTTP route、DB、repository、transaction manager、認証・認可、監査ログ永続化、業務 API の挙動には影響しない。
- unknown non-nil error を `internal_error` に寄せる fallback が追加されたことで、将来の audit writer / service / handler 実装時に raw DB error、panic detail、自由入力値、password、token、secret、CSRF token、session ID、raw request body などを `failure_reason` として保存・返却する事故を避ける土台が強化された。
- `FailureReasonFromError` の既存 semantics が維持されているため、「既知 domain error だけを分類できるか」を判定したい caller と、「監査 / safe log 用に unknown を安全分類へ落としたい caller」を分離できる。
- PostgreSQL transaction、行ロック、監査ログ永続化、public API error contract はまだ未実装であり、今回の helper だけで金融整合性や利用者向け error response が完成したわけではない。

## 推奨修正

- 今回の implementer 差分に対する修正必須事項はない。
- 次にこの helper を利用する service / handler / audit writer を実装する際は、`FailureReasonFromError` と `SafeFailureReasonFromError` の使い分けを境界ごとに明示することを推奨する。
  - domain / validation 境界: 既知 domain error の分類可否が必要なら `FailureReasonFromError`。
  - audit failure_reason / safe structured log 境界: unknown non-nil error を raw message にしないため `SafeFailureReasonFromError`。
  - public API response 境界: audit 用 `failure_reason` をそのまま利用者向け message として流用せず、別途 API error contract を設計する。
- 複数 domain sentinel error を含む `errors.Join` の優先順位は今回確定していないため、service 層で joined error を扱う必要が出た cycle で、優先順位表または単一 primary error 方針を planner に入力することを推奨する。

## 次サイクル planner への入力

- `SafeFailureReasonFromError` を利用する実装 scope に進む場合は、先に audit / service / handler 境界を分ける小さな設計または skeleton を accepted scope に含める。
- public API error response の `code` / `message` / HTTP status は、audit `failure_reason` とは別 contract として設計する。`internal_error` をそのまま利用者向け message にしない。
- 入金・出金・振込 service へ進む前に、少なくとも次の gate 順序を docs または service skeleton で固定することを推奨する:
  1. 認証 / CSRF
  2. 認可
  3. request validation
  4. 口座存在・状態確認
  5. 冪等性キー確認
  6. PostgreSQL transaction 開始
  7. 行ロック取得順序の固定
  8. 残高計算 / 取引履歴作成 / 監査ログ作成
  9. commit / rollback と失敗監査ログ
- PostgreSQL schema / migration に進む前に、`failure_reason` column の allow-list、audit log の相関 ID、raw error detail の保存禁止、失敗監査ログの独立 transaction 境界を docs と migration 方針に反映する。
- 冪等性キー scope は human note の方針どおり、操作種別、送信元口座、ログインユーザー、request body hash を含める前提で次 cycle 候補にできる。ただし DB 一意制約、重複時拒否、監査ログ分類と一体で扱う。

## 実行した確認

- `git status --short`: 作業開始時は表示なし。
- `git show --stat --oneline --decorate HEAD`: implementer 差分の対象ファイルを確認。
- `git show --name-status --format=short HEAD`: commit `16c1ec8` の変更ファイルを確認。
- `go test ./...`: 成功。
- `git diff --check`: 成功。
- `rg -n "err\.Error\(\)|internal_error|FailureReasonInternalError|SafeFailureReasonFromError|password|token|secret|CSRF|session|request body" internal/domain README.md docs/security-notes.md`: 関連する safe failure category / secret / raw request body 記述を確認。
