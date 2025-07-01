package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

// init関数はinit()の中で完結させる
func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "config",
		Description:              "ボットの各種設定を行うダッシュボードを開きます。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("config command received")

		embed := &discordgo.MessageEmbed{
			Title:       "⚙️ Luna 設定ダッシュボード",
			Description: "設定したい項目のボタンを押してください。\n設定はすべてこのサーバーに保存されます。",
			Color:       0x95A5A6,
		}
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "チケット機能", Style: discordgo.SecondaryButton, Emoji: &discordgo.ComponentEmoji{Name: "🎫"}, CustomID: "config_ticket_button"},
					discordgo.Button{Label: "ログ機能", Style: discordgo.SecondaryButton, Emoji: &discordgo.ComponentEmoji{Name: "📜"}, CustomID: "config_log_button"},
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

// 他のファイルから呼び出される関数は、名前の先頭を大文字にする
func HandleShowTicketConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := Config.GetGuildConfig(i.GuildID)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "config_ticket_modal",
			Title:    "チケット機能 設定",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "panel_channel_id", Label: "パネルを設置するチャンネルID", Style: discordgo.TextInputShort, Value: config.Ticket.PanelChannelID, Required: true}}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "category_id", Label: "チケットを作成するカテゴリのID", Style: discordgo.TextInputShort, Value: config.Ticket.CategoryID, Required: true}}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "staff_role_id", Label: "対応するスタッフロールのID", Style: discordgo.TextInputShort, Value: config.Ticket.StaffRoleID, Required: true}}},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show ticket config modal: %v", err)
	}
}
func HandleShowLogConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := Config.GetGuildConfig(i.GuildID)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "config_log_modal",
			Title:    "ログ機能 設定",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "log_channel_id", Label: "ログを送信するチャンネルのID", Style: discordgo.TextInputShort, Value: config.Log.ChannelID, Required: true},
				}},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show log config modal: %v", err)
	}
}
func HandleSaveTicketConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	config := Config.GetGuildConfig(i.GuildID)

	config.Ticket.PanelChannelID = data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	config.Ticket.CategoryID = data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	config.Ticket.StaffRoleID = data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	Config.SaveGuildConfig(i.GuildID, config)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "✅ チケット機能の設定を保存しました。", Flags: discordgo.MessageFlagsEphemeral},
	})
}
func HandleSaveLogConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	config := Config.GetGuildConfig(i.GuildID)
	config.Log.ChannelID = data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	Config.SaveGuildConfig(i.GuildID, config)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("✅ ログチャンネルを <#%s> に設定しました。", config.Log.ChannelID), Flags: discordgo.MessageFlagsEphemeral},
	})
}
