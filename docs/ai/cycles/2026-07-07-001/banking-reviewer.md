# banking-reviewer output: 2026-07-07-001

## レビュー対象

- 同 cycle の implementer 成果物 `docs/ai/cycles/2026-07-07-001/implementer.md` と、実装差分 commit `16c1ec8 Add safe failure fallback reason` を優先して確認した。
- 対象差分は `internal/domain/failure_reason.go`、`internal/domain/failure_reason_test.go`、`README.md`、`docs/security-notes.md`、`docs/ai/cycles/2026-07-07-001/implementer.md`。
- 本レビューでは、残高、元帳、取引履歴、冪等性、状態遷移、金融事故リスクの観点から、今回の safe failure category 追加が後続の監査ログ・失敗分類設計へ与える影響を中心に確認した。

## Finding

### 1. Blocking finding はなし

#### 根拠

- `FailureReasonFromError` は、既知 domain sentinel error を固定分類へ写像し、未知 error と `nil` を `"", false` にする従来 semantics を維持している。
- 追加された `SafeFailureReasonFromError` は、既知 domain error では `FailureReasonFromError` の分類を返し、未知 non-nil error では raw `err.Error()` ではなく `FailureReasonInternalError` / `internal_error` を返す。
- `nil` は failure として分類せず `"", false` のまま扱われており、成功・無エラーの操作を誤って失敗監査として記録する挙動にはなっていない。
- `FailureReason.Validate()` の allow-list に `internal_error` が追加され、同時に raw error message や secret-like value は reject されるテストが維持されている。
- README と security notes は、今回の helper を監査ログ / safe structured log 用 fallback と説明し、利用者向け HTTP error response / status code、監査ログ永続化、DB schema が未実装であることを維持している。
- 実行確認として `go test ./...`、`git diff --check`、および raw error / secret 関連語の `rg` 確認を行い、今回の差分範囲では failure category として raw unknown error を返す実装は確認しなかった。

#### 影響

- 今回の差分は残高更新、元帳 row、取引履歴、DB transaction、冪等性キー、振込状態遷移を直接変更していないため、既存の金額・残高 helper の金融整合性を壊す変更は確認しなかった。
- 将来の失敗監査ログ実装で、未知 DB error、driver error、panic 由来 error、自由入力を含む error message を `failure_reason` に保存する事故を防ぐ土台として有効である。
- `internal_error` を public API code として固定しない説明が入っているため、監査分類と利用者向けエラーを混同して口座・取引情報の過剰開示につながるリスクは今回の範囲では抑えられている。

#### 推奨修正

- 今回の実装差分に対する修正要求はない。
- 後続で audit writer / service / handler を実装するときは、`FailureReasonFromError` と `SafeFailureReasonFromError` の使い分けを境界ごとに明示すること。特に、監査ログ・safe structured log では unknown non-nil error を `SafeFailureReasonFromError` に通し、利用者向け response では別 contract を使うこと。

#### 次サイクル planner への入力

- 次に監査ログ永続化へ進む場合は、`failure_reason` に加えて `correlation_id` / `request_id` / `audit_log_id` のような調査用識別子を設計し、`internal_error` だけで運用調査を完結させようとしないこと。
- 入金・出金・振込 service に進む前に、失敗監査ログを残す境界を「入力不備、認証/認可拒否、口座状態拒否、残高不足、DB transaction 途中失敗」に分け、どの段階で `SafeFailureReasonFromError` を呼ぶかを整理すること。

### 2. 作業仮定として、複数 domain sentinel error を含む joined error の優先順位は未確定のまま残る

#### 根拠

- planner / implementer は、複数 domain sentinel error 間の優先順位表を今回 scope では確定しないと明記している。
- 現実装の `FailureReasonFromError` は `switch` の上から順に `errors.Is` を評価するため、複数の既知 domain error を含む `errors.Join` が渡された場合、実装順が分類結果になる。
- 今回追加されたテストは、accepted scope に沿って「既知 1 件 + unknown 1 件」の joined error が既知分類になることを確認しており、複数既知 error 間の外部仕様は固定していない。

#### 影響

- 現時点では業務 API、DB transaction、取引履歴、監査ログ永続化が未実装のため、直ちに残高不整合や二重送金を生む問題ではない。
- ただし将来 service 層で複数 validation error をまとめて返す設計にすると、例えば `insufficient_balance` と `account_not_active` が同時に含まれる error の監査分類が実装順に依存し、事故調査時に「本来見るべき拒否理由」が隠れる可能性がある。
- 失敗監査ログが後続の不正検知、顧客対応、障害調査の入力になる場合、分類優先順位の曖昧さは調査品質に影響する。

