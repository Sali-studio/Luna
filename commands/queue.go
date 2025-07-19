package commands

import (
	"fmt"
	"luna/interfaces"
	"strings"
	"luna/player"

	"github.com/bwmarrin/discordgo"
)

// QueueCommand は再生キューを表示するコマンドです。
type QueueCommand struct {
	Player interfaces.MusicPlayer
	Log    interfaces.Logger
}

func (c *QueueCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "queue",
		Description: "現在の再生キューを表示します。",
	}
}

func (c *QueueCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	queue := c.Player.GetQueue(i.GuildID)

	var embed *discordgo.MessageEmbed
	if len(queue) == 0 {
		embed = &discordgo.MessageEmbed{
			Title:       "🎶 再生キュー",
			Description: "現在、再生キューに曲はありません。",
			Color:       0x95a5a6, // Gray
		}
	} else {
		var queueList strings.Builder
		for idx, song := range queue {
			queueList.WriteString(fmt.Sprintf("%d. **[%s](%s)** - %s\n", idx+1, song.Title, song.URL, song.Author))
		}

		embed = &discordgo.MessageEmbed{
			Title:       "🎶 再生キュー",
			Description: queueList.String(),
			Color:       0x3498db, // Blue
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("合計 %d 曲", len(queue)),
			},
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *QueueCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *QueueCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *QueueCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *QueueCommand) GetCategory() string                                                  { return "音楽" }