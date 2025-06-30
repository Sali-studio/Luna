package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

var dashStore *storage.DashboardStore

func init() {
	var err error
	dashStore, err = storage.NewDashboardStore("dashboards.json")
	if err != nil {
		logger.Fatal.Fatalf("Failed to initialize dashboard store: %v", err)
	}

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
		channel := i.ApplicationCommandData().Options[0].ChannelValue(s)

		// まずは空のEmbedメッセージを送信
		msg, err := s.ChannelMessageSendEmbed(channel.ID, &discordgo.MessageEmbed{
			Title: "📊 サーバー統計情報 (初期化中...)",
		})
		if err != nil {
			logger.Error.Printf("Failed to send initial dashboard message: %v", err)
			return
		}

		// 設定を保存
		config := &storage.DashboardConfig{
			GuildID:   i.GuildID,
			ChannelID: channel.ID,
			MessageID: msg.ID,
		}
		if err := dashStore.Set(config); err != nil {
			logger.Error.Printf("Failed to save dashboard config: %v", err)
			return
		}

		// すぐに最初の更新を実行
		UpdateDashboard(s, config)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ ダッシュボードを <#%s> に設置しました。", channel.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// UpdateDashboard は指定されたダッシュボードのメッセージを更新します
func UpdateDashboard(s *discordgo.Session, config *storage.DashboardConfig) {
	guild, err := s.State.Guild(config.GuildID)
	if err != nil {
		logger.Error.Printf("Failed to get guild for dashboard update: %v", err)
		return
	}

	// オンラインメンバー数をカウント
	onlineMembers := 0
	for _, pres := range guild.Presences {
		if pres.Status != discordgo.StatusOffline {
			onlineMembers++
		}
	}

	// 更新後のEmbedを作成
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

	// 既存のメッセージを編集
	_, err = s.ChannelMessageEditEmbed(config.ChannelID, config.MessageID, embed)
	if err != nil {
		logger.Error.Printf("Failed to edit dashboard message: %v", err)
	}
}

// StartDashboardUpdater はすべてのダッシュボードを定期的に更新するループを開始します
func StartDashboardUpdater(s *discordgo.Session) {
	// 5分ごとに実行するTickerを作成
	ticker := time.NewTicker(5 * time.Minute)

	// 即座に最初の更新を実行
	logger.Info.Println("Running initial dashboard update...")
	for _, config := range dashStore.Configs {
		UpdateDashboard(s, config)
	}

	// Tickerのループ
	go func() {
		for range ticker.C {
			logger.Info.Println("Updating all dashboards...")
			for _, config := range dashStore.Configs {
				UpdateDashboard(s, config)
			}
		}
	}()
}
