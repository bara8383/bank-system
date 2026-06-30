# planner: 2026-06-30-001

## repo現状

- 作業開始時の `git status --short` は出力なし。未コミット変更は確認しなかった。
- 実装済み:
  - Go 標準ライブラリのみの最小 REST API サーバー。
  - `GET /healthz` と method 制限、固定 JSON response、server timeout / listen address 設定。
  - `internal/domain` の JPY 整数最小通貨単位向け `Amount` / `Balance` helper、正の取引金額、0 以上の残高、加算、残高不足を拒否する減算、overflow 検出。
  - server / router / money helper の unit test。
- 設計済みだが未実装:
  - 顧客、ユーザー、口座、取引履歴、振込依頼、監査ログの概念と初期データモデル。
  - 残高変更と取引履歴、成功監査ログを同じ PostgreSQL transaction に含める方針。
  - 失敗監査ログを業務 transaction とは独立して残す方針。
  - Cookie 認証 + CSRF token、`admin` と顧客本人の認可、MVP では `operator` 対象外とする方針。
  - PostgreSQL 行ロックを使う人間方針。
  - 冪等性キーには操作種別、送信元口座、ログインユーザー、request body hash を含める人間方針。
- 未設計または未確定:
  - PostgreSQL schema / migration / repository / transaction manager の具体実装。
  - 入金・出金・振込 API の request / response schema と error mapping。
  - 1 回あたり取引金額上限、口座残高上限、日次上限、管理者操作上限。
  - 監査ログ failure_reason の正規化一覧、IP / User-Agent の正規化、reverse proxy header の信頼境界。
  - 口座別取引順序、`balance_after` 連続性検証、reconciliation の実装方式。
- docs / 実装不一致:
  - README は現在の実装範囲と未実装機能をおおむね反映している。
  - 設計 docs は DB / 業務 API の将来方針まで含むが、実装は healthz と money domain helper に限定されている。
- レビュー未反映:
  - 2026-06-29-002 の reviewer 群は blocking / must-fix はないと判断したが、DB / 業務 API 接続前の小さな補強として domain 型の境界利用ルール、DB constraint 対応、金額上限、行ロック、監査分類を次 cycle 入力に残している。

## 入力レビュー

- human note:
  - reversal は MVP に含めない。
  - PostgreSQL は学習目的で行ロックによる悲観ロックを採用する。
  - 冪等性には操作種別、送信元口座、ログインユーザー、request body hash を含める。
  - MVP の同一冪等性キー再送は既存結果返却ではなく拒否を優先する。
  - 監査ログは独立して残す。
  - 認証は Cookie + CSRF token を前提にする。
  - `admin` は代行可能、`operator` は MVP に含めない。
- code-reviewer 2026-06-29-002:
  - 直近差分に blocking 指摘なし。
  - DB / repository 前に PostgreSQL `CHECK` 制約と domain helper の対応を検討すること。
  - 残高更新 service では DB transaction 内の残高更新・取引履歴・成功監査ログの commit / rollback test を含めること。
  - 行ロック、lock order、deadlock 方針、冪等性キー衝突時の扱いは高優先 docs scope として残すこと。
- banking-reviewer 2026-06-29-002:
  - 元帳・残高の blocking 指摘なし。
  - `Amount{}` のゼロ値は作れるため、constructor bypass を後続実装で誤用しない境界ルールが必要。
  - domain helper だけでは元帳・取引履歴の完全性は保証されないため、DB transaction / `balance_after` / 行ロック / 失敗監査ログ方針を接続する必要がある。
- security-reviewer 2026-06-29-002:
  - 認証・認可・SQL injection・秘密情報漏えいの新規リスクは確認されていない。
  - 業務 API 前に security gate、取引上限・残高上限、domain error mapping、監査 failure_reason 分類を検討すること。

## 改善候補

