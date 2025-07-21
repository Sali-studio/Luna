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
	// 最初に遅延応答を送信し、「考え中...」のような状態を示す
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		c.Log.Error("Failed to send deferred response", "error", err)
		return
	}

	// ユーザーがボイスチャンネルにいるか確認
	vs, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
	if err != nil || vs == nil || vs.ChannelID == "" {
		content := "ボイスチャンネルに参加してからコマンドを実行してください。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// ボイスチャンネルに接続
	err = c.Player.JoinVC(i.GuildID, vs.ChannelID)
	if err != nil {
		c.Log.Error("Failed to join voice channel", "error", err, "guildID", i.GuildID, "channelID", vs.ChannelID)
		content := fmt.Sprintf("ボイスチャンネルへの接続に失敗しました: %s", err.Error())
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// 成功メッセージをEmbedで送信
	embed := &discordgo.MessageEmbed{
		Title:       "✅ 接続しました！",
		Description: fmt.Sprintf("<#%s> に接続しました。", vs.ChannelID),
		Color:       0x2ecc71, // Green
	}
	// 遅延応答を編集して最終的なメッセージを送信
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *JoinCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *JoinCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *JoinCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *JoinCommand) GetCategory() string                                                  { return "音楽" }