#### 推奨修正

- 今回 scope での修正は不要。
- 次に service validation / audit persistence を設計するとき、複数失敗理由を許容するか、単一代表理由に正規化するかを決めること。
- 単一代表理由にする場合は、金融事故防止の観点から、認証/認可拒否、口座状態拒否、残高不足、金額不正、内部エラーなどの優先順位を明文化し、テストで固定すること。
- 複数理由を残す場合でも、利用者向け response とは分離し、監査用に安全な分類配列または primary / secondary reason のような形を検討すること。

#### 次サイクル planner への入力

- 入出金・振込 service の accepted scope を作る前に、「複数 validation failure を join しない / join するが代表理由を明示する / 監査上は複数分類を保存する」のいずれかを設計候補として比較すること。
- 特に振込では、振込元口座停止、振込先口座停止、残高不足、冪等性重複、認可拒否が複合し得るため、監査ログの primary failure reason が事故調査で誤解を生まないようにすること。

## 事故シナリオ

- 事故シナリオ A: audit writer が unknown DB error を raw message のまま `failure_reason` に保存し、DB 接続文字列や SQL、個人情報を監査ログに混入させる。今回の `SafeFailureReasonFromError` を監査ログ境界で使えば、unknown non-nil error は `internal_error` に寄せられるため、この事故を低減できる。
- 事故シナリオ B: service 層が複数 domain error を `errors.Join` し、監査ログでは実装順の代表理由だけが残る。残高不足が主因なのか、口座停止が主因なのかが曖昧になり、顧客問い合わせや不正調査で誤った説明につながる可能性がある。これは今回 blocking ではないが、service / audit 実装前に planner で扱うべきである。

## 作業仮定のリスク

- `internal_error` は安全な固定分類であり、詳細調査には将来の相関 ID、安全な構造化ログ、運用ログ保管場所を別途使うという仮定に依存している。相関 ID なしに監査ログ永続化だけを実装すると、秘密情報は守れても障害調査が困難になる。
- `SafeFailureReasonFromError` は監査ログ / safe structured log 用であり、public API response の最終仕様ではないという仮定に依存している。後続実装でこの分類をそのまま利用者向け message として使うと、利用者体験やセキュリティ境界が曖昧になる。
- `nil` を失敗分類しない挙動は妥当だが、後続の audit writer は「成功時」「業務拒否時」「内部エラー時」を呼び出し側で明確に分ける必要がある。成功監査に `SafeFailureReasonFromError(nil)` の空分類を混ぜる設計は避けるべきである。

## 確認したコマンド

- `git status --short`
- `cat AGENTS.md`
- `cat .codex/agents/README.md`
- `cat docs/ai/cycles/README.md`
- `cat .agents/skills/banking-ledger-review/SKILL.md`
- `cat .codex/agents/banking-reviewer.toml`
- `sed -n '1,240p' README.md`
- `find docs -maxdepth 3 -type f | sort`
- `find docs/ai/cycles/2026-07-07-001 -maxdepth 1 -type f -print -exec sh -c 'echo "--- $1 ---"; sed -n "1,260p" "$1"' sh {} \;`
- `sed -n '1,220p' docs/START_HERE.md docs/design-principles.md docs/domain-model.md docs/data-model.md docs/security-notes.md docs/test-strategy.md docs/use-cases.md docs/mvp.md docs/memo/001.md docs/ai/output/README.md`
- `find docs/ai/output/human -maxdepth 1 -type f -print -exec sh -c 'echo "--- $1 ---"; sed -n "1,220p" "$1"' sh {} \;`
- `git log --oneline --decorate -5`
- `git show --stat --patch --find-renames --find-copies --unified=80 --no-ext-diff HEAD -- internal/domain/failure_reason.go internal/domain/failure_reason_test.go README.md docs/security-notes.md`
- `nl -ba internal/domain/failure_reason.go internal/domain/failure_reason_test.go`
- `go test ./...`
- `git diff --check`
- `rg -n "err\.Error\(\)|internal_error|FailureReasonInternalError|SafeFailureReasonFromError|password|token|secret|CSRF|session|request body" internal/domain README.md docs/security-notes.md`