| 候補 | 内容 | repo上の根拠 | reviewer観点 | 実装時の注意 |
| --- | --- | --- | --- | --- |
| A | `Amount` / `Balance` に境界利用用の `Valid()` または `Validate()` helper を追加し、ゼロ値や不正値を公開関数・repository 境界で再検証できるようにする | `internal/domain/money.go` は `NewAmount` で validation するが `Amount{}` は作れる。banking-reviewer が constructor bypass リスクを指摘 | 元帳: 0 円取引混入防止。コード品質: repository / service 境界の規約を test しやすくする | domain package に限定し、HTTP / DB / API は追加しない。既存 `AddBalance` / `SubtractBalance` の挙動を壊さない |
| B | DB schema 前の docs として PostgreSQL `CHECK` 制約、行ロック、lock order、`balance_after` 連続性を具体化する | `docs/data-model.md`, `docs/design-principles.md`, `docs/test-strategy.md` に方針はあるが詳細未確定 | 元帳・コードレビューで高優先 | 今回 user 指示では code-changing scope を 1 つ以上含める必要があるため、単独採択ではなく保留または次 scope |
| C | 金額上限・残高上限・上限超過時の監査分類を docs 化する | security-reviewer が API 前の次工程リスクとして指摘 | セキュリティ: 業務上不自然な金額を拒否 | 上限値は人間レビューで変更されやすく、現時点では実装より docs が先 |
| D | 入金・出金・振込の security gate / domain error mapping skeleton を docs または code に落とす | `docs/security-notes.md`, `docs/use-cases.md` に認証認可と監査方針あり | セキュリティ: 認証済み、本人または admin、口座状態、金額 validation、監査境界 | 認証方式や API schema が未実装のため、code skeleton はまだ大きい |
| E | PostgreSQL schema / migration / repository の最小 skeleton に着手する | data model はあるが DB 接続未実装 | コード・元帳: DB 制約で最終防衛線を作れる | 行ロック、監査分類、transaction manager 方針が未確定のため、今回 scope としては大きい |

## 採択

### 採択 A: domain 型の境界再検証 helper を追加する

- 採択理由:
  - 直近の banking-reviewer が指摘した `Amount{}` constructor bypass リスクに対して、DB / 業務 API に進む前に小さな code-changing 改善で対応できる。
  - 既存 helper は残高変更時に amount を再検証しているが、将来の repository / transaction writer が `Amount.Int64()` を直接保存する前の検証方法を明示できていない。
  - `Valid()` / `Validate()` helper と unit test は domain package 内で完結し、未確定の HTTP / DB / 認証 / 監査設計を先取りしない。
  - 既存 docs の「金額は正の整数」「残高は0以上」と一致する。
- accepted scope は下記「accepted scope」に記載する。

## 却下

| 候補 | 却下理由 | 再検討条件 |
| --- | --- | --- |
| E: PostgreSQL schema / migration / repository skeleton | 現時点では行ロック順序、transaction manager、監査ログ分類、DB constraint と domain helper の責務分担が未確定で、scope が大きい。今回の小さな改善としては risk が高い | B / C / D の設計補強後、または人間が DB 実装優先を明示した場合 |
| 業務 API の追加 | 認証・認可・監査・冪等性・DB transaction が未実装であり、healthz 以外の API を先に増やすと金融事故リスクのある中途半端な endpoint になり得る | 認証認可・DB transaction・監査境界の最小設計が accepted scope 化された後 |

## 保留

| 候補 | 保留理由 | 次のアクション |
| --- | --- | --- |
| B: DB constraint / 行ロック / lock order / `balance_after` 連続性 docs | 高優先だが今回は code-changing scope を小さく進める。DB 実装前に必ず必要 | 次 cycle で docs-only または DB migration 前設計として採択候補にする |
| C: 金額上限・残高上限・監査分類 docs | API 接続前に必要だが、上限値は人間レビューで調整されやすい | 次 cycle で暫定値を作業仮定として docs 化する候補にする |
| D: security gate / domain error mapping skeleton | 認証方式、handler 構成、監査 repository が未実装のため code skeleton は premature | API / use case 実装の直前に accepted scope 化する |

## accepted scope

### 目的

`Amount` / `Balance` のゼロ値や不正値を、constructor 以外の境界でも明示的に再検証できる domain API を追加し、将来の service / repository / DB insert 境界で constructor bypass を見落としにくくする。

### 対象ファイル/領域

- `internal/domain/money.go`
- `internal/domain/money_test.go`
- `README.md`
- `docs/ai/cycles/2026-06-30-001/implementer.md`

### 実装対象

1. `Amount` に validation method を追加する。
   - 推奨名: `Validate() error`。
   - `Amount{}` や 0 以下の値を `ErrAmountMustBePositive` として拒否する。
   - 正の amount は `nil` を返す。
