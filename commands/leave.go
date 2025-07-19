package commands

import (
	"luna/interfaces"
	"luna/player"

	"github.com/bwmarrin/discordgo"
)

// LeaveCommand はボットをボイスチャンネルから切断するコマンドです。
type LeaveCommand struct {
	Player interfaces.MusicPlayer
	Log    interfaces.Logger
}

func (c *LeaveCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "leave",
		Description: "ボットを現在接続しているボイスチャンネルから切断します。",
	}
}

func (c *LeaveCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ボットがボイスチャンネルに接続しているか確認
	gp := c.Player.GetGuildPlayer(i.GuildID)
	if gp == nil || gp.(*player.GuildPlayer).VoiceConnection == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ボットはボイスチャンネルに接続していません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	c.Player.LeaveVC(i.GuildID)

	embed := &discordgo.MessageEmbed{
		Title:       "👋 切断しました",
		Description: "ボイスチャンネルから切断しました。",
		Color:       0x95a5a6, // Gray
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *LeaveCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *LeaveCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *LeaveCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *LeaveCommand) GetCategory() string                                                  { return "音楽" }
