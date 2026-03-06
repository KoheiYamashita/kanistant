package agent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/KarakuriAgent/clawdroid/pkg/tools"
)

// User represents a user in the user directory.
type User struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Channels map[string][]string `json:"channels"`
	Memo     []string            `json:"memo"`
}

// usersFile is the JSON structure for users.json.
type usersFile struct {
	Users []*User `json:"users"`
}

// UserStore manages the user directory (users.json).
type UserStore struct {
	mu       sync.RWMutex
	dataDir  string
	filePath string
	users    []*User
	// needsMigration is true when USER.md exists but users.json does not
	needsMigration bool
}

// NewUserStore creates a new UserStore and loads existing data.
func NewUserStore(dataDir string) *UserStore {
	filePath := filepath.Join(dataDir, "users.json")

	store := &UserStore{
		dataDir:  dataDir,
		filePath: filePath,
	}

	// Check migration status before loading
	_, usersErr := os.Stat(filePath)
	_, userMDErr := os.Stat(filepath.Join(dataDir, "USER.md"))
	store.needsMigration = os.IsNotExist(usersErr) && userMDErr == nil

	store.load()
	return store
}

// NeedsMigration returns true if USER.md exists but users.json does not.
func (s *UserStore) NeedsMigration() bool {
	return s.needsMigration
}

// LegacyFilePath returns the path to the legacy USER.md file.
func (s *UserStore) LegacyFilePath() string {
	return filepath.Join(s.dataDir, "USER.md")
}

func (s *UserStore) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		s.users = []*User{}
		return
	}

	var f usersFile
	if err := json.Unmarshal(data, &f); err != nil {
		s.users = []*User{}
		return
	}
	s.users = f.Users
	if s.users == nil {
		s.users = []*User{}
	}
}

func (s *UserStore) save() error {
	f := usersFile{Users: s.users}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal users: %w", err)
	}

	// Atomic write: temp file → rename
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tempFile, s.filePath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func generateUserID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "u_" + hex.EncodeToString(b)
}

// findByID returns the user with the given ID, or nil if not found.
// Caller must hold at least a read lock.
func (s *UserStore) findByID(userID string) *User {
	for _, u := range s.users {
		if u.ID == userID {
			return u
		}
	}
	return nil
}

var errUserNotFound = fmt.Errorf("user not found")

// ResolveByChannelID looks up a user by channel and sender ID.
// For websocket channel, senderID is ignored and any user linked to websocket is returned.
func (s *UserStore) ResolveByChannelID(channel, senderID string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		ids, ok := u.Channels[channel]
		if !ok {
			continue
		}
		if channel == "websocket" {
			// WebSocket: return any user linked to websocket channel
			return u
		}
		for _, id := range ids {
			if id == senderID {
				return u
			}
		}
	}
	return nil
}

// Get returns a user by ID.
func (s *UserStore) Get(userID string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.findByID(userID)
}

// List returns all users.
func (s *UserStore) List() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*User, len(s.users))
	copy(result, s.users)
	return result
}

// Create creates a new user with an initial channel ID binding.
func (s *UserStore) Create(name, channel, channelID string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// WebSocket is exclusive: only one user may be linked at a time.
	// The channelID is meaningless for websocket (resolution ignores it),
	// so normalise to a fixed marker to prevent session-specific IDs leaking in.
	if channel == "websocket" {
		channelID = "linked"
		for _, other := range s.users {
			if len(other.Channels["websocket"]) > 0 {
				return nil, fmt.Errorf("websocket is already linked to user %s (%s)", other.Name, other.ID)
			}
		}
	}

	user := &User{
		ID:       generateUserID(),
		Name:     name,
		Channels: map[string][]string{},
		Memo:     []string{},
	}
	if channel != "" && channelID != "" {
		user.Channels[channel] = []string{channelID}
	}

	s.users = append(s.users, user)
	if err := s.save(); err != nil {
		// Rollback
		s.users = s.users[:len(s.users)-1]
		return nil, err
	}
	return user, nil
}

