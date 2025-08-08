package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// --- Structures ---

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
type WelcomeConfig struct {
	Enabled   bool   `json:"enabled"`
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}

type AutoRoleConfig struct {
	Enabled bool   `json:"enabled"`
	RoleID  string `json:"role_id"`
}

// WordCount はユーザーごとの単語カウントを保持する構造体です
type WordCount struct {
	UserID string
	Word   string
	Count  int
}

// CasinoData holds a user's casino-related information.
type CasinoData struct {
	GuildID         string
	UserID          string
	Chips           int64
	PepeCoinBalance int64
	LastDaily       sql.NullTime // Use sql.NullTime to handle cases where it's not set
}

type Company struct {
	Name                string  `json:"name"`
	Code                string  `json:"code"`
	Description         string  `json:"description"`
	Price               float64 `json:"price"`
	RelatedCategories []string `json:"related_categories"` // JSONとして保存
}

// PortfolioItem represents a single stock holding for a user.
type PortfolioItem struct {
	UserID      string
	CompanyCode string
	Shares      int64
}

// --- DBStore ---

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
			autorole_config TEXT DEFAULT '{}',
			jackpot INTEGER DEFAULT 0,
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
		`CREATE TABLE IF NOT EXISTS casino_data (
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			chips INTEGER DEFAULT 1000,
			pepecoin_balance INTEGER DEFAULT 0,
			last_daily DATETIME,
			PRIMARY KEY (guild_id, user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS companies (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			price REAL NOT NULL,
			related_categories TEXT NOT NULL DEFAULT '[]'
		);`,
		`CREATE TABLE IF NOT EXISTS command_usage (
			category TEXT PRIMARY KEY,
			count INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS stocks_portfolios (
			user_id TEXT NOT NULL,
			company_code TEXT NOT NULL,
			shares INTEGER NOT NULL,
			PRIMARY KEY (user_id, company_code)
		);`,
	}
	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return err
		}
	}
	return nil
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

// --- Config Get/Set ---

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
				tx.Rollback()
				return err
			}
			tx.Commit()
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

// --- Casino Data ---

func (s *DBStore) GetCasinoData(guildID, userID string) (*CasinoData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := &CasinoData{GuildID: guildID, UserID: userID}
	query := "SELECT chips, pepecoin_balance, last_daily FROM casino_data WHERE guild_id = ? AND user_id = ?"
	err := s.db.QueryRow(query, guildID, userID).Scan(&data.Chips, &data.PepeCoinBalance, &data.LastDaily)

	if err != nil {
		if err == sql.ErrNoRows {
			data.Chips = 1000
			data.PepeCoinBalance = 0
			insertQuery := "INSERT INTO casino_data (guild_id, user_id, chips, pepecoin_balance, last_daily) VALUES (?, ?, ?, ?, NULL)"
			_, insertErr := s.db.Exec(insertQuery, guildID, userID, data.Chips, data.PepeCoinBalance)
			if insertErr != nil {
				return nil, insertErr
			}
			return data, nil
		}
		return nil, err
	}

	return data, nil
}

func (s *DBStore) UpdateCasinoData(data *CasinoData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := "UPDATE casino_data SET chips = ?, pepecoin_balance = ?, last_daily = ? WHERE guild_id = ? AND user_id = ?"
	_, err := s.db.Exec(query, data.Chips, data.PepeCoinBalance, data.LastDaily, data.GuildID, data.UserID)
	return err
}

func (s *DBStore) GetChipLeaderboard(guildID string, limit int) ([]CasinoData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := "SELECT user_id, chips FROM casino_data WHERE guild_id = ? ORDER BY chips DESC LIMIT ?"
	rows, err := s.db.Query(query, guildID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []CasinoData
	for rows.Next() {
		var data CasinoData
		data.GuildID = guildID
		if err := rows.Scan(&data.UserID, &data.Chips); err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, data)
	}

	return leaderboard, nil
}