2. `Balance` に validation method を追加する。
   - 推奨名: `Validate() error`。
   - 負の balance を `ErrBalanceMustBeNonNegative` として拒否する。
   - 0 以上の balance は `nil` を返す。
3. 既存 constructor / helper が同じ validation rule を共有するように、可能なら重複条件を `Validate()` へ寄せる。
   - `NewAmount` は不正値で `ErrAmountMustBePositive` を返す既存挙動を維持する。
   - `NewBalance` は不正値で `ErrBalanceMustBeNonNegative` を返す既存挙動を維持する。
   - `AddBalance` / `SubtractBalance` は不正 amount を拒否し、エラー時に元 balance を返す既存挙動を維持する。
4. unit test を追加・更新する。
   - `Amount.Validate()` が `Amount{}` を拒否する。
   - `Amount.Validate()` が `NewAmount(1)` で作った正の amount を受け付ける。
   - `Balance.Validate()` が 0 と正の balance を受け付ける。
   - domain package 内の test で負の `Balance` 値を直接作り、`Balance.Validate()` が拒否することを確認する。
   - 既存の加算・減算・overflow・残高不足 test は成功し続ける。
5. README を更新する。
   - 現在の実装範囲に、金額・残高 helper が constructor だけでなく境界再検証用 method を提供することを短く追記する。
   - 業務 API、DB、認証、監査ログ、冪等性は引き続き未実装と明記する既存内容と矛盾させない。
6. `docs/ai/cycles/2026-06-30-001/implementer.md` を作成し、参照した accepted scope、変更内容、scope 適合性、実装しなかったこと、テスト結果、作業仮定を記録する。

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

- `gofmt` を変更した Go ファイルに適用する。
- `go test ./...` を実行し、既存 server / router / domain tests と新規 validation tests がすべて成功することを確認する。
- `rg -n "float32|float64" internal/domain` を実行し、金額・残高 domain helper に浮動小数点を使っていないことを確認する。
- `git diff --name-only` で、変更が accepted scope 内の `internal/domain/money.go`, `internal/domain/money_test.go`, `README.md`, `docs/ai/cycles/2026-06-30-001/implementer.md` に限定されていることを確認する。

### 作業仮定

- `Validate() error` は「値 object が現在の不変条件を満たすか」を確認する method とし、金額上限・残高上限など未確定の業務上限は含めない。
- `Amount{}` は invalid と扱う。
- `Balance{}` は 0 円残高を表す valid な値として扱う。
- `Validate()` 追加後も `NewAmount` / `NewBalance` を標準生成経路とし、`Validate()` は境界再検証用の補助とする。
- エラーは既存 sentinel error を再利用し、新しい error code は増やさない。

### レビューで重点確認してほしい観点

- `Amount{}` を `Validate()` で拒否できるか。
- `Balance{}` を 0 円残高として valid に扱えているか。
- 既存 constructor / `AddBalance` / `SubtractBalance` の error と戻り値の互換性を壊していないか。
- `Validate()` が未確定の業務上限や DB / HTTP / 認証仕様を先取りしていないか。
- README が現行実装範囲と未実装一覧を矛盾なく説明しているか。

## 実装しないこと

- この planner 作業では、`docs/ai/cycles/2026-06-30-001/planner.md` 以外を書き換えない。
- Go ソースコード、テストコード、README、通常 docs は変更しない。
- 他 agent と直接同期しない。既存 cycle 成果物は repo 上の入力 artifact としてのみ扱う。
- 金融仕様の最終決定は行わず、既存 docs と reviewer 入力に基づく小さな実装 scope だけを提示する。

## 作業仮定

- cycle id はユーザー指定どおり `2026-06-30-001` とする。
- planner の書き込みは `docs/ai/cycles/2026-06-30-001/planner.md` のみに限定する。
- 今回は planner role であり、実装や README 更新は implementer の作業とする。
- `Amount` と `Balance` の不変条件は現時点では「取引金額は 0 より大きい」「残高は 0 以上」「JPY の整数最小通貨単位」に限定する。
- 金額上限、残高上限、監査 failure_reason、行ロック、DB constraint は今回の accepted scope から外し、次 cycle 以降の候補として残す。
