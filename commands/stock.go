package commands

import (
	"fmt"
	"luna/interfaces"

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
	{Name: "AIイマジニアリング", Code: "AIE", Description: "画像生成AIサービス", Price: 320.50},
	{Name: "ペペ・プロダクション", Code: "PPC", Description: "ミームコンテンツ制作", Price: 88.20},
	{Name: "グローバル・トランスポート", Code: "TRN", Description: "翻訳・国際交流支援", Price: 120.00},
	{Name: "デイリー・サプライ", Code: "DLY", Description: "日々の生活支援", Price: 95.60},
	{Name: "Lunaインフラストラクチャ", Code: "LNA", Description: "Bot自身の運営", Price: 500.00},
}

// StockCommand handles the /stock command.
type StockCommand struct {
	Store     interfaces.DataStore
	Log       interfaces.Logger
	Companies []Company // For now, we use an in-memory list
}

// NewStockCommand creates a new StockCommand.
func NewStockCommand(store interfaces.DataStore, log interfaces.Logger) *StockCommand {
	return &StockCommand{
		Store:     store,
		Log:       log,
		Companies: initialCompanies,
	}
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
		},
	}
}

func (c *StockCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Options[0].Name {
	case "list":
		c.handleList(s, i)
	}
}

func (c *StockCommand) handleList(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

func (c *StockCommand) GetCategory() string {
	return "経済"
}

func (c *StockCommand) GetComponentIDs() []string {
	return nil
}

func (c *StockCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}

func (c *StockCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
