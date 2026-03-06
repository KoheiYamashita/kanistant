package tools

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// mockUserDirectory implements UserDirectory for testing.
type mockUserDirectory struct {
	users  []*UserInfo
	nextID int
}

func newMockUserDirectory() *mockUserDirectory {
	return &mockUserDirectory{
		users:  []*UserInfo{},
		nextID: 1,
	}
}

func (m *mockUserDirectory) List() []*UserInfo {
	result := make([]*UserInfo, len(m.users))
	copy(result, m.users)
	return result
}

func (m *mockUserDirectory) Get(userID string) *UserInfo {
	for _, u := range m.users {
		if u.ID == userID {
			return u
		}
	}
	return nil
}

func (m *mockUserDirectory) Create(name, channel, channelID string) (*UserInfo, error) {
	u := &UserInfo{
		ID:       fmt.Sprintf("u_%d", m.nextID),
		Name:     name,
		Channels: map[string][]string{},
		Memo:     []string{},
	}
	m.nextID++
	if channel != "" && channelID != "" {
		u.Channels[channel] = []string{channelID}
	}
	m.users = append(m.users, u)
	return u, nil
}

func (m *mockUserDirectory) Update(userID, name string) error {
	for _, u := range m.users {
		if u.ID == userID {
			if name != "" {
				u.Name = name
			}
			return nil
		}
	}
	return fmt.Errorf("user not found: %s", userID)
}

func (m *mockUserDirectory) Delete(userID string) error {
	for i, u := range m.users {
		if u.ID == userID {
			m.users = append(m.users[:i], m.users[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("user not found: %s", userID)
}

func (m *mockUserDirectory) Link(userID, channel, channelID string) error {
	for _, u := range m.users {
		if u.ID == userID {
			u.Channels[channel] = append(u.Channels[channel], channelID)
			return nil
		}
	}
	return fmt.Errorf("user not found: %s", userID)
}

func (m *mockUserDirectory) AddMemo(userID, memo string) error {
	for _, u := range m.users {
		if u.ID == userID {
			u.Memo = append(u.Memo, memo)
			return nil
		}
	}
	return fmt.Errorf("user not found: %s", userID)
}

func (m *mockUserDirectory) RemoveMemo(userID string, index int) error {
	for _, u := range m.users {
		if u.ID == userID {
			if index < 0 || index >= len(u.Memo) {
				return fmt.Errorf("memo index out of range: %d", index)
			}
			u.Memo = append(u.Memo[:index], u.Memo[index+1:]...)
			return nil
		}
	}
	return fmt.Errorf("user not found: %s", userID)
}

func (m *mockUserDirectory) LegacyFilePath() string {
	return ""
}

// mockUserDirectoryWithPath extends mockUserDirectory with a custom legacy path.
type mockUserDirectoryWithPath struct {
	*mockUserDirectory
	legacyPath string
}

func (m *mockUserDirectoryWithPath) LegacyFilePath() string {
	return m.legacyPath
}

// mockExclusiveDirectory implements websocket exclusivity like the real UserStore.
type mockExclusiveDirectory struct {
	*mockUserDirectory
}

func newMockExclusiveDirectory() *mockExclusiveDirectory {
	return &mockExclusiveDirectory{mockUserDirectory: newMockUserDirectory()}
}

func (m *mockExclusiveDirectory) Link(userID, channel, channelID string) error {
	if channel == "websocket" {
		for _, u := range m.users {
			if u.ID != userID && len(u.Channels["websocket"]) > 0 {
				return fmt.Errorf("websocket is already linked to user %s (%s)", u.Name, u.ID)
			}
		}
	}
	return m.mockUserDirectory.Link(userID, channel, channelID)
}

func TestUserTool_NameAndDescription(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	if tool.Name() != "user" {
		t.Errorf("expected name 'user', got %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Error("parameters should not be nil")
	}
}

func TestUserTool_Create(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "create",
		"name":   "Alice",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(dir.users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(dir.users))
	}
	if dir.users[0].Name != "Alice" {
		t.Errorf("expected name Alice, got %q", dir.users[0].Name)
	}
}

func TestUserTool_Create_MissingName(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "create",
	})
	if !result.IsError {
		t.Error("expected error for missing name")
	}
}

func TestUserTool_List(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	_, _ = dir.Create("Alice", "", "")
	_, _ = dir.Create("Bob", "", "")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "list",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestUserTool_List_Empty(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "list",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if result.ForLLM != "No users registered." {
		t.Errorf("expected 'No users registered.', got %q", result.ForLLM)
	}
}

func TestUserTool_Get(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "get",
		"user_id": u.ID,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestUserTool_Get_NotFound(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "get",
		"user_id": "u_nonexistent",
	})
	if !result.IsError {
		t.Error("expected error for non-existent user")
	}
}

func TestUserTool_Get_MissingID(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "get",
	})
	if !result.IsError {
		t.Error("expected error for missing user_id")
	}
}

func TestUserTool_Update(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "update",
		"user_id": u.ID,
		"name":    "Alicia",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if dir.users[0].Name != "Alicia" {
		t.Errorf("expected name Alicia, got %q", dir.users[0].Name)
	}
}

