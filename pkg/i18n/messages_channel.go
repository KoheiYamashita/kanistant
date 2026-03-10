package i18n

func init() {
	register("en", map[string]string{
		// Telegram
		"channel.thinking": "Thinking... 💭",

		// WebSocket
		"channel.config_required": "Configuration required",

		// Telegram commands (/help, /start, /show, /list)
		"cmd.help": `/start - Start the bot
/help - Show this help message
/show [model|channel] - Show current configuration
/list [models|channels] - List available options
`,
		"cmd.start":         "Hello! I am ClawDroid 🦞",
		"cmd.show.usage":    "Usage: /show [model|channel]",
		"cmd.show.model":    "Current Model: %s",
		"cmd.show.channel":  "Current Channel: telegram",
		"cmd.show.unknown":  "Unknown parameter: %s. Try 'model' or 'channel'.",
		"cmd.list.usage":    "Usage: /list [models|channels]",
		"cmd.list.models":   "Configured Model: %s\n\nTo change models, update config.json",
		"cmd.list.channels": "Enabled Channels:\n- %s",
		"cmd.list.unknown":  "Unknown parameter: %s. Try 'models' or 'channels'.",

		// Agent loop commands (/show, /list, /switch)
		// cmd.show.usage and cmd.list.usage are shared with Telegram commands
		"agent.cmd.show.model":        "Current model: %s",
		"agent.cmd.show.channel":      "Current channel: %s",
		"agent.cmd.show.unknown":      "Unknown show target: %s",
		"agent.cmd.list.models":       "Available models: glm-4.7, claude-3-5-sonnet, gpt-4o (configured in config.json/env)",
		"agent.cmd.list.no_channels":  "No channels enabled",
		"agent.cmd.list.channels":     "Enabled channels: %s",
		"agent.cmd.list.unknown":      "Unknown list target: %s",
		"agent.cmd.switch.usage":      "Usage: /switch [model|channel] to <name>",
		"agent.cmd.switch.model":      "Switched model from %s to %s",
		"agent.cmd.switch.channel":    "Switched target channel to %s (Note: this currently only validates existence)",
		"agent.cmd.switch.not_found":  "Channel '%s' not found or not enabled",
		"agent.cmd.switch.unknown":    "Unknown switch target: %s",
		"agent.cmd.channel_mgr_error": "Channel manager not initialized",
	})

	register("ja", map[string]string{
		// Telegram
		"channel.thinking": "考え中... 💭",

		// WebSocket
		"channel.config_required": "設定が必要です",

		// Telegram commands
		"cmd.help": `/start - ボットを開始
/help - このヘルプメッセージを表示
/show [model|channel] - 現在の設定を表示
/list [models|channels] - 利用可能なオプションを一覧表示
`,
		"cmd.start":         "こんにちは！ClawDroid です 🦞",
		"cmd.show.usage":    "使い方: /show [model|channel]",
		"cmd.show.model":    "現在のモデル: %s",
		"cmd.show.channel":  "現在のチャンネル: telegram",
		"cmd.show.unknown":  "不明なパラメータ: %s。'model' か 'channel' を指定してください。",
		"cmd.list.usage":    "使い方: /list [models|channels]",
		"cmd.list.models":   "設定済みモデル: %s\n\nモデルを変更するには config.json を更新してください",
		"cmd.list.channels": "有効なチャンネル:\n- %s",
		"cmd.list.unknown":  "不明なパラメータ: %s。'models' か 'channels' を指定してください。",

		// Agent loop commands
		// cmd.show.usage and cmd.list.usage are shared with Telegram commands
		"agent.cmd.show.model":        "現在のモデル: %s",
		"agent.cmd.show.channel":      "現在のチャンネル: %s",
		"agent.cmd.show.unknown":      "不明な表示対象: %s",
		"agent.cmd.list.models":       "利用可能なモデル: glm-4.7, claude-3-5-sonnet, gpt-4o（config.json/env で設定）",
		"agent.cmd.list.no_channels":  "有効なチャンネルはありません",
		"agent.cmd.list.channels":     "有効なチャンネル: %s",
		"agent.cmd.list.unknown":      "不明な一覧対象: %s",
		"agent.cmd.switch.usage":      "使い方: /switch [model|channel] to <名前>",
		"agent.cmd.switch.model":      "モデルを %s から %s に切り替えました",
		"agent.cmd.switch.channel":    "対象チャンネルを %s に切り替えました（注: 現在は存在確認のみ）",
		"agent.cmd.switch.not_found":  "チャンネル '%s' が見つからないか有効ではありません",
		"agent.cmd.switch.unknown":    "不明な切り替え対象: %s",
		"agent.cmd.channel_mgr_error": "チャンネルマネージャーが初期化されていません",
	})
}
