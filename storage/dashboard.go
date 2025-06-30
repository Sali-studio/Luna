package storage

import (
	"encoding/json"
	"os"
	"sync"
)

// ダッシュボードの設定を保存するための構造体
type DashboardConfig struct {
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
}

// 設定をファイルに保存/読み込みするための構造体
type DashboardStore struct {
	mu      sync.Mutex
	path    string
	Configs map[string]*DashboardConfig // Key: GuildID
}

func NewDashboardStore(path string) (*DashboardStore, error) {
	store := &DashboardStore{
		path:    path,
		Configs: make(map[string]*DashboardConfig),
	}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}

func (s *DashboardStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, &s.Configs)
}

func (s *DashboardStore) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s.Configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Set はサーバーのダッシュボード設定を保存します
func (s *DashboardStore) Set(config *DashboardConfig) error {
	s.Configs[config.GuildID] = config
	return s.save()
}
