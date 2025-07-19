package commands

import (
	"fmt"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// SkipCommand は音楽をスキップするコマンドです。
type SkipCommand struct {
	Player interfaces.MusicPlayer
	Log    interfaces.Logger
}

func (c *SkipCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "skip",
		Description: "現在再生中の音楽をスキップし、キューの次の曲を再生します。",
	}
}

func (c *SkipCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.Player.Skip(i.GuildID)

	// 次の曲があるか確認
	queue := c.Player.GetQueue(i.GuildID)
	var embed *discordgo.MessageEmbed
	if len(queue) > 0 {
		nextSong := queue[0] // スキップ後の次の曲
		embed = &discordgo.MessageEmbed{
			Title:       "⏭️ スキップしました",
			Description: fmt.Sprintf("次の曲: **[%s](%s)**", nextSong.Title, nextSong.URL),
			Color:       0xf1c40f, // Yellow
		}
	} else {
		embed = &discordgo.MessageEmbed{
			Title:       "⏭️ スキップしました",
			Description: "キューに次の曲はありません。",
			Color:       0xf1c40f, // Yellow
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *SkipCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *SkipCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *SkipCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *SkipCommand) GetCategory() string                                                  { return "音楽" }
