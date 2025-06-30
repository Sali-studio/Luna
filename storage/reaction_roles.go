package storage

import (
	"encoding/json"
	"os"
	"sync"
)

// リアクションロールの設定を保存するための構造体
type ReactionRole struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	RoleID    string `json:"role_id"`
}

// 設定をファイルに保存/読み込みするための構造体
type ReactionRoleStore struct {
	mu    sync.Mutex
	path  string
	Roles map[string]*ReactionRole // メッセージIDと絵文字をキーにする
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

// load はファイルから設定を読み込みます
func (s *ReactionRoleStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, &s.Roles)
}

// save は現在の設定をファイルに保存します
func (s *ReactionRoleStore) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.Roles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Add は新しいリアクションロール設定を追加します
func (s *ReactionRoleStore) Add(rr *ReactionRole) error {
	key := rr.MessageID + ":" + rr.Emoji
	s.Roles[key] = rr
	return s.save()
}

// Get は指定されたメッセージと絵文字に対応するロールIDを返します
func (s *ReactionRoleStore) Get(messageID, emoji string) (string, bool) {
	key := messageID + ":" + emoji
	rr, found := s.Roles[key]
	if !found {
		return "", false
	}
	return rr.RoleID, true
}
