package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewUserStore(t *testing.T) {
	store := NewUserStore(t.TempDir())

	if store == nil {
		t.Fatal("NewUserStore returned nil")
	}
	users := store.List()
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestNewUserStore_LoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	f := usersFile{Users: []*User{
		{ID: "u_abc", Name: "Alice", Channels: map[string][]string{}, Memo: []string{}},
	}}
	data, _ := json.MarshalIndent(f, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "users.json"), data, 0644)

	store := NewUserStore(tmpDir)
	users := store.List()
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("expected name Alice, got %q", users[0].Name)
	}
}

func TestCreate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewUserStore(tmpDir)

	user, err := store.Create("Bob", "discord", "12345")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.Name != "Bob" {
		t.Errorf("expected name Bob, got %q", user.Name)
	}
	if user.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ids := user.Channels["discord"]; len(ids) != 1 || ids[0] != "12345" {
		t.Errorf("unexpected channels: %v", user.Channels)
	}

	// Verify persistence
	store2 := NewUserStore(tmpDir)
	if len(store2.List()) != 1 {
		t.Error("user should be persisted to disk")
	}
}

func TestResolveByChannelID(t *testing.T) {
	store := NewUserStore(t.TempDir())
	store.Create("Alice", "discord", "111")
	store.Create("Bob", "slack", "222")

	// Found
	u := store.ResolveByChannelID("discord", "111")
	if u == nil || u.Name != "Alice" {
		t.Errorf("expected Alice, got %v", u)
	}

	// Not found
	u = store.ResolveByChannelID("discord", "999")
	if u != nil {
		t.Errorf("expected nil, got %v", u)
	}

	// Wrong channel
	u = store.ResolveByChannelID("slack", "111")
	if u != nil {
		t.Errorf("expected nil, got %v", u)
	}
}

func TestResolveByChannelID_WebSocket(t *testing.T) {
	store := NewUserStore(t.TempDir())
	user, _ := store.Create("Alice", "", "")
	store.Link(user.ID, "websocket", "ws1")

	// WebSocket ignores senderID — any linked user is returned
	u := store.ResolveByChannelID("websocket", "anything")
	if u == nil || u.Name != "Alice" {
		t.Errorf("expected Alice for websocket, got %v", u)
	}
}

func TestUpdate(t *testing.T) {
	store := NewUserStore(t.TempDir())
	user, _ := store.Create("Alice", "", "")

	if err := store.Update(user.ID, "Alicia"); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	u := store.Get(user.ID)
	if u.Name != "Alicia" {
		t.Errorf("expected name Alicia, got %q", u.Name)
	}
}

func TestUpdate_EmptyName(t *testing.T) {
	store := NewUserStore(t.TempDir())
	user, _ := store.Create("Alice", "", "")

	// Empty name should not change the existing name
	if err := store.Update(user.ID, ""); err != nil {
		t.Fatalf("Update with empty name failed: %v", err)
	}
	u := store.Get(user.ID)
	if u.Name != "Alice" {
		t.Errorf("expected name Alice (unchanged), got %q", u.Name)
	}
}

func TestLink(t *testing.T) {
	store := NewUserStore(t.TempDir())
	user, _ := store.Create("Alice", "", "")

	if err := store.Link(user.ID, "telegram", "tg123"); err != nil {
		t.Fatalf("Link failed: %v", err)
	}
	u := store.Get(user.ID)
	if ids := u.Channels["telegram"]; len(ids) != 1 || ids[0] != "tg123" {
		t.Errorf("unexpected channels: %v", u.Channels)
	}

	// Idempotent — linking same ID again should be a no-op
	if err := store.Link(user.ID, "telegram", "tg123"); err != nil {
		t.Fatalf("duplicate Link failed: %v", err)
	}
	u = store.Get(user.ID)
	if len(u.Channels["telegram"]) != 1 {
		t.Errorf("duplicate link should be idempotent, got %v", u.Channels["telegram"])
	}
}

func TestAddMemo(t *testing.T) {
	store := NewUserStore(t.TempDir())
	user, _ := store.Create("Alice", "", "")

	if err := store.AddMemo(user.ID, "likes tea"); err != nil {
		t.Fatalf("AddMemo failed: %v", err)
	}
	u := store.Get(user.ID)
	if len(u.Memo) != 1 || u.Memo[0] != "likes tea" {
		t.Errorf("unexpected memo: %v", u.Memo)
	}
}

