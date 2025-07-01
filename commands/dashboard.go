package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

type DashboardCommand struct {
	Store *storage.ConfigStore
}

func (c *DashboardCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "dashboard",
		Description:              "サーバーの統計情報を表示するダッシュボードを設置します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
}

func (c *DashboardCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		guild, err = s.Guild(i.GuildID) // StateになければAPIから取得
		if err != nil {
			logger.Error.Printf("ダッシュボード用のサーバー情報取得に失敗: %v", err)
			return
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 %s のダッシュボード", guild.Name),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "メンバー数", Value: fmt.Sprintf("%d人", guild.MemberCount), Inline: true},
			{Name: "オンライン", Value: "更新中...", Inline: true}, // オンライン数は動的に更新する必要がある
			{Name: "ブーストレベル", Value: fmt.Sprintf("Level %d", guild.PremiumTier), Inline: true},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: guild.IconURL()},
	}

	msg, err := s.ChannelMessageSendEmbed(i.ChannelID, embed)
	if err != nil {
		logger.Error.Printf("ダッシュボードの送信に失敗: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ ダッシュボードの作成に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// 作成したメッセージのIDなどを保存
	config := c.Store.GetGuildConfig(i.GuildID)
	config.Dashboard.ChannelID = msg.ChannelID
	config.Dashboard.MessageID = msg.ID
	c.Store.Save()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "✅ ダッシュボードを作成しました。", Flags: discordgo.MessageFlagsEphemeral},
	})

	// TODO: 定期的にダッシュボードを更新する処理をスケジューラに追加する
}

func (c *DashboardCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *DashboardCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *DashboardCommand) GetComponentIDs() []string                                            { return []string{} }
