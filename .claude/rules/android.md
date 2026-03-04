---
paths:
  - "android/**/*.kt"
  - "android/**/*.kts"
  - "android/**/*.xml"
  - "android/**/*.gradle*"
---

# Android フロントエンド

## モジュール構成

Gradle マルチモジュール構成:
- **`app/`**: Activity, Service, AccessibilityService, セットアップウィザード
- **`backend/`**: Go バックエンドローダー（Embedded 版 / Termux 版の切り替え）
- **`core/`**: 共有コア（data/domain/ui レイヤー）
- **`feature/`**: 機能モジュール（chat）

## ビルドフレーバー

- `embedded`: Go バックエンドを `libclawdroid.so` として jniLibs に含む
- `termux`: バックエンドなし（Termux 上で別途 Go バイナリを実行）

## 技術スタック

- Kotlin + Jetpack Compose（UI）
- Room（ローカル DB、チャット履歴）
- Koin（DI）— `app/src/main/java/io/clawdroid/di/AppModule.kt` で定義
- kotlinx.serialization
- KSP（Kotlin Symbol Processing）
- Coroutines + StateFlow（非同期・リアクティブ）

## 設計パターン

### Clean Architecture レイヤー分離
- **Domain** (`core/domain`): Android 依存なし。インターフェース・UseCase・ドメインモデル
- **Data** (`core/data`): Repository 実装、Room DAO、WebSocket クライアント、Mapper
- **Presentation** (`feature/`, `app/`): Compose UI、ViewModel

### MVI/MVVM ハイブリッド（ViewModel + StateFlow + Sealed Event）
- `ChatUiState` データクラスが UI 状態の単一ソース
- `sealed interface ChatEvent` でユーザー操作を型安全に表現
- ViewModel が `MutableStateFlow` を保持し、`onEvent()` でイベント処理

### Repository パターン
- `ChatRepository` インターフェース（domain 層）→ `ChatRepositoryImpl`（data 層）
- `StateFlow<List<ChatMessage>>`, `StateFlow<ConnectionState>` でリアクティブに公開
- 書き込みは `suspend fun`

### UseCase パターン
- 薄いラッパー。`operator fun invoke()` で呼び出し
- Koin で `factory {}` として注入（毎回新規インスタンス）

### データマッピング（3層）
- **Entity**（Room DB）↔ **DTO**（WebSocket）↔ **Domain Model**
- `MessageMapper` オブジェクトで変換を一元管理

### Backend ライフサイクル抽象化
- `BackendLifecycle` インターフェース + `StateFlow<BackendState>`（STOPPED/STARTING/RUNNING/ERROR）
- `embedded` フレーバー: `EmbeddedBackendLifecycle`（アプリ内で Go 起動）
- `termux` フレーバー: `NoopBackendLifecycle`（外部実行）

### Sealed Interface の活用
- `SttResult`、`ListState`、`SaveState` 等で網羅的パターンマッチ

### WebSocket 通信
- `WebSocketClient` が `StateFlow<ConnectionState>` と `SharedFlow<WsOutgoing>` を公開
- 指数バックオフによる自動再接続
- メッセージタイプ（`status`, `tool_request`, `exit` 等）で分岐処理
