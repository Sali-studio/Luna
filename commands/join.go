package commands

import (
	"fmt"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// JoinCommand はボットをボイスチャンネルに接続させるコマンドです。
type JoinCommand struct {
	Player interfaces.MusicPlayer
	Log    interfaces.Logger
}

func (c *JoinCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "join",
		Description: "ボットをあなたのいるボイスチャンネルに接続させます。",
	}
}

func (c *JoinCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ユーザーがボイスチャンネルにいるか確認
	vs := i.Member.VoiceState
	if vs == nil || vs.ChannelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ボイスチャンネルに参加してからコマンドを実行してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// ボイスチャンネルに接続
	err := c.Player.JoinVC(i.GuildID, vs.ChannelID)
	if err != nil {
		c.Log.Error("Failed to join voice channel", "error", err, "guildID", i.GuildID, "channelID", vs.ChannelID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("ボイスチャンネルへの接続に失敗しました: %s", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 成功メッセージをEmbedで送信
	embed := &discordgo.MessageEmbed{
		Title:       "✅ 接続しました！",
		Description: fmt.Sprintf("<#%s> に接続しました。", vs.ChannelID),
		Color:       0x2ecc71, // Green
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *JoinCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *JoinCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *JoinCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *JoinCommand) GetCategory() string                                                  { return "音楽" }
