package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"
	"math/rand"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// Company represents a company in the stock market.
type Company struct {
	Name        string
	Code        string
	Description string
	Price       float64
}

// TODO: Move this to a database later
var initialCompanies = []Company{
	{Name: "カジノ・ロワイヤル", Code: "CSN", Description: "カジノ運営", Price: 150.75},
	{Name: "AIアート", Code: "AIE", Description: "画像生成AIサービス", Price: 320.50},
	{Name: "ペペ・プロダクション", Code: "PPC", Description: "ミームコンテンツ制作", Price: 88.20},
	{Name: "グローバル・トランスポート", Code: "TRN", Description: "翻訳・国際交流支援", Price: 120.00},
	{Name: "デイリー・サプライ", Code: "DLY", Description: "日々の生活支援", Price: 95.60},
	{Name: "Lunaインフラストラクチャ", Code: "LNA", Description: "Bot自身の運営", Price: 500.00},
}

// StockCommand handles the /stock command.
type StockCommand struct {
	Store     interfaces.DataStore
	Log       interfaces.Logger
	Companies []storage.Company // Now uses the struct from storage
	mu        sync.RWMutex
}

// NewStockCommand creates a new StockCommand.
func NewStockCommand(store interfaces.DataStore, log interfaces.Logger) *StockCommand {
	sc := &StockCommand{
		Store: store,
		Log:   log,
	}
	go sc.loadInitialCompanies()
	return sc
}

func (c *StockCommand) loadInitialCompanies() {
	companies, err := c.Store.GetAllCompanies()
	if err != nil {
		c.Log.Error("Failed to load companies from DB", "error", err)
		return
	}
	c.mu.Lock()
	c.Companies = companies
	c.mu.Unlock()
	c.Log.Info("Successfully loaded companies from DB", "count", len(companies))
}

func (c *StockCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "stock",
		Description: "株式市場関連のコマンド",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "list",
				Description: "上場企業の株価一覧を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "buy",
				Description: "指定した企業の株を購入します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "code", Description: "銘柄コード (例: CSN)", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "購入する株数", Required: true, MinValue: &[]float64{1}[0]},
				},
			},
			{
				Name:        "sell",
				Description: "保有している株を売却します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "code", Description: "銘柄コード (例: CSN)", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "売却する株数", Required: true, MinValue: &[]float64{1}[0]},
				},
			},
			{
				Name:        "portfolio",
				Description: "あなたのポートフォリオ（資産状況）を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "確認したいユーザー（任意）", Required: false},
				},
			},
			{
				Name:        "info",
				Description: "企業の詳細情報を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "code", Description: "銘柄コード (例: CSN)", Required: true},
				},
			},
			{
				Name:        "leaderboard",
				Description: "株式資産を含めたサーバー内の資産家ランキングを表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *StockCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Options[0].Name {
	case "list":
		c.handleList(s, i)
	case "buy":
		c.handleBuy(s, i)
	case "sell":
		c.handleSell(s, i)
	case "portfolio":
		c.handlePortfolio(s, i)
	case "info":
		c.handleInfo(s, i)
	case "leaderboard":
		c.handleLeaderboard(s, i)
	}
}

