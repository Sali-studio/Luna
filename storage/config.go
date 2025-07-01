package storage

import (
	"encoding/json"
	"os"
	"sync"
)

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
type ReactionRoleConfig struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	RoleID    string `json:"role_id"`
}
type GuildConfig struct {
	Ticket        TicketConfig                   `json:"ticket"`
	Log           LogConfig                      `json:"log"`
	TempVC        TempVCConfig                   `json:"temp_vc"`
	Dashboard     DashboardConfig                `json:"dashboard"`
	ReactionRoles map[string]*ReactionRoleConfig `json:"reaction_roles"`
}
type ConfigStore struct {
	mu      sync.Mutex
	path    string
	Configs map[string]*GuildConfig
}

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
func (s *ConfigStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, &s.Configs)
}
func (s *ConfigStore) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s.Configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
func (s *ConfigStore) GetGuildConfig(guildID string) *GuildConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, ok := s.Configs[guildID]
	if !ok {
		config = &GuildConfig{
			ReactionRoles: make(map[string]*ReactionRoleConfig),
		}
		s.Configs[guildID] = config
	} else if config.ReactionRoles == nil {
		config.ReactionRoles = make(map[string]*ReactionRoleConfig)
	}
	return config
}
func (s *ConfigStore) SaveGuildConfig(guildID string, config *GuildConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Configs[guildID] = config
	return s.save()
}
