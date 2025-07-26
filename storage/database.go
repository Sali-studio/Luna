package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// CachedMessage はDBに保存するメッセージの構造体です
type CachedMessage struct {
	MessageID string
	Content   string
	AuthorID  string
}

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

type WelcomeConfig struct {
	Enabled   bool   `json:"enabled"`
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}

// WordCount はユーザーごとの単語カウントを保持する構造体です
type WordCount struct {
	UserID string
	Word   string
	Count  int
}

type DBStore struct {
	db *sql.DB
	mu sync.RWMutex
}

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

func (s *DBStore) initTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS guilds (
			guild_id TEXT PRIMARY KEY,
			ticket_config TEXT DEFAULT '{}',
			log_config TEXT DEFAULT '{}',
			temp_vc_config TEXT DEFAULT '{}',
			dashboard_config TEXT DEFAULT '{}',
			bump_config TEXT DEFAULT '{}',
			welcome_config TEXT DEFAULT '{}',
			ticket_counter INTEGER DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS tickets (
			channel_id TEXT PRIMARY KEY,
			guild_id TEXT,
			user_id TEXT,
			status TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS message_cache (
			message_id TEXT PRIMARY KEY,
			content TEXT,
			author_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS quiz_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			topic TEXT NOT NULL,
			question TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS word_counts (
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			word TEXT NOT NULL,
			count INTEGER DEFAULT 0,
			PRIMARY KEY (guild_id, user_id, word)
		);`,
		`CREATE TABLE IF NOT EXISTS countable_words (
			guild_id TEXT NOT NULL,
			word TEXT NOT NULL,
			PRIMARY KEY (guild_id, word)
		);`,
	}
	}
	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return err
		}
	}
	return nil
}

// CreateMessageCache はメッセージをDBに保存します
func (s *DBStore) CreateMessageCache(messageID, content, authorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT OR REPLACE INTO message_cache (message_id, content, author_id) VALUES (?, ?, ?)", messageID, content, authorID)
	return err
}

// GetMessageCache はメッセージをDBから取得し、その後削除します
func (s *DBStore) GetMessageCache(messageID string) (*CachedMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msg CachedMessage
	err := s.db.QueryRow("SELECT message_id, content, author_id FROM message_cache WHERE message_id = ?", messageID).Scan(&msg.MessageID, &msg.Content, &msg.AuthorID)
	if err != nil {
		return nil, err
	}

	// 取得後、古いレコードなので削除
	_, err = s.db.Exec("DELETE FROM message_cache WHERE message_id = ?", messageID)
	if err != nil {
		// 削除に失敗しても、取得はできているのでメッセージは返す
		return &msg, nil
	}

	return &msg, nil
}

func (s *DBStore) Close() {
	s.db.Close()
}

func (s *DBStore) PingDB() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Ping()
}

func (s *DBStore) upsertGuild(tx *sql.Tx, guildID string) error {
	_, err := tx.Exec("INSERT OR IGNORE INTO guilds (guild_id) VALUES (?)", guildID)
	return err
}

func (s *DBStore) GetConfig(guildID, configName string, configStruct interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := fmt.Sprintf("SELECT %s FROM guilds WHERE guild_id = ?", configName)
	var configJSON sql.NullString
	err := s.db.QueryRow(query, guildID).Scan(&configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			s.mu.RUnlock()
			s.mu.Lock()
			tx, _ := s.db.Begin()
			if err := s.upsertGuild(tx, guildID); err != nil {
				if err := tx.Rollback(); err != nil {
					// We can't do much if the rollback fails, so we'll just log it.
					fmt.Printf("Failed to rollback transaction: %v", err)
				}
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			s.mu.Unlock()
			s.mu.RLock()
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
	defer func() { _ = tx.Rollback() }()
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

func (s *DBStore) GetNextTicketCounter(guildID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
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

// --- Quiz History ---

// SaveQuizQuestion saves a new quiz question to the history.
func (s *DBStore) SaveQuizQuestion(guildID, topic, question string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT INTO quiz_history (guild_id, topic, question) VALUES (?, ?, ?)", guildID, topic, question)
	return err
}

// GetRecentQuizQuestions retrieves the most recent questions for a given guild and topic.
func (s *DBStore) GetRecentQuizQuestions(guildID, topic string, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT question FROM quiz_history WHERE guild_id = ? AND topic = ? ORDER BY created_at DESC LIMIT ?", guildID, topic, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []string
	for rows.Next() {
		var question string
		if err := rows.Scan(&question); err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	return questions, nil
}

// --- Word Count ---

// IncrementWordCount は指定されたユーザーの単語のカウントを1増やします。
func (s *DBStore) IncrementWordCount(guildID, userID, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := `
		INSERT INTO word_counts (guild_id, user_id, word, count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(guild_id, user_id, word) DO UPDATE SET count = count + 1;
	`
	_, err := s.db.Exec(query, guildID, userID, word)
	return err
}

// GetWordCount は指定されたユーザーの単語のカウントを取得します。
func (s *DBStore) GetWordCount(guildID, userID, word string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int
	query := "SELECT count FROM word_counts WHERE guild_id = ? AND user_id = ? AND word = ?"
	err := s.db.QueryRow(query, guildID, userID, word).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // レコードがない場合は0を返す
		}
		return 0, err
	}
	return count, nil
}

// GetWordCountRanking は指定された単語のサーバー内ランキングを取得します。
func (s *DBStore) GetWordCountRanking(guildID, word string, limit int) ([]WordCount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := `
		SELECT user_id, count
		FROM word_counts
		WHERE guild_id = ? AND word = ?
		ORDER BY count DESC
		LIMIT ?;
	`
	rows, err := s.db.Query(query, guildID, word, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ranking []WordCount
	for rows.Next() {
		var wc WordCount
		wc.Word = word
		if err := rows.Scan(&wc.UserID, &wc.Count); err != nil {
			return nil, err
		}
		ranking = append(ranking, wc)
	}
	return ranking, nil
}

// --- Countable Words ---

// AddCountableWord は、サーバーのカウント対象単語を追加します。
func (s *DBStore) AddCountableWord(guildID, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := "INSERT OR IGNORE INTO countable_words (guild_id, word) VALUES (?, ?)"
	_, err := s.db.Exec(query, guildID, word)
	return err
}

// RemoveCountableWord は、サーバーのカウント対象単語を削除します。
func (s *DBStore) RemoveCountableWord(guildID, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := "DELETE FROM countable_words WHERE guild_id = ? AND word = ?"
	_, err := s.db.Exec(query, guildID, word)
	return err
}

// GetCountableWords は、サーバーのカウント対象単語のリストを取得します。
func (s *DBStore) GetCountableWords(guildID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := "SELECT word FROM countable_words WHERE guild_id = ?"
	rows, err := s.db.Query(query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	return words, nil
}