func (c *StockCommand) handleList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	embed := &discordgo.MessageEmbed{
		Title:       "📈 株式市場",
		Description: "現在の上場企業一覧です。",
		Color:       0x2ecc71, // Green
	}

	for _, company := range c.Companies {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("**%s (%s)**", company.Name, company.Code),
			Value:  fmt.Sprintf("```\n現在価格: %.2f PPC\n```\n*事業内容: %s*", company.Price, company.Description),
			Inline: false,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *StockCommand) handleBuy(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	code := strings.ToUpper(options[0].StringValue())
	amount := options[1].IntValue()
	userID := i.Member.User.ID
	guildID := i.GuildID

	company, exists := c.findCompanyByCode(code)
	if !exists {
		sendErrorResponse(s, i, "指定された銘柄コードの企業は存在しません。")
		return
	}

	totalCost := int64(company.Price * float64(amount))

	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for stock buy", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if casinoData.PepeCoinBalance < totalCost {
		sendErrorResponse(s, i, fmt.Sprintf("PepeCoinが足りません！\n購入に必要なPPC: `%d`\n現在のPPC: `%d`", totalCost, casinoData.PepeCoinBalance))
		return
	}

	// Perform transaction
	casinoData.PepeCoinBalance -= totalCost
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data for stock buy", "error", err)
		sendErrorResponse(s, i, "購入処理中にエラーが発生しました。")
		return
	}

	if err := c.Store.UpdateUserPortfolio(userID, code, amount); err != nil {
		c.Log.Error("Failed to update portfolio for stock buy", "error", err)
		// TODO: Add logic to refund PepeCoin if this fails
		sendErrorResponse(s, i, "ポートフォリオの更新中にエラーが発生しました。")
		return
	}

	sendSuccessResponse(s, i, fmt.Sprintf("**%s (%s)** の株を **%d** 株、**%d** PPC で購入しました。", company.Name, company.Code, amount, totalCost))
}

func (c *StockCommand) handleSell(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	code := strings.ToUpper(options[0].StringValue())
	amountToSell := options[1].IntValue()
	userID := i.Member.User.ID
	guildID := i.GuildID

	company, exists := c.findCompanyByCode(code)
	if !exists {
		sendErrorResponse(s, i, "指定された銘柄コードの企業は存在しません。")
		return
	}

	portfolio, err := c.Store.GetUserPortfolio(userID)
	if err != nil {
		c.Log.Error("Failed to get user portfolio for stock sell", "error", err)
		sendErrorResponse(s, i, "ポートフォリオの取得中にエラーが発生しました。")
		return
	}

	currentShares, ok := portfolio[code]
	if !ok || currentShares < amountToSell {
		sendErrorResponse(s, i, fmt.Sprintf("保有株数が足りません。\n銘柄: %s\n保有数: %d", code, currentShares))
		return
	}

	totalProceeds := int64(company.Price * float64(amountToSell))

	// Perform transaction
	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for stock sell", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	casinoData.PepeCoinBalance += totalProceeds
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data for stock sell", "error", err)
		sendErrorResponse(s, i, "売却処理中にエラーが発生しました。")
		return
	}

	if err := c.Store.UpdateUserPortfolio(userID, code, -amountToSell); err != nil {
		c.Log.Error("Failed to update portfolio for stock sell", "error", err)
		// Attempt to revert the balance change
		casinoData.PepeCoinBalance -= totalProceeds
		c.Store.UpdateCasinoData(casinoData)
		sendErrorResponse(s, i, "ポートフォリオの更新中にエラーが発生しました。")
		return
	}

	sendSuccessResponse(s, i, fmt.Sprintf("**%s (%s)** の株を **%d** 株、**%d** PPC で売却しました。", company.Name, company.Code, amountToSell, totalProceeds))
}

func (c *StockCommand) findCompanyByCode(code string) (*storage.Company, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, company := range c.Companies {
		if company.Code == code {
			return &company, true
		}
	}
	return nil, false
}

