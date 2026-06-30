# planner: 2026-06-30-001

## repo現状

- 作業開始時の `git status --short` は出力なし。未コミット変更は確認しなかった。
- 実装済み:
  - Go 標準ライブラリのみの最小 REST API server。
  - `GET /healthz`、method 制限、固定 JSON response、server timeout、listen address 環境変数設定。
  - `internal/domain` の JPY 整数最小通貨単位向け `Amount` / `Balance` helper。
  - `Amount.Validate()` は 0 以下の取引金額を拒否し、`Balance.Validate()` は負の残高を拒否する。
  - `NewAmount` / `NewBalance` は `Validate()` を経由し、`AddBalance` / `SubtractBalance` は不正 `Amount` を拒否する。
  - `AddBalance` は overflow を検出し、`SubtractBalance` は残高不足を拒否して元残高を返す。
  - server / router / money helper の unit test。
- 設計済みだが未実装:
  - 顧客、ログインユーザー、口座、取引履歴、振込依頼、監査ログの概念と初期データモデル。
  - 残高変更、取引履歴、成功監査ログを同じ PostgreSQL transaction に含める方針。
  - 失敗監査ログを業務 transaction とは独立して残す方針。
  - Cookie 認証 + CSRF token、顧客本人または `admin` の認可、MVP では `operator` を対象外にする方針。
  - PostgreSQL 行ロックを使う人間方針。
  - 冪等性には操作種別、送信元口座、ログインユーザー、request body hash を含め、MVP の同一キー再送は既存結果返却ではなく拒否する人間方針。
- 未設計または未確定:
  - PostgreSQL schema / migration / repository / transaction manager の具体実装。
  - 入金・出金・振込 API の request / response schema と error mapping。
  - `AddBalance` / `SubtractBalance` が開始 `Balance` の validation 責務を持つか、service / repository 境界だけが持つかの code 上の統一。
  - 1 回あたり取引金額上限、口座残高上限、日次上限、管理者操作上限。
  - 監査ログ `failure_reason` の正規化一覧、IP / User-Agent の正規化、reverse proxy header の信頼境界。
  - 口座別取引順序、`balance_after` 連続性検証、reconciliation の実装方式。
- docs / 実装不一致:
  - README は現在の実装範囲と未実装機能をおおむね反映している。
  - 設計 docs は DB / 業務 API / 認証 / 監査ログ / 冪等性の将来方針まで含むが、実装は healthz と money domain helper に限定されている。
- レビュー未反映:
  - 同一 cycle の reviewer 群は `Amount.Validate()` / `Balance.Validate()` 追加に blocking 指摘なし。
  - 一方で、`AddBalance` / `SubtractBalance` は引数 `Balance` 自体を再検証していないため、将来の repository / mapper / service 接続前に責務を明確化する必要があると複数 reviewer が指摘している。
  - DB / 業務 API 接続前の高優先入力として、PostgreSQL `CHECK` 制約、行ロック順序、`balance_after` 連続性、金額上限、監査分類、domain error mapping が残っている。

## 入力レビュー

- human notes:
  - reversal は MVP に含めない。通常取引を先に明確化する。
  - PostgreSQL は学習目的で行ロックによる悲観ロックを採用する。
  - 冪等性には操作種別、送信元口座、ログインユーザー、request body hash を含める。
  - MVP の同一冪等性キー再送は既存結果返却ではなく拒否を優先する。
  - 監査ログは成功・失敗とも残し、失敗監査ログは業務 transaction とは独立して残す。
  - 認証は Cookie + CSRF token を前提にする。
  - `admin` は顧客操作の代行可能、`operator` は MVP に含めない。
  - 実装粒度は少し大きくしてもよいという人間メモがあるが、金融事故リスクを避けるため、未確定な DB / API へ進む前に domain 境界を小さく固める。
- code-reviewer 2026-06-30-001:
  - 直近差分に blocking 指摘なし。
  - `Amount.Validate()` / `Balance.Validate()` は境界再検証 helper として整合している。
  - `AddBalance` / `SubtractBalance` が開始 `Balance` を再検証しない点は現時点では blocking ではないが、将来の境界責務を明確にする必要がある。
  - DB schema / repository 着手前に、domain `Validate()` と PostgreSQL `CHECK` 制約の対応関係を設計 scope にすることを推奨している。
