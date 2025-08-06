package commands

import (
	"fmt"
	"luna/interfaces"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// WordConfigCommand はカウント対象単語の管理を行うコマンドです。
type WordConfigCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *WordConfigCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "wordconfig",
		Description:              "カウント対象の単語を管理します。",
		DefaultMemberPermissions: &[]int64{int64(discordgo.PermissionManageGuild)}[0],
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "add",
				Description: "新しい単語をカウント対象に追加します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "word",
						Description: "追加する単語",
						Required:    true,
					},
				},
			},
			{
				Name:        "remove",
				Description: "単語をカウント対象から削除します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "word",
						Description: "削除する単語",
						Required:    true,
					},
				},
			},
			{
				Name:        "list",
				Description: "カウント対象の単語一覧を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *WordConfigCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "add":
		c.handleAdd(s, i)
	case "remove":
		c.handleRemove(s, i)
	case "list":
		c.handleList(s, i)
	}
}

func (c *WordConfigCommand) handleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	word := i.ApplicationCommandData().Options[0].Options[0].StringValue()
	err := c.Store.AddCountableWord(i.GuildID, word)
	if err != nil {
		c.Log.Error("Failed to add countable word", "error", err)
		sendErrorResponse(s, i, "単語の追加に失敗しました。）")
		return
	}
	embed := &discordgo.MessageEmbed{
		Title: "✅ 単語を追加しました",
		Description: fmt.Sprintf("「%s」をカウント対象に追加しました。", word),
		Color: 0x77b255, // Green
	}
	sendEmbedResponse(s, i, embed)
}

func (c *WordConfigCommand) handleRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	word := i.ApplicationCommandData().Options[0].Options[0].StringValue()
	err := c.Store.RemoveCountableWord(i.GuildID, word)
	if err != nil {
		c.Log.Error("Failed to remove countable word", "error", err)
		sendErrorResponse(s, i, "単語の削除に失敗しました。）")
		return
	}
	embed := &discordgo.MessageEmbed{
		Title: "🗑️ 単語を削除しました",
		Description: fmt.Sprintf("「%s」をカウント対象から削除しました。", word),
		Color: 0xe74c3c, // Red
	}
	sendEmbedResponse(s, i, embed)
}

func (c *WordConfigCommand) handleList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	words, err := c.Store.GetCountableWords(i.GuildID)
	if err != nil {
		c.Log.Error("Failed to get countable words", "error", err)
		sendErrorResponse(s, i, "単語リストの取得に失敗しました。）")
		return
	}

	var description string
	if len(words) == 0 {
		description = "現在、カウント対象に設定されている単語はありません。"
	} else {
		description = "- " + strings.Join(words, "\n- ")
	}

	embed := &discordgo.MessageEmbed{
		Title: "📊 カウント対象の単語一覧",
		Description: description,
		Color: 0x3498db, // Blue
	}
	sendEmbedResponse(s, i, embed)
}



func (c *WordConfigCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *WordConfigCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *WordConfigCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *WordConfigCommand) GetCategory() string                                                  { return "管理" }
