# security-reviewer: 2026-07-05-001

## Finding

### 1. Blocking finding なし

- 重大度: なし
- 今回の実装差分は `internal/domain` の pure domain helper と README / cycle 成果物更新に限定されており、直接的な外部攻撃面の増加は確認しなかった。
- 新規の HTTP route、request / response schema、SQL、DB 接続、repository、認証、認可、Cookie / CSRF、ログ出力、秘密情報処理は追加されていない。
- `TransactionType.Validate()` は MVP 対象の `deposit` / `withdrawal` / `transfer_debit` / `transfer_credit` のみを許可し、空文字、未知値、`reversal` を `ErrInvalidTransactionType` で拒否している。
- `ApplyTransaction` は `balance`、`amount`、`transactionType` の順に再検証し、失敗時は元の `balance` を返すため、今回の helper 単体では不正入力で残高計算結果を進める挙動は確認しなかった。

## 根拠

- 差分確認:
  - `git diff --stat 6ab8f33..HEAD` で、今回の実装差分が `README.md`、`docs/ai/cycles/2026-07-05-001/implementer.md`、`internal/domain/transaction.go`、`internal/domain/transaction_test.go` に限定されていることを確認した。
  - `git diff 6ab8f33..HEAD -- README.md internal/domain/transaction.go internal/domain/transaction_test.go docs/ai/cycles/2026-07-05-001/implementer.md` で、実装内容が accepted scope の取引種別 helper とテスト、README 更新であることを確認した。
- セキュリティ境界:
  - 新規 helper は `errors` のみを import しており、HTTP、DB、ファイル、環境変数、ログ、外部通信へ触れていない。
  - `ApplyTransaction` は transaction row、監査ログ、永続化、認証・認可状態を扱わず、取引種別に応じた残高計算候補を返すだけである。
- 入力検証:
  - `TransactionType.Validate()` は許可リスト方式で実装されている。
  - `ApplyTransaction` は constructor を経由しない `Balance` / `Amount` を再検証している。
  - `deposit` / `transfer_credit` は `AddBalance`、`withdrawal` / `transfer_debit` は `SubtractBalance` に委譲しており、既存の overflow / 残高不足 / invalid amount / invalid balance の sentinel error を維持している。
- テスト:
  - `go test ./...` が成功した。
  - 追加テストで valid / invalid transaction type、`reversal` 拒否、残高増加、残高減少、残高不足、invalid balance、invalid amount、invalid transaction type、overflow、validation 順序が確認されている。

## 影響

- 今回差分のままでは、外部入力から直接呼ばれる API が増えていないため、認証 bypass、認可 bypass、SQL injection、CSRF、secret leakage、ログへの機密情報出力といった直接リスクは増えていない。
- `reversal` を valid type に含めていないため、取消・組戻し・訂正の未確定仕様を利用者入力で有効化してしまうリスクは今回時点では抑えられている。
- 一方で、`ApplyTransaction` は「残高計算 helper」であり、口座所有者確認、ロール確認、口座ステータス確認、冪等性、監査ログ、DB transaction、取引履歴永続化を保証しない。後続 cycle で業務 API / service からこの helper を使う場合、これらの security gate を別レイヤーで必ず組み合わせないと、未認可の残高変更、二重実行、監査証跡欠落、片側だけ成功する資金移動につながる。
- `ErrInvalidTransactionType` は外部応答としては安全な汎用 error だが、API response と audit `failure_reason` の mapping が未確定のままだと、後続実装で raw transaction type や request body を監査ログへ残す、または利用者へ内部分類を過剰に返す実装が混入する余地が残る。

## 推奨修正

- 今回差分への blocking 修正は不要。
- 後続で入金・出金・振込の service / handler を追加する前に、`ApplyTransaction` を直接公開境界から呼ばない設計を明示する。
  - handler 境界: 認証済み actor、CSRF token、request size / content type、入力 schema validation を確認する。
  - authorization 境界: 顧客本人の口座か、許可された admin / operator action かを確認する。`EnsureAccountCanTransact` は認可ではなく口座状態 gate として扱う。
  - domain / service 境界: `EnsureAccountCanTransact`、冪等性キー、日次上限や金額上限の暫定方針、`ApplyTransaction`、取引履歴作成、監査ログ作成を明確な順序で組み合わせる。
  - DB 境界: 残高更新、取引履歴、成功監査ログを同一 PostgreSQL transaction に含め、失敗監査ログはロールバック後の独立 transaction に残す。
- API response / audit `failure_reason` mapping を次 cycle で設計する。
  - 外部応答は `invalid_request`、`insufficient_balance`、`account_not_available` など安全な分類に寄せる。
  - audit `failure_reason` は `invalid_transaction_type`、`invalid_amount`、`invalid_balance_state`、`insufficient_balance`、`account_not_active` など運用調査に必要な安全な分類にする。
  - raw request body、password、token、secret、CSRF token、session ID、過剰な個人情報、未加工の自由入力値は監査ログに保存しない。
- `reversal` は今回同様に拒否を維持し、取消・組戻し・訂正を扱う cycle では「対象取引との関連」「二重取消防止」「取消可否」「監査ログ」「残高方向」「権限」を別 scope として設計する。

## 次サイクル planner への入力

1. 優先候補: domain error / API response / audit `failure_reason` mapping を docs 化する。
   - 対象 error: `ErrAmountMustBePositive`、`ErrBalanceMustBeNonNegative`、`ErrInsufficientBalance`、`ErrBalanceOverflow`、`ErrInvalidAccountStatus`、`ErrAccountNotActive`、`ErrInvalidTransactionType`。
   - 目的: 利用者向け応答、運用調査向け監査分類、ログ出力禁止情報を分離し、後続 API 実装時の情報露出と監査欠落を防ぐ。
2. 優先候補: 入金・出金・振込 service 前の security gate 順序を docs 化する。
   - 認証、CSRF、owner / role authorization、口座ステータス、入力 validation、冪等性、残高計算、DB transaction、成功 / 失敗監査ログの順序を明示する。
   - `ApplyTransaction` はこのうち残高計算だけを担う helper として位置づける。
3. 優先候補: 冪等性キー設計を具体化する。
   - 操作種別、送信元口座、ログインユーザー、request body hash を含める案を検討する。
   - MVP で重複キーを既存結果返却にするか拒否にするか、監査ログへどう残すかを決める。
4. 保留候補: `reversal` / 取消 / 組戻し / 訂正。
   - 現時点では valid transaction type に含めない方針を維持する。
   - 扱う場合は、通常取引 helper への小変更として混ぜず、別 cycle で二重取消防止と監査要件を含めて設計する。
