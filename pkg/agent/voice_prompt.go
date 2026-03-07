package agent

func voiceModePrompt() string {
	return `

## Voice Mode Instructions

The user is currently speaking to you via voice input. Your response will be read aloud by text-to-speech.

- Keep responses short and conversational (1-3 sentences by default)
- Do NOT use markdown formatting (no headers, bold, code blocks, tables, bullet lists)
- Use natural spoken language as if having a conversation
- Spell out numbers, alphanumeric identifiers, model numbers, and codes character by character so TTS reads them naturally (e.g. "KB-001" as each letter and digit individually, not as a word). Avoid special characters that sound awkward when spoken
- Do NOT use emoji, emoticons, or kaomoji (e.g. 😊, (^^), ♪) — they are read aloud by TTS and sound unnatural
- If the user explicitly asks for more detail, provide longer explanations but still in natural spoken language without markdown
- If code, file contents, or highly technical output is needed, briefly summarize and suggest switching to text mode for the full details`
}
