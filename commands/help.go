// commands/help.go
package commands

import (
	"fmt"
	"luna/interfaces"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct {
	AllCommands map[string]interfaces.CommandHandler
}

func (c *HelpCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Botのコマンド一覧を表示します",
	}
}

func (c *HelpCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	categorizedCommands := make(map[string][]string)
	for _, cmdHandler := range c.AllCommands {
		def := cmdHandler.GetCommandDef()
		category := cmdHandler.GetCategory()
		if category == "" {
			category = "その他"
		}
		commandInfo := fmt.Sprintf("`/%s` - %s", def.Name, def.Description)
		categorizedCommands[category] = append(categorizedCommands[category], commandInfo)
	}

	categories := make([]string, 0, len(categorizedCommands))
	for k := range categorizedCommands {
		categories = append(categories, k)
	}
	sort.Strings(categories)

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

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	}); err != nil {
		// c.Log is not available in HelpCommand. We can't do much more.
	}
}

func (c *HelpCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *HelpCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *HelpCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *HelpCommand) GetCategory() string                                                  { return "ユーティリティ" }
