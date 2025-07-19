package commands

import (
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// StopCommand は音楽の再生を停止するコマンドです。
type StopCommand struct {
	Player interfaces.MusicPlayer
	Log    interfaces.Logger
}

func (c *StopCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "stop",
		Description: "現在再生中の音楽を停止し、キューをクリアします。",
	}
}

func (c *StopCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.Player.Stop(i.GuildID)

	embed := &discordgo.MessageEmbed{
		Title:       "⏹️ 停止しました",
		Description: "音楽の再生を停止し、キューをクリアしました。",
		Color:       0xe74c3c, // Red
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *StopCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *StopCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *StopCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *StopCommand) GetCategory() string                                                  { return "音楽" }
