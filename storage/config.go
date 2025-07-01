package storage

import (
	"encoding/json"
	"io/ioutil"
	"luna/logger"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// GuildConfig は各サーバーごとの設定を保持します
type GuildConfig struct {
	Ticket struct {
		CategoryID  string `json:"category_id"`
		StaffRoleID string `json:"staff_role_id"`
	} `json:"ticket"`
	Bump struct {
		Reminder    bool   `json:"reminder"`
		BumpRoleID  string `json:"bump_role_id"`
		BumpChannel string `json:"bump_channel"`
	} `json:"bump"`
	ReactionRole map[string]string `json:"reaction_role"` // messageID_emoji -> roleID
}

// NewGuildConfig はデフォルト値でGuildConfigを生成します
func NewGuildConfig() *GuildConfig {
	return &GuildConfig{
		ReactionRole: make(map[string]string),
	}
}

// ConfigStore は設定ファイル(config.json)の読み書きを管理します
type ConfigStore struct {
	filePath string
	mu       sync.RWMutex
	data     []byte
}

// NewConfigStore は新しいConfigStoreを初期化して返します
func NewConfigStore(filePath string) (*ConfigStore, error) {
	store := &ConfigStore{
		filePath: filePath,
	}
	if err := store.Load(); err != nil {
		// ファイルが存在しない場合は空のJSONで初期化
		logger.Warning.Printf("設定ファイル '%s' が見つかりませんでした。新しいファイルを作成します。", filePath)
		store.data = []byte("{}")
		if err := store.Save(); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// Load はファイルから設定を読み込みます
func (s *ConfigStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	s.data, err = ioutil.ReadFile(s.filePath)
	return err
}

// Save は現在の設定をファイルに書き込みます
func (s *ConfigStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return ioutil.WriteFile(s.filePath, s.data, 0644)
}

// GetGuildConfig は指定されたGuildIDの設定を取得します
func (s *ConfigStore) GetGuildConfig(guildID string) *GuildConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configJSON := gjson.Get(string(s.data), guildID).Raw
	if configJSON == "" {
		return NewGuildConfig()
	}

	var config GuildConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		logger.Error.Printf("Guild %s の設定のパースに失敗しました: %v", guildID, err)
		return NewGuildConfig()
	}
	return &config
}

// SetGuildConfig は指定されたGuildIDの設定を更新します
func (s *ConfigStore) SetGuildConfig(guildID string, config *GuildConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	// sjsonを使ってJSONデータを更新
	s.data, err = sjson.SetRaw(string(s.data), guildID, string(configJSON))
	return err
}
