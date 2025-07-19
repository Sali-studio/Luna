package commands

import (
	"fmt"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// WordCountCommand は単語のカウントを扱うコマンドです。
type WordCountCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *WordCountCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "wordcount",
		Description: "指定した単語のあなたの発言回数を表示します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "word",
				Description: "カウントを調べたい単語",
				Required:    true,
			},
		},
	}
}

func (c *WordCountCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	word := options[0].StringValue()

	// ユーザーの単語カウントを取得
	count, err := c.Store.GetWordCount(i.GuildID, i.Member.User.ID, word)
	if err != nil {
		c.Log.Error("Failed to get word count", "error", err, "guildID", i.GuildID, "userID", i.Member.User.ID, "word", word)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラーが発生しました。カウントを取得できませんでした。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			c.Log.Error("Failed to send error response", "error", err)
		}
		return
	}

	// 見やすいEmbedを作成
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("「%s」カウント", word),
		Description: fmt.Sprintf("<@%s> さんの現在の「%s」発言回数です。", i.Member.User.ID, word),
		Color:       0x77b255, // Discordの緑色っぽい色
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "回数",
				Value: fmt.Sprintf("**%d** 回", count),
			},
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.Username,
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	// インタラクションにEmbedで応答
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		c.Log.Error("Failed to send embed response", "error", err)
	}
}

func (c *WordCountCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *WordCountCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *WordCountCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *WordCountCommand) GetCategory() string                                                  { return "Fun" }