// GetAllUserIDsInCasino returns all user IDs that have casino data for a specific guild.
func (s *DBStore) GetAllUserIDsInCasino(guildID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT user_id FROM casino_data WHERE guild_id = ?", guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}

// --- Stocks ---



// --- Jackpot ---

func (s *DBStore) GetJackpot(guildID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var jackpot int64
	query := "SELECT jackpot FROM guilds WHERE guild_id = ?"
	err := s.db.QueryRow(query, guildID).Scan(&jackpot)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return jackpot, nil
}

func (s *DBStore) UpdateJackpot(guildID string, newJackpot int64) error {
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

	query := "UPDATE guilds SET jackpot = ? WHERE guild_id = ?"
	_, err = tx.Exec(query, newJackpot, guildID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *DBStore) AddToJackpot(guildID string, amount int64) (int64, error) {
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

	updateQuery := "UPDATE guilds SET jackpot = jackpot + ? WHERE guild_id = ?"
	if _, err := tx.Exec(updateQuery, amount, guildID); err != nil {
		return 0, err
	}

	var newJackpot int64
	selectQuery := "SELECT jackpot FROM guilds WHERE guild_id = ?"
	if err := tx.QueryRow(selectQuery, guildID).Scan(&newJackpot); err != nil {
		return 0, err
	}

	return newJackpot, tx.Commit()
}

// --- Word Count ---

func (s *DBStore) IncrementWordCount(guildID, userID, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := `
		INSERT INTO word_counts (guild_id, user_id, word, count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(guild_id, user_id, word) DO UPDATE SET count = count + 1;`
	_, err := s.db.Exec(query, guildID, userID, word)
	return err
}

func (s *DBStore) GetWordCount(guildID, userID, word string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int
	query := "SELECT count FROM word_counts WHERE guild_id = ? AND user_id = ? AND word = ?"
	err := s.db.QueryRow(query, guildID, userID, word).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (s *DBStore) GetWordCountRanking(guildID, word string, limit int) ([]WordCount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := `
		SELECT user_id, count
		FROM word_counts
		WHERE guild_id = ? AND word = ?
		ORDER BY count DESC
		LIMIT ?;`
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

func (s *DBStore) AddCountableWord(guildID, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := "INSERT OR IGNORE INTO countable_words (guild_id, word) VALUES (?, ?)"
	_, err := s.db.Exec(query, guildID, word)
	return err
}

func (s *DBStore) RemoveCountableWord(guildID, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := "DELETE FROM countable_words WHERE guild_id = ? AND word = ?"
	_, err := s.db.Exec(query, guildID, word)
	return err
}

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

// --- Misc ---

func (s *DBStore) CreateMessageCache(messageID, content, authorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT OR REPLACE INTO message_cache (message_id, content, author_id) VALUES (?, ?, ?)", messageID, content, authorID)
	return err
}

func (s *DBStore) GetMessageCache(messageID string) (*CachedMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msg CachedMessage
	err := s.db.QueryRow("SELECT message_id, content, author_id FROM message_cache WHERE message_id = ?", messageID).Scan(&msg.MessageID, &msg.Content, &msg.AuthorID)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec("DELETE FROM message_cache WHERE message_id = ?", messageID)
	if err != nil {
		return &msg, nil
	}

	return &msg, nil
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

func (s *DBStore) SaveQuizQuestion(guildID, topic, question string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT INTO quiz_history (guild_id, topic, question) VALUES (?, ?, ?)", guildID, topic, question)
	return err
}

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

func (s *DBStore) GetRecentMessagesByUser(guildID, userID string, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := "SELECT content FROM message_cache WHERE author_id = ? ORDER BY created_at DESC LIMIT ?"
	rows, err := s.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		messages = append(messages, content)
	}

	return messages, nil
}

// --- Stocks ---

func (s *DBStore) GetUserPortfolio(userID string) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT company_code, shares FROM stocks_portfolios WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	portfolio := make(map[string]int64)
	for rows.Next() {
		var companyCode string
		var shares int64
		if err := rows.Scan(&companyCode, &shares); err != nil {
			return nil, err
		}
		portfolio[companyCode] = shares
	}
	return portfolio, nil
}

func (s *DBStore) UpdateUserPortfolio(userID, companyCode string, shares int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO stocks_portfolios (user_id, company_code, shares)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, company_code) DO UPDATE SET shares = shares + ?;
	`
	_, err := s.db.Exec(query, userID, companyCode, shares, shares)
	return err
}

// GetAllCompanies retrieves all companies from the database.
func (s *DBStore) GetAllCompanies() ([]Company, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT code, name, description, price, related_categories FROM companies")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []Company
	for rows.Next() {
		var c Company
		var categoriesJSON string
		if err := rows.Scan(&c.Code, &c.Name, &c.Description, &c.Price, &categoriesJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(categoriesJSON), &c.RelatedCategories); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, nil
}

// GetCompanyByCode retrieves a single company by its code.
func (s *DBStore) GetCompanyByCode(code string) (*Company, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c Company
	var categoriesJSON string
	query := "SELECT code, name, description, price, related_categories FROM companies WHERE code = ?"
	err := s.db.QueryRow(query, code).Scan(&c.Code, &c.Name, &c.Description, &c.Price, &categoriesJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(categoriesJSON), &c.RelatedCategories); err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateCompanyPrices updates the prices of multiple companies in a single transaction.
func (s *DBStore) UpdateCompanyPrices(prices map[string]float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("UPDATE companies SET price = ? WHERE code = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for code, price := range prices {
		if _, err := stmt.Exec(price, code); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// IncrementCommandUsage increments the usage count for a given command category.
func (s *DBStore) IncrementCommandUsage(category string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO command_usage (category, count)
		VALUES (?, 1)
		ON CONFLICT(category) DO UPDATE SET count = count + 1;
	`
	_, err := s.db.Exec(query, category)
	return err
}

// GetAndResetCommandUsage retrieves all command usage counts and resets them to zero.
func (s *DBStore) GetAndResetCommandUsage() (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query("SELECT category, count FROM command_usage")
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	defer rows.Close()

	usage := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			tx.Rollback()
			return nil, err
		}
		usage[category] = count
	}

	// Reset counts
	_, err = tx.Exec("UPDATE command_usage SET count = 0")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return usage, tx.Commit()
}