- security-reviewer 2026-06-30-001:
  - 直近差分に認証 bypass、認可 bypass、SQL injection、秘密情報漏えい、監査ログ汚染の直接的な問題はない。
  - `AddBalance` / `SubtractBalance` が開始 `Balance` を再検証しない点は、将来の repository / service 接続時の入力検証・domain invariant 防御リスクとして指摘されている。
  - 業務 API 前に金額上限、残高上限、domain error mapping、監査 `failure_reason` 分類を整理することを推奨している。
- banking-reviewer 2026-06-30-001:
  - 直近差分に元帳・残高観点の blocking 指摘なし。
  - constructor bypass、履歴欠落、並行出金、振込片側成功を後続実装時の事故シナリオとして残している。
  - すぐ業務 API へ進まず、DB constraint、行ロック、lock order、`balance_after` 連続性、reconciliation を docs / test 方針へ落とすことを推奨している。
- TODO / FIXME:
  - 実装コード上の明示的な TODO / FIXME は確認できなかった。未確定事項は docs と cycle artifact に記録されている。

## 改善候補

| 候補 | 内容 | repo上の根拠 | reviewer観点 | 実装時の注意 |
| --- | --- | --- | --- | --- |
| A | `AddBalance` / `SubtractBalance` の先頭で開始 `Balance` も `Validate()` し、負の開始残高を `ErrBalanceMustBeNonNegative` として拒否する | `Balance.Validate()` は追加済みだが、演算 helper は `Amount` だけを検証している。複数 reviewer が将来境界責務の曖昧さを指摘 | 元帳: 破損残高からの演算継続を防ぐ。Security: repository / mapper 接続時の invariant 防御。Code: helper 責務を test で固定 | 既存の正規経路挙動を壊さず、エラー時は元 balance を返す。HTTP / DB / API は追加しない |
| B | DB schema 前の docs として PostgreSQL `CHECK` 制約、行ロック、lock order、`balance_after` 連続性、reconciliation 方針を具体化する | `docs/data-model.md`, `docs/design-principles.md`, `docs/test-strategy.md` に方針はあるが詳細未確定。reviewer 群が高優先入力として残している | 元帳・コードレビューで高優先 | 今回 user 指示では planner.md 以外を書き換えないため、この turn では accepted scope として implementer に渡す場合のみ扱う |
| C | 金額上限・残高上限・上限超過時の監査分類を docs 化する | security-reviewer が API 前の次工程リスクとして指摘 | セキュリティ: 業務上不自然な金額を拒否。監査: failure_reason を検索可能にする | 上限値は人間レビューで変更されやすく、まず docs scope が適切 |
| D | 入金・出金・振込の security gate / domain error mapping skeleton を docs または code に落とす | `docs/security-notes.md`, `docs/use-cases.md` に認証認可と監査方針あり | セキュリティ: 認証済み、本人または admin、口座状態、金額 validation、監査境界 | 認証方式や API schema が未実装のため、code skeleton はまだ大きい |
| E | PostgreSQL schema / migration / repository の最小 skeleton に着手する | data model はあるが DB 接続未実装 | コード・元帳: DB 制約で最終防衛線を作れる | 行ロック、監査分類、transaction manager 方針が未確定で、今回 scope としては大きい |

## 採択

### 採択 A: 残高演算 helper で開始 `Balance` も再検証する

- 採択理由:
  - 直近 cycle の実装で `Balance.Validate()` が追加されたため、それを残高演算 helper の入口にも接続できる。
  - code-reviewer / security-reviewer が共通して「開始 `Balance` を helper が検証するのか、境界だけで検証するのか」を次工程リスクとして指摘している。
  - 学習用ミニバンキングとして、DB / repository から復元した破損残高を検出せず演算継続するより、domain helper で fail closed する方が金融事故リスクを下げられる。
  - `internal/domain` 内の小さな code-changing scope で完結し、未確定の HTTP / DB / 認証 / 監査 / 冪等性を先取りしない。
  - 人間メモの「実装粒度を少し大きくしてもよい」に対して、DB / API へ急がず、既存 helper の責務固定という安全な範囲で前進できる。
