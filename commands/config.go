package commands

import (
	"fmt"
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
		logger.Info.Println("config command received")

		// 設定ダッシュボードのEmbedを作成
		embed := &discordgo.MessageEmbed{
			Title:       "⚙️ Luna 設定ダッシュボード",
			Description: "設定したい項目のボタンを押してください。",
			Color:       0x95A5A6, // グレー
		}

		// 各機能の設定ボタンを作成
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "チケット機能",
						Style:    discordgo.SecondaryButton,
						Emoji:    discordgo.ComponentEmoji{Name: "🎫"},
						CustomID: "config_ticket_button",
					},
					discordgo.Button{
						Label:    "ログ機能",
						Style:    discordgo.SecondaryButton,
						Emoji:    discordgo.ComponentEmoji{Name: "📜"},
						CustomID: "config_log_button",
					},
					discordgo.Button{
						Label:    "一時VC機能",
						Style:    discordgo.SecondaryButton,
						Emoji:    discordgo.ComponentEmoji{Name: "🔊"},
						CustomID: "config_temp_vc_button",
					},
				},
			},
		}

		// ダッシュボードを本人にだけ見える形で送信
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// --- 各設定モーダルを表示する関数群 ---

// HandleShowTicketConfigModal はチケット設定モーダルを表示します
func HandleShowTicketConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "config_ticket_modal",
			Title:    "チケット機能 設定",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "category_id", Label: "チケットを作成するカテゴリのID", Style: discordgo.TextInputShort, Required: true},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "staff_role_id", Label: "対応するスタッフロールのID", Style: discordgo.TextInputShort, Required: true},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show ticket config modal: %v", err)
	}
}

// HandleShowLogConfigModal はログ設定モーダルを表示します
func HandleShowLogConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "config_log_modal",
			Title:    "ログ機能 設定",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "log_channel_id", Label: "ログを送信するチャンネルのID", Style: discordgo.TextInputShort, Required: true},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show log config modal: %v", err)
	}
}

// HandleShowTempVCConfigModal は一時VC設定モーダルを表示します
func HandleShowTempVCConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "config_temp_vc_modal",
			Title:    "一時VC機能 設定",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "lobby_channel_id", Label: "ロビーとして使うボイスチャンネルのID", Style: discordgo.TextInputShort, Required: true},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{CustomID: "creation_category_id", Label: "VCを作成するカテゴリのID", Style: discordgo.TextInputShort, Required: true},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show temp vc config modal: %v", err)
	}
}

// --- モーダルから送信された設定を保存する関数群 ---

func HandleSaveTicketConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	categoryID := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	staffRoleIDValue := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	ticketCategoryID[i.GuildID] = categoryID
	ticketStaffRoleID[i.GuildID] = staffRoleIDValue

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ チケット機能の設定を保存しました。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleSaveLogConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	logChannelIDValue := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	logChannelID[i.GuildID] = logChannelIDValue

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ ログチャンネルを <#%s> に設定しました。", logChannelIDValue),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleSaveTempVCConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	lobbyID := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	categoryID := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	tempVCLobbyID[i.GuildID] = lobbyID
	tempVCCategoryID[i.GuildID] = categoryID

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ 一時VC機能の設定を保存しました。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
