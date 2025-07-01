package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

const (
	// ボタンのカスタムID
	CreateTicketButtonID = "create_ticket_button"
)

type TicketCommand struct {
	Store *storage.ConfigStore
}

// GetCommandDef は /ticket-setup コマンドの定義を返します
func (c *TicketCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "ticket-setup",
		Description:              "チケット作成パネルをこのチャンネルに設置します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
	}
}

// Handle は /ticket-setup コマンドの処理です
func (c *TicketCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "サポートチケット",
					Description: "下のボタンを押してサポートチケットを作成してください。",
					Color:       0x5865F2, // Discord Blurple
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "チケットを作成",
							Style:    discordgo.PrimaryButton,
							CustomID: CreateTicketButtonID,
							Emoji: discordgo.ComponentEmoji{
								Name: "🎫",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("チケットパネルの送信に失敗: %v", err)
	}
}

// HandleComponent はチケット作成ボタンが押されたときの処理です
func (c *TicketCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.MessageComponentData().CustomID != CreateTicketButtonID {
		return // このコマンドが処理するボタンではない
	}

	// 「考え中...」と即時応答する
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Error.Printf("チケット作成の応答(defer)に失敗: %v", err)
		return
	}

	guildID := i.GuildID
	config := c.Store.GetGuildConfig(guildID)

	// 設定がされているかチェック
	if config.Ticket.CategoryID == "" || config.Ticket.StaffRoleID == "" {
		content := "❌ チケット機能がまだ設定されていません。サーバー管理者に連絡してください。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// チケットチャンネルを作成
	ch, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("ticket-%s", i.Member.User.Username),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: config.Ticket.CategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ // @everyone を非表示に
				ID:   guildID,
				Type: discordgo.PermissionOverwriteTypeRole,
				Deny: discordgo.PermissionViewChannel,
			},
			{ // チケット作成者を表示
				ID:    i.Member.User.ID,
				Type:  discordgo.PermissionOverwriteTypeMember,
				Allow: discordgo.PermissionViewChannel,
			},
			{ // スタッフロールを表示
				ID:    config.Ticket.StaffRoleID,
				Type:  discordgo.PermissionOverwriteTypeRole,
				Allow: discordgo.PermissionViewChannel,
			},
		},
	})

	if err != nil {
		logger.Error.Printf("チケットチャンネルの作成に失敗: %v", err)
		content := "❌ チャンネルの作成に失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// 最初のメッセージを送信
	// ...

	// 成功メッセージを送信
	content := fmt.Sprintf("✅ チケットを作成しました: <#%s>", ch.ID)
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}

func (c *TicketCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