- accepted scope は下記「accepted scope」に記載する。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| E: PostgreSQL schema / migration / repository skeleton | 現時点では行ロック順序、transaction manager、監査ログ分類、DB constraint と domain helper の責務分担、migration 方針が未確定で scope が大きい | B / C / D の設計補強後、または人間が DB 実装優先を明示した場合 |
| 業務 API の追加 | 認証・認可・監査・冪等性・DB transaction が未実装であり、healthz 以外の API を先に増やすと金融事故リスクのある中途半端な endpoint になり得る | 認証認可・DB transaction・監査境界の最小設計が accepted scope 化された後 |
| reversal / 取消取引 | 人間 note で MVP 対象外と明示されている。通常取引、二重実行防止、監査を先に固める必要がある | MVP の通常入出金・振込・取引履歴・監査ログが安定した後 |

## 保留

| 候補 | 保留理由 | 次のアクション |
| --- | --- | --- |
| B: DB constraint / 行ロック / lock order / `balance_after` 連続性 / reconciliation docs | 高優先だが、今回は直近 reviewer が具体的に指摘した domain helper の曖昧さを先に潰す | 次 cycle で docs-only または DB migration 前設計として採択候補にする |
| C: 金額上限・残高上限・監査分類 docs | API 接続前に必要だが、上限値は人間レビューで調整されやすい | 次 cycle で暫定値を作業仮定として docs 化する候補にする |
| D: security gate / domain error mapping skeleton | 認証方式、handler 構成、監査 repository が未実装のため code skeleton は premature | API / use case 実装の直前に accepted scope 化する |

## accepted scope

### 目的

`AddBalance` / `SubtractBalance` が開始 `Balance` の不変条件も検証するようにし、DB / repository / mapper / test fixture などから constructor を経由しない不正な負の残高が渡された場合に、残高演算を継続せず fail closed できる domain 境界を作る。

### 対象ファイル/領域

- `internal/domain/money.go`
- `internal/domain/money_test.go`
- `README.md`
- `docs/ai/cycles/2026-06-30-001/implementer.md`

### 実装対象

1. `AddBalance(balance Balance, amount Amount) (Balance, error)` の先頭で `balance.Validate()` を呼ぶ。
   - 負の開始 `Balance` の場合、元の `balance` と `ErrBalanceMustBeNonNegative` を返す。
   - 開始 `Balance` が valid で `amount` が invalid の場合は、既存どおり元の `balance` と `ErrAmountMustBePositive` を返す。
   - valid な入力、overflow 検出、成功時の戻り値は既存挙動を維持する。
2. `SubtractBalance(balance Balance, amount Amount) (Balance, error)` の先頭で `balance.Validate()` を呼ぶ。
   - 負の開始 `Balance` の場合、元の `balance` と `ErrBalanceMustBeNonNegative` を返す。
   - 開始 `Balance` が valid で `amount` が invalid の場合は、既存どおり元の `balance` と `ErrAmountMustBePositive` を返す。
   - valid な入力、残高不足時に元 `balance` と `ErrInsufficientBalance` を返す挙動、成功時の戻り値は既存挙動を維持する。
3. validation 順序を test で固定する。
   - `balance.Validate()` を先に行う。
   - 開始 `Balance` が負、かつ `Amount{}` も invalid の場合、返す error は `ErrBalanceMustBeNonNegative` とする。
   - この順序により、破損した既存残高を「取引金額の問題」より優先して検出する。
4. unit test を追加・更新する。
   - `AddBalance` が負の開始 `Balance` を拒否し、元 `balance` を返す。
   - `SubtractBalance` が負の開始 `Balance` を拒否し、元 `balance` を返す。
   - 負の開始 `Balance` と invalid `Amount` が同時に渡った場合、`ErrBalanceMustBeNonNegative` が優先されることを `AddBalance` / `SubtractBalance` の少なくとも片方、可能なら両方で確認する。
   - 既存の正の加算、invalid amount、overflow、残高内減算、残高不足、`Amount.Validate()` / `Balance.Validate()` tests は成功し続ける。
5. README を必要最小限で更新する。
   - 現在の実装範囲に、残高加算・減算 helper が取引金額だけでなく開始残高も再検証することを短く追記する。
   - 業務 API、DB、認証、監査ログ、冪等性は引き続き未実装と明記する既存内容と矛盾させない。