// SeedInitialCompanies は、データベースに初期の企業データを投入します。
func (s *DBStore) SeedInitialCompanies() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	initialCompanies := []Company{
		{Name: "カジノ・ロワイヤル", Code: "CSN", Description: "カジノ運営", Price: 150.75, RelatedCategories: []string{"カジノ"}},
		{Name: "AIイマジニアリング", Code: "AIE", Description: "画像生成AIサービス", Price: 320.50, RelatedCategories: []string{"AI"}},
		{Name: "グローバル・トランスポート", Code: "TRN", Description: "翻訳・国際交流支援", Price: 120.00, RelatedCategories: []string{"AI"}},
		{Name: "ペペ・プロダクション", Code: "PPC", Description: "ミームコンテンツ制作", Price: 88.20, RelatedCategories: []string{"Fun"}},
		{Name: "デイリー・サプライ", Code: "DLY", Description: "日々の生活支援", Price: 95.60, RelatedCategories: []string{"カジノ", "ユーティリティ"}},
		{Name: "Lunaインフラストラクチャ", Code: "LNA", Description: "Bot自身の運営", Price: 500.00, RelatedCategories: []string{"ユーティリティ", "管理"}},
		{Name: "ギーク・トイズ", Code: "GKT", Description: "ツール・計算機開発", Price: 75.00, RelatedCategories: []string{"ユーティリティ", "ポケモン", "ツール"}},
		{Name: "ミューズ・エンタテインメント", Code: "MUS", Description: "音楽配信サービス", Price: 180.00, RelatedCategories: []string{"音楽"}},
		{Name: "アシスタント・ギルド", Code: "ASG", Description: "AIアシスタント・Q&A", Price: 250.00, RelatedCategories: []string{"AI"}},
		{Name: "セキュリティ・ソリューションズ", Code: "SCS", Description: "サーバー管理・セキュリティ", Price: 220.00, RelatedCategories: []string{"管理", "ユーティリティ"}},
		{Name: "サンダーリーグ", Code: "WTR", Description: "War Thunder関連ツール", Price: 60.00, RelatedCategories: []string{"War Thunder"}},
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO companies (code, name, description, price, related_categories) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, company := range initialCompanies {
		categoriesJSON, _ := json.Marshal(company.RelatedCategories)
		_, err := stmt.Exec(company.Code, company.Name, company.Description, company.Price, string(categoriesJSON))
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
