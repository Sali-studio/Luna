package storage

import (
	"encoding/json"
	"os"
	"sync"
)

type ReactionRole struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	RoleID    string `json:"role_id"`
}
type ReactionRoleStore struct {
	mu    sync.Mutex
	path  string
	Roles map[string]*ReactionRole
}

func NewReactionRoleStore(path string) (*ReactionRoleStore, error) {
	store := &ReactionRoleStore{
		path:  path,
		Roles: make(map[string]*ReactionRole),
	}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}
func (s *ReactionRoleStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, &s.Roles)
}
func (s *ReactionRoleStore) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s.Roles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
func (s *ReactionRoleStore) Add(rr *ReactionRole) error {
	key := rr.MessageID + ":" + rr.Emoji
	s.Roles[key] = rr
	return s.save()
}
func (s *ReactionRoleStore) Get(messageID, emoji string) (string, bool) {
	key := messageID + ":" + emoji
	rr, found := s.Roles[key]
	if !found {
		return "", false
	}
	return rr.RoleID, true
}