6. `docs/ai/cycles/2026-06-30-001/implementer.md` を更新し、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

### 実装しないこと

- HTTP route、handler、request / response schema は追加しない。
- 顧客登録、口座作成、入金、出金、振込、残高照会、取引履歴照会、監査ログ照会の業務 API は実装しない。
- PostgreSQL 接続、migration、DB schema、SQL、repository、transaction manager は作らない。
- 認証、認可、Cookie session、CSRF token、ログアウトは実装しない。
- 監査ログ永続化、監査ログ分類、outbox、非同期補償は実装しない。
- 冪等性キー、`transfer_requests` 状態遷移、request body hash、同一キー衝突処理は実装しない。
- 取引金額上限、残高上限、日次上限は今回の code には入れない。
- PostgreSQL 行ロック、lock order、deadlock retry / fail 方針は実装しない。
- 多通貨、為替、小数、丸め、手数料、利息、取消 / reversal は実装しない。
- 外部依存ライブラリは追加しない。

### テスト方針

- `gofmt -w internal/domain/money.go internal/domain/money_test.go` を実行する。
- `go test ./...` を実行し、既存 server / router / domain tests と新規 validation tests がすべて成功することを確認する。
- `rg -n "float32|float64" internal/domain` を実行し、金額・残高 domain helper に浮動小数点を使っていないことを確認する。一致なしの exit code は期待結果として扱う。
- `git diff --name-only` と `git ls-files --others --exclude-standard` で、変更が accepted scope 内の `internal/domain/money.go`, `internal/domain/money_test.go`, `README.md`, `docs/ai/cycles/2026-06-30-001/implementer.md` に限定されていることを確認する。

### 作業仮定

- `Balance{}` は 0 円残高を表す valid な値として扱う。
- 負の `Balance` は、通常の外部 package 経路では作れないが、同一 package の mapper / test / 将来の repository helper では発生し得る破損値として扱う。
- `AddBalance` / `SubtractBalance` は「valid な `Balance` だけを前提にする」よりも、「自分の入口で `Balance` と `Amount` を両方検証する」責務を持つ。
- validation 順序は開始 `Balance` を先、`Amount` を後にする。破損した既存残高は、リクエスト金額不備よりも優先的に検出する。
- エラーは既存 sentinel error を再利用し、新しい error code は増やさない。
- 金額上限・残高上限など未確定の業務上限は今回の helper validation には含めない。

### レビューで重点確認してほしい観点

- 負の開始 `Balance` で `AddBalance` / `SubtractBalance` が演算を継続せず、元 `balance` と `ErrBalanceMustBeNonNegative` を返すか。
- invalid `Amount`、overflow、残高不足の既存 error と戻り値の互換性を壊していないか。
- validation 順序が test で明確になっており、破損開始残高が優先して検出されるか。
- domain helper が未確定の業務上限、HTTP / DB / 認証 / 監査仕様を先取りしていないか。
- README が現行実装範囲と未実装一覧を矛盾なく説明しているか。

## 実装しないこと

- この planner 作業では、`docs/ai/cycles/2026-06-30-001/planner.md` 以外を書き換えない。
- Go ソースコード、テストコード、README、通常 docs は変更しない。
- 他 agent と直接同期しない。既存 cycle 成果物は repo 上の入力 artifact としてのみ扱う。
- 金融仕様の最終決定は行わず、既存 docs と reviewer 入力に基づく小さな実装 scope だけを提示する。

## 作業仮定

- cycle id はユーザー指定がないため、指示どおり `2026-06-30-001` とする。
- 既に同 cycle の implementer / reviewer 成果物が存在するため、本 planner では現在 repo と既存 reviewer 出力を入力として、同じ `planner.md` を次の実装向けに更新する。
- planner の書き込みは `docs/ai/cycles/2026-06-30-001/planner.md` のみに限定する。
- 今回は planner role であり、実装や README 更新は implementer の作業とする。
- `Amount` と `Balance` の不変条件は現時点では「取引金額は 0 より大きい」「残高は 0 以上」「JPY の整数最小通貨単位」に限定する。
- 金額上限、残高上限、監査 `failure_reason`、行ロック、DB constraint、API error mapping は今回の accepted scope から外し、次 cycle 以降の候補として残す。
