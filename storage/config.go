package storage

import (
	"encoding/json"
	"os"
	"sync"
)

// --- 各機能ごとの設定 ---

type TicketConfig struct {
	PanelChannelID string `json:"panel_channel_id"`
	CategoryID     string `json:"category_id"`
	StaffRoleID    string `json:"staff_role_id"`
	Counter        int    `json:"counter"`
}

type LogConfig struct {
	ChannelID string `json:"channel_id"`
}

type TempVCConfig struct {
	LobbyID    string `json:"lobby_id"`
	CategoryID string `json:"category_id"`
}

type DashboardConfig struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
}

type BumpConfig struct {
	ChannelID string `json:"channel_id"`
	RoleID    string `json:"role_id"`
	Reminder  bool   `json:"reminder"`
}

// --- サーバー全体の総合設定 ---

type GuildConfig struct {
	Ticket        TicketConfig      `json:"ticket"`
	Log           LogConfig         `json:"log"`
	TempVC        TempVCConfig      `json:"temp_vc"`
	Dashboard     DashboardConfig   `json:"dashboard"`
	Bump          BumpConfig        `json:"bump"`
	ReactionRoles map[string]string `json:"reaction_roles"` // messageID_emoji -> roleID
}

// ConfigStore は設定ファイル(config.json)の読み書きを管理します
type ConfigStore struct {
	mu      sync.Mutex
	path    string
	Configs map[string]*GuildConfig // GuildIDをキーとする設定のマップ
}

// NewConfigStore は新しいConfigStoreを初期化して返します
func NewConfigStore(path string) (*ConfigStore, error) {
	store := &ConfigStore{
		path:    path,
		Configs: make(map[string]*GuildConfig),
	}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}

// load はファイルから設定を読み込みます
func (s *ConfigStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(file) == 0 {
		s.Configs = make(map[string]*GuildConfig)
		return nil
	}
	return json.Unmarshal(file, &s.Configs)
}

// save は現在の設定をファイルに書き込みます
func (s *ConfigStore) save() error {
	data, err := json.MarshalIndent(s.Configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// GetGuildConfig は指定されたGuildIDの設定を取得します。存在しない場合は新しい設定を作成して返します。
func (s *ConfigStore) GetGuildConfig(guildID string) *GuildConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, ok := s.Configs[guildID]
	if !ok {
		config = &GuildConfig{
			ReactionRoles: make(map[string]string),
		}
		s.Configs[guildID] = config
	}
	return config
}

// Save はすべての設定をファイルに保存するパブリックメソッドです
func (s *ConfigStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}
