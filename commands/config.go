package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

// int64Ptr は int64 のポインタを返すヘルパー関数です
func int64Ptr(v int64) *int64 {
	return &v
}

type ConfigCommand struct {
	Store *storage.ConfigStore
}

func (c *ConfigCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "config",
		Description:              "サーバー固有の設定を管理します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "ticket",
				Description: "チケット機能の設定",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:         discordgo.ApplicationCommandOptionChannel,
						Name:         "category",
						Description:  "チケットが作成されるカテゴリ",
						ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory},
						Required:     true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionRole,
						Name:        "staff_role",
						Description: "チケットに対応するスタッフのロール",
						Required:    true,
					},
				},
			},
			// ... 他のサブコマンド (bump-configなど) もここに追加 ...
		},
	}
}

func (c *ConfigCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subCommand := i.ApplicationCommandData().Options[0]
	switch subCommand.Name {
	case "ticket":
		c.handleTicketConfig(s, i, subCommand.Options)
	default:
		// 未知のサブコマンド
	}
}

func (c *ConfigCommand) handleTicketConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandOption) {
	var categoryID, staffRoleID string

	for _, opt := range options {
		switch opt.Name {
		case "category":
			categoryID = opt.ChannelValue(s).ID
		case "staff_role":
			staffRoleID = opt.RoleValue(s, i.GuildID).ID
		}
	}

	guildID := i.GuildID
	config := c.Store.GetGuildConfig(guildID)
	config.Ticket.CategoryID = categoryID
	config.Ticket.StaffRoleID = staffRoleID

	if err := c.Store.SetGuildConfig(guildID, config); err != nil {
		logger.Error.Printf("チケット設定の更新に失敗 (Guild: %s): %v", guildID, err)
		content := "❌ 設定の保存中にエラーが発生しました。"
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if err := c.Store.Save(); err != nil {
		logger.Error.Printf("設定ファイルの書き込みに失敗 (Guild: %s): %v", guildID, err)
		// ... エラーレスポンス ...
		return
	}

	content := fmt.Sprintf(
		"✅ チケット設定を更新しました。\n- カテゴリ: <#%s>\n- スタッフロール: <@&%s>",
		categoryID,
		staffRoleID,
	)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *ConfigCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ConfigCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
