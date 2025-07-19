package commands

import (
	"fmt"
	"luna/interfaces"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// WordRankingCommand は単語のランキングを扱うコマンドです。
type WordRankingCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *WordRankingCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "wordranking",
		Description: "指定した単語の発言回数ランキングを表示します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "word",
				Description: "ランキングを調べたい単語",
				Required:    true,
			},
		},
	}
}

func (c *WordRankingCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	word := options[0].StringValue()

	// ランキングデータを取得 (上位10件)
	ranking, err := c.Store.GetWordCountRanking(i.GuildID, word, 10)
	if err != nil {
		c.Log.Error("Failed to get word count ranking", "error", err, "guildID", i.GuildID, "word", word)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラーが発生しました。ランキングを取得できませんでした。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			c.Log.Error("Failed to send error response", "error", err)
		}
		return
	}

	if len(ranking) == 0 {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("まだ誰も「%s」と言っていないようです！", word),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			c.Log.Error("Failed to send no data response", "error", err)
		}
		return
	}

	// ランキングを見やすい文字列に整形
	var description strings.Builder
	for idx, item := range ranking {
		var medal string
		switch idx {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("%2d.", idx+1)
		}
		description.WriteString(fmt.Sprintf("%s <@%s> **%d** 回\n", medal, item.UserID, item.Count))
	}

	// 見やすいEmbedを作成
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("「%s」発言回数ランキング", word),
		Description: description.String(),
		Color:       0xffd700, // 金色っぽい色
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.imgur.com/C1zH4iH.png", // ランキングっぽいアイコン
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("サーバー「%s」のランキング", i.GuildID),
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

func (c *WordRankingCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *WordRankingCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *WordRankingCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *WordRankingCommand) GetCategory() string                                                  { return "Fun" }