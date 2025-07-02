package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// --- 各機能ごとの設定用Struct ---
type TicketConfig struct {
	PanelChannelID string `json:"panel_channel_id"`
	CategoryID     string `json:"category_id"`
	StaffRoleID    string `json:"staff_role_id"`
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
type ReactionRole struct {
	MessageID string
	EmojiID   string
	GuildID   string
	RoleID    string
}
type Schedule struct {
	ID        int
	GuildID   string
	ChannelID string
	CronSpec  string
	Message   string
}

// DBStore はデータベース接続と操作を管理します
type DBStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDBStore は新しいDBStoreを初期化し、データベースに接続します
func NewDBStore(dataSourceName string) (*DBStore, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	store := &DBStore{db: db}
	if err = store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

// initTables は、必要なテーブルが存在しない場合に作成します
func (s *DBStore) initTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS guilds (
			guild_id TEXT PRIMARY KEY,
			ticket_config TEXT DEFAULT '{}',
			log_config TEXT DEFAULT '{}',
			temp_vc_config TEXT DEFAULT '{}',
			dashboard_config TEXT DEFAULT '{}',
			bump_config TEXT DEFAULT '{}',
			ticket_counter INTEGER DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS tickets (
			channel_id TEXT PRIMARY KEY,
			guild_id TEXT,
			user_id TEXT,
			status TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS reaction_roles (
			message_id TEXT, emoji_id TEXT, guild_id TEXT, role_id TEXT,
			PRIMARY KEY (message_id, emoji_id)
		);`,
		`CREATE TABLE IF NOT EXISTS schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT, guild_id TEXT, channel_id TEXT, cron_spec TEXT, message TEXT
		);`,
	}
	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return err
		}
	}
	return nil
}

// Close はデータベース接続を安全に閉じます
func (s *DBStore) Close() {
	s.db.Close()
}

// upsertGuild はguildsテーブルにguildIDが存在することを保証します
func (s *DBStore) upsertGuild(tx *sql.Tx, guildID string) error {
	_, err := tx.Exec("INSERT OR IGNORE INTO guilds (guild_id) VALUES (?)", guildID)
	return err
}

// --- 汎用的な設定の読み書き関数 ---
func (s *DBStore) GetConfig(guildID, configName string, configStruct interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := fmt.Sprintf("SELECT %s FROM guilds WHERE guild_id = ?", configName)
	var configJSON sql.NullString
	err := s.db.QueryRow(query, guildID).Scan(&configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			// 新規サーバーの場合、一度行を作成しておく
			s.mu.RUnlock() // RLockを一度解除して書き込みロックを取得
			s.mu.Lock()
			tx, _ := s.db.Begin()
			s.upsertGuild(tx, guildID)
			tx.Commit()
			s.mu.Unlock()
			s.mu.RLock() // 再度RLock
			return nil
		}
		return err
	}
	if configJSON.Valid {
		return json.Unmarshal([]byte(configJSON.String), configStruct)
	}
	return nil
}

func (s *DBStore) SaveConfig(guildID, configName string, configStruct interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.upsertGuild(tx, guildID); err != nil {
		return err
	}
	data, err := json.Marshal(configStruct)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("UPDATE guilds SET %s = ? WHERE guild_id = ?", configName)
	_, err = tx.Exec(query, string(data), guildID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// --- チケット専用関数 ---
func (s *DBStore) GetNextTicketCounter(guildID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := s.upsertGuild(tx, guildID); err != nil {
		return 0, err
	}

	var counter int
	err = tx.QueryRow("SELECT ticket_counter FROM guilds WHERE guild_id = ?", guildID).Scan(&counter)
	if err != nil {
		return 0, err
	}

	counter++
	_, err = tx.Exec("UPDATE guilds SET ticket_counter = ? WHERE guild_id = ?", counter, guildID)
	if err != nil {
		return 0, err
	}
	return counter, tx.Commit()
}

func (s *DBStore) CreateTicketRecord(channelID, guildID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT INTO tickets (channel_id, guild_id, user_id, status) VALUES (?, ?, ?, 'open')", channelID, guildID, userID)
	return err
}

func (s *DBStore) CloseTicketRecord(channelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE tickets SET status = 'closed' WHERE channel_id = ?", channelID)
	return err
}

// --- リアクションロールとスケジュールのための専用関数 ---
func (s *DBStore) GetReactionRole(guildID, messageID, emojiID string) (ReactionRole, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var rr ReactionRole
	err := s.db.QueryRow("SELECT role_id FROM reaction_roles WHERE guild_id = ? AND message_id = ? AND emoji_id = ?", guildID, messageID, emojiID).Scan(&rr.RoleID)
	rr.GuildID, rr.MessageID, rr.EmojiID = guildID, messageID, emojiID
	return rr, err
}

func (s *DBStore) SaveReactionRole(rr ReactionRole) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT OR REPLACE INTO reaction_roles (message_id, emoji_id, guild_id, role_id) VALUES (?, ?, ?, ?)", rr.MessageID, rr.EmojiID, rr.GuildID, rr.RoleID)
	return err
}

func (s *DBStore) SaveSchedule(schedule Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT INTO schedules (guild_id, channel_id, cron_spec, message) VALUES (?, ?, ?, ?)", schedule.GuildID, schedule.ChannelID, schedule.CronSpec, schedule.Message)
	return err
}

func (s *DBStore) GetAllSchedules() ([]Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, guild_id, channel_id, cron_spec, message FROM schedules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []Schedule
	for rows.Next() {
		var sc Schedule
		if err := rows.Scan(&sc.ID, &sc.GuildID, &sc.ChannelID, &sc.CronSpec, &sc.Message); err != nil {
			return nil, err
		}
		schedules = append(schedules, sc)
	}
	return schedules, nil
}