func TestUserTool_Update_NoName(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	// name omitted — should NOT error
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "update",
		"user_id": u.ID,
	})
	if result.IsError {
		t.Fatalf("update with no name should not error, got: %s", result.ForLLM)
	}
	if dir.users[0].Name != "Alice" {
		t.Errorf("name should be unchanged, got %q", dir.users[0].Name)
	}
}

func TestUserTool_Update_MissingID(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "update",
		"name":   "Alicia",
	})
	if !result.IsError {
		t.Error("expected error for missing user_id")
	}
}

func TestUserTool_Delete(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "delete",
		"user_id": u.ID,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(dir.users) != 0 {
		t.Error("expected user to be deleted")
	}
}

func TestUserTool_Delete_NotFound(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "delete",
		"user_id": "u_nonexistent",
	})
	if !result.IsError {
		t.Error("expected error for non-existent user")
	}
}

func TestUserTool_Link(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":     "link",
		"user_id":    u.ID,
		"channel":    "discord",
		"channel_id": "12345",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if ids := dir.users[0].Channels["discord"]; len(ids) != 1 || ids[0] != "12345" {
		t.Errorf("unexpected channels: %v", dir.users[0].Channels)
	}
}

func TestUserTool_Link_MissingParams(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	// Missing channel
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":     "link",
		"user_id":    u.ID,
		"channel_id": "12345",
	})
	if !result.IsError {
		t.Error("expected error for missing channel")
	}

	// Missing channel_id
	result = tool.Execute(context.Background(), map[string]interface{}{
		"action":  "link",
		"user_id": u.ID,
		"channel": "discord",
	})
	if !result.IsError {
		t.Error("expected error for missing channel_id")
	}
}

func TestUserTool_AddMemo(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "add_memo",
		"user_id": u.ID,
		"memo":    "likes tea",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(dir.users[0].Memo) != 1 || dir.users[0].Memo[0] != "likes tea" {
		t.Errorf("unexpected memo: %v", dir.users[0].Memo)
	}
}

func TestUserTool_AddMemo_MissingParams(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "add_memo",
		"user_id": u.ID,
	})
	if !result.IsError {
		t.Error("expected error for missing memo")
	}
}

func TestUserTool_RemoveMemo(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")
	_ = dir.AddMemo(u.ID, "first")
	_ = dir.AddMemo(u.ID, "second")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":     "remove_memo",
		"user_id":    u.ID,
		"memo_index": float64(0),
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(dir.users[0].Memo) != 1 || dir.users[0].Memo[0] != "second" {
		t.Errorf("unexpected memo after remove: %v", dir.users[0].Memo)
	}
}

func TestUserTool_RemoveMemo_MissingIndex(t *testing.T) {
	dir := newMockUserDirectory()
	tool := NewUserTool(dir, false)
	u, _ := dir.Create("Alice", "", "")
	_ = dir.AddMemo(u.ID, "test")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "remove_memo",
		"user_id": u.ID,
	})
	if !result.IsError {
		t.Error("expected error for missing memo_index")
	}
}

func TestUserTool_ReadLegacy(t *testing.T) {
	// hasLegacyFile = false → error
	tool := NewUserTool(newMockUserDirectory(), false)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read_legacy",
	})
	if !result.IsError {
		t.Error("expected error when no legacy file")
	}
}

func TestUserTool_DeleteLegacy(t *testing.T) {
	tmpDir := t.TempDir()
	legacyPath := tmpDir + "/USER.md"
	_ = os.WriteFile(legacyPath, []byte("legacy data"), 0644)

	dir := &mockUserDirectoryWithPath{
		mockUserDirectory: newMockUserDirectory(),
		legacyPath:        legacyPath,
	}
	tool := NewUserTool(dir, true)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "delete_legacy",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	// File should be deleted
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Error("expected USER.md to be deleted")
	}

	// Second call should fail (hasLegacyFile is now false)
	result = tool.Execute(context.Background(), map[string]interface{}{
		"action": "delete_legacy",
	})
	if !result.IsError {
		t.Error("expected error on second delete_legacy call")
	}
}

func TestUserTool_DeleteLegacy_NoFile(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "delete_legacy",
	})
	if !result.IsError {
		t.Error("expected error when hasLegacyFile is false")
	}
}

func TestUserTool_Link_WebSocketExclusive(t *testing.T) {
	dir := newMockExclusiveDirectory()
	tool := NewUserTool(dir, false)

	alice, _ := dir.Create("Alice", "websocket", "ws1")

	bob, _ := dir.Create("Bob", "", "")

	// Link Bob to websocket should fail
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":     "link",
		"user_id":    bob.ID,
		"channel":    "websocket",
		"channel_id": "ws2",
	})
	if !result.IsError {
		t.Error("expected error when linking websocket to second user")
	}

	// Non-websocket link should succeed
	result = tool.Execute(context.Background(), map[string]interface{}{
		"action":     "link",
		"user_id":    alice.ID,
		"channel":    "discord",
		"channel_id": "12345",
	})
	if result.IsError {
		t.Fatalf("non-websocket link should succeed: %s", result.ForLLM)
	}
}

func TestUserTool_UnknownAction(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "unknown",
	})
	if !result.IsError {
		t.Error("expected error for unknown action")
	}
}

func TestUserTool_MissingAction(t *testing.T) {
	tool := NewUserTool(newMockUserDirectory(), false)

	result := tool.Execute(context.Background(), map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error for missing action")
	}
}
