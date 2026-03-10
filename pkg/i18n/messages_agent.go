package i18n

func init() {
	register("en", map[string]string{
		"agent.migration_notice":         "USER.md found. User management has migrated to a new format (users.json).\nAsk in chat to migrate, or update manually.\n\nFor manual update, create ~/.clawdroid/data/users.json in this format:\n```json\n{\n  \"users\": [{\n    \"name\": \"Your Name\",\n    \"channels\": { \"websocket\": [\"default\"] },\n    \"memo\": [\"Preferred language: English\"]\n  }]\n}\n```",
		"agent.context_window_warning":   "⚠️ Context window exceeded. Compressing history and retrying...",
		"agent.memory_threshold_warning": "⚠️ Memory threshold reached. Optimizing conversation history...",
		"agent.rate_limited":             "Rate limited: %s. Please try again later.",
		"agent.rate_limited_tool":        "Rate limited: %s",
	})

	register("ja", map[string]string{
		"agent.migration_notice":         "USER.md が見つかりました。ユーザー管理が新しい形式（users.json）に変わりました。\nチャットで移行を依頼するか、手動で更新してください。\n\n手動更新の場合、以下の形式で ~/.clawdroid/data/users.json を作成:\n```json\n{\n  \"users\": [{\n    \"name\": \"あなたの名前\",\n    \"channels\": { \"websocket\": [\"default\"] },\n    \"memo\": [\"Preferred language: Japanese\"]\n  }]\n}\n```",
		"agent.context_window_warning":   "⚠️ コンテキストウィンドウの上限を超えました。履歴を圧縮してリトライしています...",
		"agent.memory_threshold_warning": "⚠️ メモリしきい値に達しました。会話履歴を最適化しています...",
		"agent.rate_limited":             "レート制限中: %s。しばらくしてからお試しください。",
		"agent.rate_limited_tool":        "レート制限中: %s",
	})
}
