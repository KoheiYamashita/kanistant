---
paths:
  - "cmd/**"
  - "pkg/**"
  - "Makefile"
  - "go.mod"
  - "go.sum"
---

# Go バックエンド

## アーキテクチャ

- **エントリポイント**: `cmd/clawdroid/main.go` — ワークスペースファイルを `//go:embed` で埋め込み、各サービスを初期化・起動
- **`pkg/agent/`**: エージェントループ（LLM 呼び出し → ツール実行の繰り返し）、コンテキスト構築、セッション管理
- **`pkg/tools/`**: ツールレジストリ＋各ツール実装。`registry.go` で登録、`toolloop.go` でループ実行。`types.go` に `Tool` インターフェース定義
- **`pkg/bus/`**: メッセージバス（Inbound/Outbound チャンネル間のメッセージルーティング）
- **`pkg/channels/`**: メッセージングアダプター（WebSocket, Telegram, Discord, Slack, WhatsApp, LINE）
- **`pkg/config/`**: JSON 設定 + `caarlos0/env` による環境変数バインディング。`migration.go` でバージョン間のスキーマ移行
- **`pkg/providers/`**: `any-llm-go` を使った LLM プロバイダー抽象化
- **`pkg/mcp/`**: Model Context Protocol クライアント管理

## 設計パターン

### Tool インターフェース階層（Strategy パターン）
- `Tool`: 基本インターフェース（`Name()`, `Description()`, `Parameters()`, `Execute()`）
- `ContextualTool`: チャンネル/チャット ID を受け取るツール用（`SetContext()`）
- `ActivatableTool`: 条件付きで有効/無効を切り替え（`IsActive()`）
- `AsyncTool`: 非同期実行＋コールバック（`SetCallback()`）
- Go の暗黙的インターフェース充足で、各ツールが必要な機能にオプトイン

### ToolRegistry（Registry パターン）
- `map[string]Tool` + `sync.RWMutex` でスレッドセーフに管理
- `ExecuteWithContext()` 内で型アサーションにより ContextualTool/AsyncTool を判別

### ToolResult（結果型パターン）
- `ForLLM`/`ForUser`/`Silent`/`IsError`/`Async`/`Media` フィールドで出力を制御
- コンストラクタ関数: `ErrorResult()`, `SilentResult()`, `AsyncResult()`, `UserResult()`
- メソッドチェーン: `ErrorResult("msg").WithError(err)`

### メッセージバス（Pub/Sub）
- バッファ付きチャンネル（容量100）で Inbound/Outbound をルーティング
- `select` + `ctx.Done()` でコンテキストキャンセル対応

### Channel アダプター
- `Channel` インターフェース（`Start`, `Stop`, `Send`, `IsRunning`, `IsAllowed`）
- `Manager` がコンフィグに基づき条件的にインスタンス化（Factory パターン）
- バックグラウンド goroutine で Outbound メッセージをディスパッチ

### 状態永続化（Atomic Write）
- temp ファイル書き込み → `os.Rename()` でアトミックに置換（クラッシュ耐性）
- Session/State 両方でこのパターンを使用

### 並行処理パターン
- `sync.RWMutex`: Registry, Config, Session, State の読み書き保護
- `atomic.Bool`: AgentLoop の実行状態をロックフリーで確認
- `sync.Map`: セッション単位の要約処理の追跡
- `context.Context` による cascading cancellation + シグナルハンドリングで graceful shutdown

### 設定の3層オーバーライド
- デフォルト値 → JSON ファイル → 環境変数（`caarlos0/env`）の順に上書き

### CGO_ENABLED=0
- 静的リンクで Android を含む全環境でのポータビリティを確保

### ワークスペース制約
- ファイル操作ツールは `workspace_path` 配下に制限（セキュリティ）

## ワークスペースファイル (`workspace/`)

`//go:embed` でバイナリに埋め込まれるテンプレート:
- `IDENTITY.md`, `SOUL.md`, `AGENT.md` — ボットの人格・動作指針
- `USER.md` — ユーザー設定テンプレート
- `HEARTBEAT.md` — ハートビートプロンプト

## 設定マイグレーション

`pkg/config/migration.go` でバージョンベースのスキーマ移行を管理。

- `Config.Version`（`config_version` JSON フィールド）が現在の `ConfigVersion` 定数より小さい場合、起動時に自動マイグレーション実行
- マイグレーション関数は `migrateV0ToV1`, `migrateV1ToV2`, ... のスライスで管理し、順番に適用
- 新しいフィールドを追加する場合: (1) `config.go` にフィールド追加 → (2) `ConfigVersion` をインクリメント → (3) `migrateVxToVy` 関数でデフォルト値をセット
- 現在のバージョン: `ConfigVersion = 2`

## テスト

Go の標準テスト (`*_test.go`)。`testify` を使用。テーブル駆動テストパターンが多い。