func TestRemoveMemo(t *testing.T) {
	store := NewUserStore(t.TempDir())
	user, _ := store.Create("Alice", "", "")
	store.AddMemo(user.ID, "first")
	store.AddMemo(user.ID, "second")

	if err := store.RemoveMemo(user.ID, 0); err != nil {
		t.Fatalf("RemoveMemo failed: %v", err)
	}
	u := store.Get(user.ID)
	if len(u.Memo) != 1 || u.Memo[0] != "second" {
		t.Errorf("unexpected memo after remove: %v", u.Memo)
	}

	// Out of range
	if err := store.RemoveMemo(user.ID, 5); err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestDelete(t *testing.T) {
	store := NewUserStore(t.TempDir())
	user, _ := store.Create("Alice", "", "")

	if err := store.Delete(user.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if store.Get(user.ID) != nil {
		t.Error("expected user to be deleted")
	}
	if len(store.List()) != 0 {
		t.Error("expected empty user list")
	}

	// Delete non-existent
	if err := store.Delete("u_nonexistent"); err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestLink_WebSocketExclusive(t *testing.T) {
	store := NewUserStore(t.TempDir())
	alice, _ := store.Create("Alice", "", "")
	bob, _ := store.Create("Bob", "", "")

	// Link Alice to websocket — channelID should be normalised
	if err := store.Link(alice.ID, "websocket", "session-xyz"); err != nil {
		t.Fatalf("first websocket link should succeed: %v", err)
	}
	u := store.Get(alice.ID)
	if ids := u.Channels["websocket"]; len(ids) != 1 || ids[0] != "linked" {
		t.Errorf("websocket channelID should be normalised to 'linked', got %v", ids)
	}

	// Link Bob to websocket should fail
	if err := store.Link(bob.ID, "websocket", "ws2"); err == nil {
		t.Error("expected error when linking websocket to second user")
	}

	// Re-linking Alice should succeed (replace, not accumulate)
	if err := store.Link(alice.ID, "websocket", "different-session"); err != nil {
		t.Fatalf("re-linking websocket should succeed: %v", err)
	}
	u = store.Get(alice.ID)
	if ids := u.Channels["websocket"]; len(ids) != 1 || ids[0] != "linked" {
		t.Errorf("websocket should still have exactly one 'linked' entry, got %v", ids)
	}
}

func TestCreate_WebSocketExclusive(t *testing.T) {
	store := NewUserStore(t.TempDir())
	alice, _ := store.Create("Alice", "websocket", "session-abc-123")

	// channelID should be normalised to "linked"
	u := store.Get(alice.ID)
	if ids := u.Channels["websocket"]; len(ids) != 1 || ids[0] != "linked" {
		t.Errorf("websocket channelID should be normalised to 'linked', got %v", ids)
	}

	// Creating another user with websocket should fail
	_, err := store.Create("Bob", "websocket", "ws2")
	if err == nil {
		t.Error("expected error when creating user with websocket already linked")
	}

	// Creating with non-websocket channel should succeed
	bob, err := store.Create("Bob", "discord", "12345")
	if err != nil {
		t.Fatalf("non-websocket create should succeed: %v", err)
	}
	if bob.Name != "Bob" {
		t.Errorf("expected name Bob, got %q", bob.Name)
	}
}

func TestNeedsMigration(t *testing.T) {
	// No USER.md, no users.json → no migration needed
	store := NewUserStore(t.TempDir())
	if store.NeedsMigration() {
		t.Error("should not need migration without USER.md")
	}

	// USER.md exists, no users.json → migration needed
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "USER.md"), []byte("legacy data"), 0644)
	store = NewUserStore(tmpDir)
	if !store.NeedsMigration() {
		t.Error("should need migration when USER.md exists without users.json")
	}

	// Both exist → no migration needed
	tmpDir = t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "USER.md"), []byte("legacy"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "users.json"), []byte(`{"users":[]}`), 0644)
	store = NewUserStore(tmpDir)
	if store.NeedsMigration() {
		t.Error("should not need migration when both files exist")
	}
}