// Update updates a user's name.
func (s *UserStore) Update(userID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := s.findByID(userID)
	if u == nil {
		return fmt.Errorf("%w: %s", errUserNotFound, userID)
	}
	if name != "" {
		u.Name = name
	}
	return s.save()
}

// Link adds a channel ID to a user.
func (s *UserStore) Link(userID, channel, channelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := s.findByID(userID)
	if u == nil {
		return fmt.Errorf("%w: %s", errUserNotFound, userID)
	}
	// WebSocket is exclusive: only one user may be linked at a time.
	// Normalise channelID to a fixed marker (resolution ignores it).
	if channel == "websocket" {
		channelID = "linked"
		for _, other := range s.users {
			if other.ID != userID && len(other.Channels["websocket"]) > 0 {
				return fmt.Errorf("websocket is already linked to user %s (%s)", other.Name, other.ID)
			}
		}
		u.Channels["websocket"] = []string{channelID}
		return s.save()
	}
	for _, id := range u.Channels[channel] {
		if id == channelID {
			return nil // Already linked
		}
	}
	u.Channels[channel] = append(u.Channels[channel], channelID)
	return s.save()
}

// AddMemo adds a memo entry to a user.
func (s *UserStore) AddMemo(userID, memo string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := s.findByID(userID)
	if u == nil {
		return fmt.Errorf("%w: %s", errUserNotFound, userID)
	}
	u.Memo = append(u.Memo, memo)
	return s.save()
}

// RemoveMemo removes a memo entry from a user by index.
func (s *UserStore) RemoveMemo(userID string, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := s.findByID(userID)
	if u == nil {
		return fmt.Errorf("%w: %s", errUserNotFound, userID)
	}
	if index < 0 || index >= len(u.Memo) {
		return fmt.Errorf("memo index out of range: %d", index)
	}
	u.Memo = append(u.Memo[:index], u.Memo[index+1:]...)
	return s.save()
}

// Delete removes a user by ID.
func (s *UserStore) Delete(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, u := range s.users {
		if u.ID == userID {
			s.users = append(s.users[:i], s.users[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("%w: %s", errUserNotFound, userID)
}

// --- tools.UserDirectory adapter methods ---

func userToInfo(u *User) *tools.UserInfo {
	if u == nil {
		return nil
	}
	return &tools.UserInfo{
		ID:       u.ID,
		Name:     u.Name,
		Channels: u.Channels,
		Memo:     u.Memo,
	}
}

// AsDirectory returns the UserStore as a tools.UserDirectory.
// List and Get are adapted to return tools.UserInfo.
func (s *UserStore) AsDirectory() tools.UserDirectory {
	return &userStoreAdapter{store: s}
}

type userStoreAdapter struct {
	store *UserStore
}

func (a *userStoreAdapter) List() []*tools.UserInfo {
	users := a.store.List()
	result := make([]*tools.UserInfo, len(users))
	for i, u := range users {
		result[i] = userToInfo(u)
	}
	return result
}

func (a *userStoreAdapter) Get(userID string) *tools.UserInfo {
	return userToInfo(a.store.Get(userID))
}

func (a *userStoreAdapter) Create(name, channel, channelID string) (*tools.UserInfo, error) {
	u, err := a.store.Create(name, channel, channelID)
	if err != nil {
		return nil, err
	}
	return userToInfo(u), nil
}

func (a *userStoreAdapter) Update(userID, name string) error {
	return a.store.Update(userID, name)
}

func (a *userStoreAdapter) Delete(userID string) error {
	return a.store.Delete(userID)
}

func (a *userStoreAdapter) Link(userID, channel, channelID string) error {
	return a.store.Link(userID, channel, channelID)
}

func (a *userStoreAdapter) AddMemo(userID, memo string) error {
	return a.store.AddMemo(userID, memo)
}

func (a *userStoreAdapter) RemoveMemo(userID string, index int) error {
	return a.store.RemoveMemo(userID, index)
}

func (a *userStoreAdapter) LegacyFilePath() string {
	return a.store.LegacyFilePath()
}
