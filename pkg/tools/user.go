package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// UserDirectory is the interface for user directory operations.
// Implemented by agent.UserStore.
type UserDirectory interface {
	List() []*UserInfo
	Get(userID string) *UserInfo
	Create(name, channel, channelID string) (*UserInfo, error)
	Update(userID, name string) error
	Delete(userID string) error
	Link(userID, channel, channelID string) error
	AddMemo(userID, memo string) error
	RemoveMemo(userID string, index int) error
	LegacyFilePath() string
}

// UserInfo represents a user for the tool layer.
type UserInfo struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Channels map[string][]string `json:"channels"`
	Memo     []string            `json:"memo"`
}

// UserTool provides CRUD operations for the user directory.
type UserTool struct {
	dir           UserDirectory
	hasLegacyFile bool
}

func NewUserTool(dir UserDirectory, hasLegacyFile bool) *UserTool {
	return &UserTool{
		dir:           dir,
		hasLegacyFile: hasLegacyFile,
	}
}

// requireString extracts a required string argument, returning an error result if missing.
func requireString(args map[string]interface{}, key, action string) (string, *ToolResult) {
	v, _ := args[key].(string)
	if v == "" {
		return "", ErrorResult(fmt.Sprintf("%s is required for %s", key, action))
	}
	return v, nil
}

func (t *UserTool) Name() string {
	return "user"
}

func (t *UserTool) Description() string {
	desc := "Manage user directory. Actions: list, get, create, update, delete, link, add_memo, remove_memo"
	if t.hasLegacyFile {
		desc += ", read_legacy, delete_legacy"
	}
	return desc
}

