package commands

import (
	"fmt"
	"luna/handlers"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct {
	// Botに登録されている全てのコマンドを保持するためのマップ
	AllCommands map[string]handlers.CommandHandler
}

func (c *HelpCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Botのコマンド一覧を表示します",
	}
}

func (c *HelpCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// カテゴリごとにコマンドを分類
	categorizedCommands := make(map[string][]string)
	for _, cmdHandler := range c.AllCommands {
		def := cmdHandler.GetCommandDef()
		category := cmdHandler.GetCategory()
		if category == "" {
			category = "その他" // カテゴリ未設定のコマンド
		}
		commandInfo := fmt.Sprintf("`/%s` - %s", def.Name, def.Description)
		categorizedCommands[category] = append(categorizedCommands[category], commandInfo)
	}

	// カテゴリ名をソートして、表示順を固定
	categories := make([]string, 0, len(categorizedCommands))
	for k := range categorizedCommands {
		categories = append(categories, k)
	}
	sort.Strings(categories)

	// Embedを作成
	embed := &discordgo.MessageEmbed{
		Title:       "Luna Bot コマンド一覧",
		Description: "利用可能なコマンドは以下の通りです。",
		Color:       0x7289da,
		Fields:      []*discordgo.MessageEmbedField{},
	}

	for _, category := range categories {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("📂 %s", category),
			Value: strings.Join(categorizedCommands[category], "\n"),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *HelpCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *HelpCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *HelpCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *HelpCommand) GetCategory() string                                                  { return "ユーティリティ" }
