package commands

import (
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "config",
		Description:              "ボットの各種設定を行うダッシュボードを開きます。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		embed := &discordgo.MessageEmbed{
			Title:       "⚙️ Luna 設定ダッシュボード",
			Description: "設定したい項目のボタンを押してください。",
			Color:       0x95A5A6,
		}
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "チケット機能", Style: discordgo.SecondaryButton, Emoji: &discordgo.ComponentEmoji{Name: "🎫"}, CustomID: "config_ticket_button"},
					discordgo.Button{Label: "ログ機能", Style: discordgo.SecondaryButton, Emoji: &discordgo.ComponentEmoji{Name: "📜"}, CustomID: "config_log_button"},
					// ★★★ 一時VC設定ボタンをここに追加 ★★★
					discordgo.Button{Label: "一時VCセットアップ", Style: discordgo.SuccessButton, Emoji: &discordgo.ComponentEmoji{Name: "🔊"}, CustomID: "config_temp_vc_setup"},
				},
			},
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components, Flags: discordgo.MessageFlagsEphemeral},
		})
	}
	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// --- 各設定モーダルを表示する関数 ---
func HandleShowTicketConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate) { /* ... */ }
func HandleShowLogConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate)    { /* ... */ }

// --- モーダルから送信された設定を保存する関数 ---
func HandleSaveTicketConfig(s *discordgo.Session, i *discordgo.InteractionCreate) { /* ... */ }
func HandleSaveLogConfig(s *discordgo.Session, i *discordgo.InteractionCreate)    { /* ... */ }

// ★★★ 一時VCのセットアップを実行する関数 ★★★
func HandleExecuteTempVCSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⏳ 一時VC機能のセットアップを開始します...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	config := Config.GetGuildConfig(i.GuildID)

	// カテゴリを作成
	category, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name: "🎤 Temp Voice Channels",
		Type: discordgo.ChannelTypeGuildCategory,
	})
	if err != nil {
		logger.Error.Printf("Failed to create temp VC category: %v", err)
		return
	}

	// ロビーチャンネルを作成
	lobby, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:     "➕ Join to Create",
		Type:     discordgo.ChannelTypeGuildVoice,
		ParentID: category.ID,
	})
	if err != nil {
		logger.Error.Printf("Failed to create lobby channel: %v", err)
		s.ChannelDelete(category.ID) // ロールバック
		return
	}

	// 設定を保存
	config.TempVC.LobbyID = lobby.ID
	config.TempVC.CategoryID = category.ID
	Config.SaveGuildConfig(i.GuildID, config)

	content := "✅ セットアップが完了しました！\n新しく作成された「➕ Join to Create」チャンネルに参加すると一時的なVCが作成されます。"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}
