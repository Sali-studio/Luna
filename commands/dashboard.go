package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type DashboardCommand struct {
	Store     *storage.ConfigStore
	Scheduler *cron.Cron
}

func (c *DashboardCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "dashboard-setup",
		Description:              "サーバーの統計情報を表示するダッシュボードを設置します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
}

func (c *DashboardCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})

	msg, err := s.ChannelMessageSendEmbed(i.ChannelID, &discordgo.MessageEmbed{
		Title:       "📊 ダッシュボード",
		Description: "統計情報を更新中...",
	})
	if err != nil {
		logger.Error("ダッシュボードの初期送信に失敗", "error", err)
		content := "❌ ダッシュボードの作成に失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	config := c.Store.GetGuildConfig(i.GuildID)
	config.Dashboard.ChannelID = msg.ChannelID
	config.Dashboard.MessageID = msg.ID
	c.Store.Save()

	c.Scheduler.AddFunc("@every 5m", func() { c.updateDashboard(s, i.GuildID) })
	c.updateDashboard(s, i.GuildID)

	content := "✅ ダッシュボードを作成し、5分ごとの自動更新をセットしました。"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
}

func (c *DashboardCommand) updateDashboard(s *discordgo.Session, guildID string) {
	config := c.Store.GetGuildConfig(guildID)
	if config.Dashboard.ChannelID == "" || config.Dashboard.MessageID == "" {
		return
	}

	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			logger.Error("ダッシュボード更新用のサーバー情報取得に失敗", "error", err, "guildID", guildID)
			return
		}
	}

	onlineMembers := 0
	for _, pres := range guild.Presences {
		if pres.Status != discordgo.StatusOffline {
			onlineMembers++
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 %s のダッシュボード", guild.Name),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "メンバー数", Value: fmt.Sprintf("%d人", guild.MemberCount), Inline: true},
			{Name: "オンライン", Value: fmt.Sprintf("%d人", onlineMembers), Inline: true},
			{Name: "ブースト", Value: fmt.Sprintf("Level %d (%d Boosts)", guild.PremiumTier, guild.PremiumSubscriptionCount), Inline: true},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: guild.IconURL("")},
		Footer:    &discordgo.MessageEmbedFooter{Text: "最終更新"},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageEditEmbed(config.Dashboard.ChannelID, config.Dashboard.MessageID, embed)
}

func (c *DashboardCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *DashboardCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *DashboardCommand) GetComponentIDs() []string                                            { return []string{} }
