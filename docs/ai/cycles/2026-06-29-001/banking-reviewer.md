# banking-reviewer: 2026-06-29-001

## レビュー対象

- `docs/ai/cycles/2026-06-29-001/planner.md`
- `docs/ai/cycles/2026-06-29-001/implementer.md`
- 実装差分: `docs/design-principles.md`, `docs/data-model.md`, `docs/test-strategy.md`, `docs/ai/cycles/2026-06-29-001/implementer.md`

実装差分は docs-only で、Go ソースコード、DB schema、migration、業務 API は変更されていない。元帳・残高方向・成功時の DB transaction 境界を具体化する accepted scope には概ね沿っている。

## Finding 1: 失敗時監査ログの扱いが既存 docs と衝突して見える

### 根拠

- `docs/design-principles.md` の追記では、失敗時監査ログを未確定事項としている。
- 一方で、同じ `docs/design-principles.md` では重要操作の監査ログに結果と失敗理由を残す原則が既にある。
- `docs/test-strategy.md` 既存部分では、出金失敗時に失敗ログが残ることを結合テスト観点にしている。
- `docs/security-notes.md` でも重要操作の成功と失敗を監査ログに残す方針がある。

### 影響

実装者が「失敗時監査ログは未確定なので不要」と読んだ場合、残高不足、権限拒否、二重送信、途中失敗などの証跡が残らない設計になり得る。金融事故調査では、成功取引だけでなく失敗した操作の連続性も重要であり、不正試行や残高不足の反復を追跡できないリスクがある。

### 推奨修正

次 cycle で「失敗時監査ログを残す必要性」と「どの transaction 境界で残すか」を分けて記述する。たとえば、重要操作の失敗証跡は残す方針を維持しつつ、業務 DB transaction の rollback と独立して記録する方式、監査ログ書き込み失敗時の fail closed / 補償方式は人間確認事項として残す。

### 次サイクル planner への入力

監査ログ専用 scope を採択候補にする。対象は `docs/design-principles.md`, `docs/security-notes.md`, `docs/test-strategy.md`, 必要なら `docs/data-model.md`。成功・失敗の記録要否、rollback との関係、監査ログ書き込み失敗時の扱いを分離して整理する。

## Finding 2: 振込依頼の状態遷移境界が成功時更新だけに寄っている

### 根拠

- `docs/design-principles.md` の追記では、振込成功時に同一 DB transaction へ含める更新として、振込元残高減少、振込先残高増加、2件の取引履歴、振込依頼の成功状態更新を挙げている。
- `docs/data-model.md` の `transfer_requests.status` は `accepted`, `processing`, `succeeded`, `failed`, `cancelled` を候補にしている。
- しかし、`accepted` から `processing`、`succeeded` / `failed` への許可遷移、処理開始前・処理中クラッシュ・リトライ時の扱いは未整理のまま残っている。

### 影響

将来の実装で、振込依頼だけが `accepted` のまま残る、`processing` の再実行可否が曖昧になる、または成功済み依頼が別経路で再処理される余地が残る。これは二重送金、未処理依頼の放置、利用者への結果返却不整合につながる。

### 推奨修正

振込の状態遷移表を docs に追加し、各状態で許可される操作、冪等な再送時の返却結果、失敗時に `failed` を記録する条件、再試行可能な失敗と不可逆な失敗の扱いを分ける。成功時の資金移動と成功状態更新は同一 DB transaction に含める現方針を維持する。

### 次サイクル planner への入力

`transfer_requests` の状態遷移 docs 化を採択候補にする。冪等性キー詳細と密接に関係するため、同一 scope で「同一キー同一内容」「同一キー異内容」「処理中再送」「成功後再送」の扱いを設計案として明示する。

## Finding 3: `balance_after` の連続性検証ルールがまだ不足している

### 根拠

- `docs/data-model.md` の追記で、`balance_after` は対象口座へ取引を適用した直後の口座残高で、0以上かつ更新後残高と一致すると定義された。
- 既存の `docs/data-model.md` では、将来的に取引履歴の合計と現在残高が一致するかを定期検証するとしている。
- ただし、同一口座内の取引順序、前回残高から今回 `amount` と `transaction_type` を適用した結果が `balance_after` になること、欠番・重複・順序逆転をどう検出するかは未定義である。

### 影響

現在残高と最新 `balance_after` だけを合わせる実装になると、途中の取引履歴欠落、二重記録、順序誤りを見逃す可能性がある。監査時に「どの取引で残高がどう変わったか」を再計算できず、元帳としての説明力が弱くなる。

### 推奨修正

次 cycle で、同一口座内の取引順序と残高連続性の検証ルールを docs に追加する。最低限、口座ごとに取引を確定順で並べ、前回 `balance_after` に transaction type の増減方向と `amount` を適用した値が今回 `balance_after` と一致することをテスト観点に含める。

### 次サイクル planner への入力

`docs/data-model.md` と `docs/test-strategy.md` に、口座別取引順序、残高連続性、振込の debit / credit 関連付け、定期リコンシリエーションの検証方針を追加する scope を検討する。DB schema 確定前の docs-only scope として扱える。

## 人間確認事項

1. 失敗時監査ログは業務 transaction が rollback しても必ず残す方針にするか。
2. 監査ログ書き込み失敗時に、残高変更や振込成功を fail closed で止めるか、別経路で補償するか。
3. `transfer_requests.processing` のまま中断した依頼を、自動再試行、手動調査、失敗確定のどれで扱うか。
4. `reversal` を MVP 初期に含めるか、初期は取消未実装かつ既存取引履歴不可変の方針だけにするか。

## 総評

今回の docs-only 差分は、残高増減方向、振込の二面性、`balance_after`、成功時の同一 DB transaction 境界を明確にしており、元帳実装前の前提整理として有効である。次 cycle では、成功時整合性の周辺に残っている失敗証跡、状態遷移、残高連続性を先に詰めると、DB schema と業務 API 実装時の金融事故リスクを下げられる。
