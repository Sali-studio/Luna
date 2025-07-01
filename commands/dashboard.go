package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "dashboard-setup",
		Description:              "サーバー統計情報のライブダッシュボードを設置します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "ダッシュボードを設置するチャンネル",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Required:     true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		channelID := i.ApplicationCommandData().Options[0].Value.(string)

		// まずは空のEmbedメッセージを送信
		msg, err := s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
			Title: "📊 サーバー統計情報 (初期化中...)",
		})
		if err != nil {
			logger.Error.Printf("Failed to send initial dashboard message: %v", err)
			return
		}

		// 設定を保存
		config := Config.GetGuildConfig(i.GuildID)
		config.Dashboard.ChannelID = channelID
		config.Dashboard.MessageID = msg.ID
		Config.SaveGuildConfig(i.GuildID, config)

		// すぐに最初の更新を実行
		UpdateDashboard(s, i.GuildID, config)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ ダッシュボードを <#%s> に設置しました。", channelID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// UpdateDashboard は指定されたダッシュボードのメッセージを更新します
func UpdateDashboard(s *discordgo.Session, guildID string, config *storage.GuildConfig) {
	// ダッシュボードが設定されていなければ何もしない
	if config.Dashboard.ChannelID == "" || config.Dashboard.MessageID == "" {
		return
	}

	guild, err := s.State.Guild(guildID)
	if err != nil {
		logger.Error.Printf("Failed to get guild for dashboard update: %v", err)
		return
	}

	onlineMembers := 0
	for _, pres := range guild.Presences {
		if pres.Status != discordgo.StatusOffline {
			onlineMembers++
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 %s サーバー統計情報", guild.Name),
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🟢 オンライン", Value: fmt.Sprintf("%d人", onlineMembers), Inline: true},
			{Name: "👥 総メンバー", Value: fmt.Sprintf("%d人", guild.MemberCount), Inline: true},
			{Name: "✨ ブーストレベル", Value: fmt.Sprintf("Level %d", guild.PremiumTier), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("最終更新: %s", time.Now().Format("2006/01/02 15:04:05")),
		},
	}

	_, err = s.ChannelMessageEditEmbed(config.Dashboard.ChannelID, config.Dashboard.MessageID, embed)
	if err != nil {
		logger.Error.Printf("Failed to edit dashboard message: %v", err)
	}
}

// StartDashboardUpdater はすべてのダッシュボードを定期的に更新するループを開始します
func StartDashboardUpdater(s *discordgo.Session) {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		// 起動時にまず全てのギルドの設定を更新
		logger.Info.Println("Running initial dashboard update...")
		for guildID, config := range Config.Configs {
			UpdateDashboard(s, guildID, config)
		}
		// その後、定期実行
		for range ticker.C {
			logger.Info.Println("Updating all dashboards...")
			for guildID, config := range Config.Configs {
				UpdateDashboard(s, guildID, config)
			}
		}
	}()
}