func (t *UserTool) Parameters() map[string]interface{} {
	actions := []string{"list", "get", "create", "update", "delete", "link", "add_memo", "remove_memo"}
	if t.hasLegacyFile {
		actions = append(actions, "read_legacy", "delete_legacy")
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The action to perform",
				"enum":        actions,
			},
			"user_id": map[string]interface{}{
				"type":        "string",
				"description": "User ID (required for get, update, delete, link, add_memo, remove_memo)",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "User name (required for create, optional for update)",
			},
			"channel": map[string]interface{}{
				"type":        "string",
				"description": "Channel name (required for link, optional for create)",
			},
			"channel_id": map[string]interface{}{
				"type":        "string",
				"description": "Channel-specific user ID (required for link, optional for create)",
			},
			"memo": map[string]interface{}{
				"type":        "string",
				"description": "Memo text (required for add_memo)",
			},
			"memo_index": map[string]interface{}{
				"type":        "number",
				"description": "Memo index to remove (required for remove_memo, 0-based)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *UserTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "list":
		return t.execList()
	case "get":
		return t.execGet(args)
	case "create":
		return t.execCreate(args)
	case "update":
		return t.execUpdate(args)
	case "delete":
		return t.execDelete(args)
	case "link":
		return t.execLink(args)
	case "add_memo":
		return t.execAddMemo(args)
	case "remove_memo":
		return t.execRemoveMemo(args)
	case "read_legacy":
		return t.execReadLegacy()
	case "delete_legacy":
		return t.execDeleteLegacy()
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *UserTool) execList() *ToolResult {
	users := t.dir.List()
	if len(users) == 0 {
		return SilentResult("No users registered.")
	}
	data, _ := json.MarshalIndent(users, "", "  ")
	return SilentResult(string(data))
}

func (t *UserTool) execGet(args map[string]interface{}) *ToolResult {
	userID, errResult := requireString(args, "user_id", "get")
	if errResult != nil {
		return errResult
	}
	user := t.dir.Get(userID)
	if user == nil {
		return ErrorResult(fmt.Sprintf("user not found: %s", userID))
	}
	data, _ := json.MarshalIndent(user, "", "  ")
	return SilentResult(string(data))
}

func (t *UserTool) execCreate(args map[string]interface{}) *ToolResult {
	name, errResult := requireString(args, "name", "create")
	if errResult != nil {
		return errResult
	}
	channel, _ := args["channel"].(string)
	channelID, _ := args["channel_id"].(string)

	user, err := t.dir.Create(name, channel, channelID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create user: %v", err))
	}
	data, _ := json.MarshalIndent(user, "", "  ")
	return SilentResult(fmt.Sprintf("User created:\n%s", string(data)))
}

func (t *UserTool) execUpdate(args map[string]interface{}) *ToolResult {
	userID, errResult := requireString(args, "user_id", "update")
	if errResult != nil {
		return errResult
	}
	name, _ := args["name"].(string)
	if err := t.dir.Update(userID, name); err != nil {
		return ErrorResult(fmt.Sprintf("failed to update user: %v", err))
	}
	return SilentResult(fmt.Sprintf("User %s updated successfully.", userID))
}

func (t *UserTool) execDelete(args map[string]interface{}) *ToolResult {
	userID, errResult := requireString(args, "user_id", "delete")
	if errResult != nil {
		return errResult
	}
	if err := t.dir.Delete(userID); err != nil {
		return ErrorResult(fmt.Sprintf("failed to delete user: %v", err))
	}
	return SilentResult(fmt.Sprintf("User %s deleted.", userID))
}

func (t *UserTool) execLink(args map[string]interface{}) *ToolResult {
	userID, errResult := requireString(args, "user_id", "link")
	if errResult != nil {
		return errResult
	}
	channel, errResult := requireString(args, "channel", "link")
	if errResult != nil {
		return errResult
	}
	channelID, errResult := requireString(args, "channel_id", "link")
	if errResult != nil {
		return errResult
	}
	if err := t.dir.Link(userID, channel, channelID); err != nil {
		return ErrorResult(fmt.Sprintf("failed to link: %v", err))
	}
	return SilentResult(fmt.Sprintf("Linked %s:%s to user %s.", channel, channelID, userID))
}

func (t *UserTool) execAddMemo(args map[string]interface{}) *ToolResult {
	userID, errResult := requireString(args, "user_id", "add_memo")
	if errResult != nil {
		return errResult
	}
	memo, errResult := requireString(args, "memo", "add_memo")
	if errResult != nil {
		return errResult
	}
	if err := t.dir.AddMemo(userID, memo); err != nil {
		return ErrorResult(fmt.Sprintf("failed to add memo: %v", err))
	}
	return SilentResult("Memo added successfully.")
}

func (t *UserTool) execRemoveMemo(args map[string]interface{}) *ToolResult {
	userID, errResult := requireString(args, "user_id", "remove_memo")
	if errResult != nil {
		return errResult
	}
	indexF, ok := args["memo_index"].(float64)
	if !ok {
		return ErrorResult("memo_index is required for remove_memo")
	}
	if err := t.dir.RemoveMemo(userID, int(indexF)); err != nil {
		return ErrorResult(fmt.Sprintf("failed to remove memo: %v", err))
	}
	return SilentResult("Memo removed successfully.")
}

func (t *UserTool) execDeleteLegacy() *ToolResult {
	if !t.hasLegacyFile {
		return ErrorResult("no legacy USER.md file found")
	}
	if err := os.Remove(t.dir.LegacyFilePath()); err != nil {
		return ErrorResult(fmt.Sprintf("failed to delete USER.md: %v", err))
	}
	t.hasLegacyFile = false
	return SilentResult("USER.md has been deleted.")
}

func (t *UserTool) execReadLegacy() *ToolResult {
	if !t.hasLegacyFile {
		return ErrorResult("no legacy USER.md file found")
	}
	data, err := os.ReadFile(t.dir.LegacyFilePath())
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read USER.md: %v", err))
	}
	return SilentResult(fmt.Sprintf(`Legacy USER.md contents:

%s

--- Migration Guide ---
To migrate this data to the new user directory:
1. Review the contents above with the user
2. Use 'create' to create a user with their name
3. Use 'add_memo' to record relevant preferences (language, timezone, etc.)
4. Use 'link' to associate their channel IDs (ask the user for their IDs on each channel)
5. Once migration is complete, inform the user that USER.md is no longer needed
6. Use 'delete_legacy' to delete the old USER.md file`, string(data)))
}