// UpdateStockPrices は、コマンド利用状況に基づいて株価を更新します。
func (c *StockCommand) UpdateStockPrices() {
	usage, err := c.Store.GetAndResetCommandUsage()
	if err != nil {
		c.Log.Error("Failed to get command usage", "error", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	newPrices := make(map[string]float64)
	for i, company := range c.Companies {
		activityFactor := 0.0
		for _, category := range company.RelatedCategories {
			activityFactor += float64(usage[category])
		}

		// アクティビティに基づいて価格変動率を計算
		// 基本変動率 + アクティビティによる変動
		baseChange := (rand.Float64() - 0.5) * 0.02 // -1% to +1%
		activityChange := activityFactor * 0.001    // 1 usage = +0.1% change
		decay := -0.005                             // 何も使われないと少しずつ下がる

		changePercent := baseChange + activityChange + decay
		newPrice := company.Price * (1 + changePercent)

		// 価格が極端になりすぎないように制限
		if newPrice < 1.0 {
			newPrice = 1.0
		}

		c.Companies[i].Price = newPrice
		newPrices[company.Code] = newPrice
	}

	if err := c.Store.UpdateCompanyPrices(newPrices); err != nil {
		c.Log.Error("Failed to update company prices in DB", "error", err)
		return
	}

	c.Log.Info("Stock prices updated based on command usage", "usage_data", usage)
}

func (c *StockCommand) handlePortfolio(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	var targetUser *discordgo.User
	if len(options) > 0 {
		targetUser = options[0].UserValue(s)
	} else {
		targetUser = i.Member.User
	}

	portfolio, err := c.Store.GetUserPortfolio(targetUser.ID)
	if err != nil {
		c.Log.Error("Failed to get user portfolio", "error", err)
		sendErrorResponse(s, i, "ポートフォリオの取得に失敗しました。")
		return
	}

	casinoData, err := c.Store.GetCasinoData(i.GuildID, targetUser.ID)
	if err != nil {
		c.Log.Error("Failed to get casino data for portfolio", "error", err)
		sendErrorResponse(s, i, "ユーザー情報の取得に失敗しました。")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("💼 %s のポートフォリオ", targetUser.Username),
		Color: 0x3498db, // Blue
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL(""),
		},
	}

	var totalStockValue float64
	var stockDetails strings.Builder

	if len(portfolio) == 0 {
		stockDetails.WriteString("現在、株式を保有していません。")
	} else {
		for code, shares := range portfolio {
			company, exists := c.findCompanyByCode(code)
			if !exists {
				continue // Should not happen if data is consistent
			}
			currentValue := company.Price * float64(shares)
			totalStockValue += currentValue
			stockDetails.WriteString(fmt.Sprintf("**%s (%s)**\n", company.Name, company.Code))
			stockDetails.WriteString(fmt.Sprintf("保有数: `%d`株\n評価額: `%.2f` PPC\n\n", shares, currentValue))
		}
	}

	totalAssets := totalStockValue + float64(casinoData.PepeCoinBalance)

	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:  "💰 総資産",
			Value: fmt.Sprintf("**`%.2f` PPC**", totalAssets),
		},
		{
			Name:   "保有株式",
			Value:  fmt.Sprintf("評価額合計: `%.2f` PPC", totalStockValue),
			Inline: true,
		},
		{
			Name:   "現金",
			Value:  fmt.Sprintf("`%d` PPC", casinoData.PepeCoinBalance),
			Inline: true,
		},
		{
			Name:  "保有銘柄一覧",
			Value: stockDetails.String(),
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *StockCommand) handleInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	code := strings.ToUpper(i.ApplicationCommandData().Options[0].Options[0].StringValue())

	company, exists := c.findCompanyByCode(code)
	if !exists {
		sendErrorResponse(s, i, "指定された銘柄コードの企業は存在しません。")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🏢 %s (%s)", company.Name, company.Code),
		Description: company.Description,
		Color:       0x1abc9c, // Turquoise
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "現在価格",
				Value:  fmt.Sprintf("**`%.2f` PPC**", company.Price),
				Inline: true,
			},
			{
				Name:   "関連カテゴリ",
				Value:  strings.Join(company.RelatedCategories, ", "),
				Inline: true,
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *StockCommand) handleLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// This is a complex operation, might be slow on large servers.
	// For now, we'll just show a placeholder.
	sendErrorResponse(s, i, "この機能は現在開発中です。")
}


func (c *StockCommand) GetCategory() string {
	return "経済"
}

func (c *StockCommand) GetComponentIDs() []string {
	return nil
}

func (c *StockCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}

func (c *StockCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
