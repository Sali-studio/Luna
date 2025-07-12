package commands

import (
	"fmt"
	"luna/bot"
	"luna/logger"
	"luna/storage"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	DashboardShowInfoButtonID  = "dashboard_show_info"
	DashboardShowRolesButtonID = "dashboard_show_roles"
)

type DashboardCommand struct {
	Store     bot.DataStore
	Scheduler bot.Scheduler
	Log       logger.Logger
}

func (c *DashboardCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "dashboard-setup",
		Description:              "インタラクティブな高機能ダッシュボードを設置します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
}

func (c *DashboardCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})

	msg, err := s.ChannelMessageSendEmbed(i.ChannelID, &discordgo.MessageEmbed{
		Title: "📊 ダッシュボード", Description: "統計情報を収集中...", Color: 0x3498db,
	})
	if err != nil {
		c.Log.Error("ダッシュボードの初期送信に失敗", "error", err)
		content := "❌ ダッシュボードの作成に失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	var config storage.DashboardConfig
	config.ChannelID = msg.ChannelID
	config.MessageID = msg.ID
	if err := c.Store.SaveConfig(i.GuildID, "dashboard_config", config); err != nil {
		c.Log.Error("ダッシュボード設定の保存に失敗", "error", err, "guildID", i.GuildID)
		return
	}

	c.Scheduler.AddFunc("@hourly", func() { c.updateDashboard(s, i.GuildID) })
	c.updateDashboard(s, i.GuildID)

	content := "✅ ダッシュボードを作成し、1時間ごとの自動更新をセットしました。"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
}

func (c *DashboardCommand) updateDashboard(s *discordgo.Session, guildID string) {
	var config storage.DashboardConfig
	if err := c.Store.GetConfig(guildID, "dashboard_config", &config); err != nil || config.ChannelID == "" || config.MessageID == "" {
		return
	}

	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			return
		}
	}

	memberCount := guild.MemberCount
	botCount := 0
	for _, member := range guild.Members {
		if member.User.Bot {
			botCount++
		}
	}
	humanCount := memberCount - botCount
	onlineMembers := 0
	for _, pres := range guild.Presences {
		if pres.Status != discordgo.StatusOffline {
			onlineMembers++
		}
	}
	textChannelCount, voiceChannelCount, categoryCount := 0, 0, 0
	for _, ch := range guild.Channels {
		switch ch.Type {
		case discordgo.ChannelTypeGuildText:
			textChannelCount++
		case discordgo.ChannelTypeGuildVoice:
			voiceChannelCount++
		case discordgo.ChannelTypeGuildCategory:
			categoryCount++
		}
	}
	roleCount, emojiCount := len(guild.Roles), len(guild.Emojis)
	guildIDInt, _ := discordgo.SnowflakeTimestamp(guild.ID)

	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("📊 %s のサーバーダッシュボード", guild.Name),
		Color:     0x7289da,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: guild.IconURL("")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "👥 メンバー", Value: fmt.Sprintf("```ini\n[ Total ] %d\n[ Human ] %d\n[ Bot ] %d\n[ Online ] %d\n```", memberCount, humanCount, botCount, onlineMembers), Inline: true},
			{Name: "📁 コンテンツ", Value: fmt.Sprintf("```ini\n[ Category ] %d\n[ Text ch ] %d\n[ Voice ch ] %d\n[ Roles ] %d\n[ Emojis ] %d\n```", categoryCount, textChannelCount, voiceChannelCount, roleCount, emojiCount), Inline: true},
			{Name: "💎 ブースト", Value: fmt.Sprintf("```ini\n[ Level ] %d\n[ Boosts ] %d\n```", guild.PremiumTier, guild.PremiumSubscriptionCount), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("サーバー作成日: %s", guildIDInt.Format("2006/01/02")),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "詳細情報", Style: discordgo.SecondaryButton, CustomID: DashboardShowInfoButtonID, Emoji: &discordgo.ComponentEmoji{Name: "ℹ️"}},
				discordgo.Button{Label: "ロール一覧", Style: discordgo.SecondaryButton, CustomID: DashboardShowRolesButtonID, Emoji: &discordgo.ComponentEmoji{Name: "📜"}},
			},
		},
	}

	// ★★★ エラーの修正箇所 ★★★
	// 1. まずEmbedのスライスを作成します
	embeds := []*discordgo.MessageEmbed{embed}
	// 2. MessageEdit構造体には、そのスライスへのポインタを渡します
	editData := &discordgo.MessageEdit{
		Channel:    config.ChannelID,
		ID:         config.MessageID,
		Embeds:     &embeds,
		Components: &components,
	}
	_, err = s.ChannelMessageEditComplex(editData)

	if err != nil {
		c.Log.Error("ダッシュボードの更新に失敗", "error", err)
	}
}

func (c *DashboardCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.MessageComponentData().CustomID {
	case DashboardShowInfoButtonID:
		c.showServerInfo(s, i)
	case DashboardShowRolesButtonID:
		c.showRolesList(s, i)
	}
}

func (c *DashboardCommand) showServerInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guild, _ := s.State.Guild(i.GuildID)

	featureStrings := make([]string, len(guild.Features))
	for i, f := range guild.Features {
		featureStrings[i] = string(f)
	}
	features := "なし"
	if len(featureStrings) > 0 {
		features = strings.Join(featureStrings, ", ")
	}

	embed := &discordgo.MessageEmbed{
		Title: "サーバー詳細情報",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "サーバーID", Value: guild.ID},
			{Name: "オーナー", Value: fmt.Sprintf("<@%s>", guild.OwnerID)},
			{Name: "認証レベル", Value: verificationLevelToString(guild.VerificationLevel)},
			{Name: "サーバー機能", Value: fmt.Sprintf("```\n%s\n```", features)},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *DashboardCommand) showRolesList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guild, _ := s.State.Guild(i.GuildID)
	var rolesStr strings.Builder
	for _, role := range guild.Roles {
		rolesStr.WriteString(fmt.Sprintf("<@&%s> (`%s`)\n", role.ID, role.ID))
	}
	embed := &discordgo.MessageEmbed{
		Title:       "ロール一覧",
		Description: rolesStr.String(),
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func verificationLevelToString(level discordgo.VerificationLevel) string {
	switch level {
	case discordgo.VerificationLevelNone:
		return "なし"
	case discordgo.VerificationLevelLow:
		return "低"
	case discordgo.VerificationLevelMedium:
		return "中"
	case discordgo.VerificationLevelHigh:
		return "高"
	case discordgo.VerificationLevelVeryHigh:
		return "最高"
	default:
		return "不明"
	}
}

func (c *DashboardCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *DashboardCommand) GetComponentIDs() []string {
	return []string{DashboardShowInfoButtonID, DashboardShowRolesButtonID}
}
func (c *DashboardCommand) GetCategory() string { return "管理" }

