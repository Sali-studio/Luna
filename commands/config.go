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
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "panel_channel", Description: "チケットパネルを設置するチャンネル", Required: true, ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText}},
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "category", Description: "チケットが作成されるカテゴリ", ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory}, Required: true},
					{Type: discordgo.ApplicationCommandOptionRole, Name: "staff_role", Description: "チケットに対応するスタッフのロール", Required: true},
				},
			},
			{
				Name:        "logging",
				Description: "ログ出力チャンネルを設定します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "ログを出力するチャンネル", Required: true, ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText}},
				},
			},
			{
				Name:        "temp-vc",
				Description: "一時ボイスチャンネル機能の設定",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "lobby_channel", Description: "入室するとVCが作成されるロビーチャンネル", ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildVoice}, Required: true},
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "category", Description: "一時VCが作成されるカテゴリ", ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory}, Required: true},
				},
			},
			{
				Name:        "bump-reminder",
				Description: "BUMPリマインダー機能の設定",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionBoolean, Name: "enable", Description: "機能を有効にするか", Required: true},
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "BUMPコマンドが実行されるチャンネル", Required: true, ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText}},
					{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "リマインド時にメンションするロール", Required: true},
				},
			},
		},
	}
}

func (c *ConfigCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subCommand := i.ApplicationCommandData().Options[0]
	switch subCommand.Name {
	case "ticket":
		c.handleTicketConfig(s, i, subCommand.Options)
	case "logging":
		c.handleLoggingConfig(s, i, subCommand.Options)
	case "temp-vc":
		c.handleTempVCConfig(s, i, subCommand.Options)
	case "bump-reminder":
		c.handleBumpConfig(s, i, subCommand.Options)
	}
}

func (c *ConfigCommand) handleTicketConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandOption) {
	panelChannelID := options[0].ChannelValue(s).ID
	categoryID := options[1].ChannelValue(s).ID
	staffRoleID := options[2].RoleValue(s, i.GuildID).ID

	config := c.Store.GetGuildConfig(i.GuildID)
	config.Ticket.PanelChannelID = panelChannelID
	config.Ticket.CategoryID = categoryID
	config.Ticket.StaffRoleID = staffRoleID

	if err := c.Store.Save(); err != nil {
		logger.Error.Printf("設定ファイルの書き込みに失敗 (Guild: %s): %v", i.GuildID, err)
		return
	}

	content := fmt.Sprintf("✅ チケット設定を更新しました。\n- パネルチャンネル: <#%s>\n- カテゴリ: <#%s>\n- スタッフロール: <@&%s>", panelChannelID, categoryID, staffRoleID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *ConfigCommand) handleLoggingConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandOption) {
	channelID := options[0].ChannelValue(s).ID

	config := c.Store.GetGuildConfig(i.GuildID)
	config.Log.ChannelID = channelID
	c.Store.Save()

	content := fmt.Sprintf("✅ ログチャンネルを <#%s> に設定しました。", channelID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *ConfigCommand) handleTempVCConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandOption) {
	lobbyID := options[0].ChannelValue(s).ID
	categoryID := options[1].ChannelValue(s).ID

	config := c.Store.GetGuildConfig(i.GuildID)
	config.TempVC.LobbyID = lobbyID
	config.TempVC.CategoryID = categoryID
	c.Store.Save()

	content := fmt.Sprintf("✅ 一時VC設定を更新しました。\n- ロビーチャンネル: <#%s>\n- 作成先カテゴリ: <#%s>", lobbyID, categoryID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *ConfigCommand) handleBumpConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandOption) {
	enable := options[0].BoolValue()
	channelID := options[1].ChannelValue(s).ID
	roleID := options[2].RoleValue(s, i.GuildID).ID

	config := c.Store.GetGuildConfig(i.GuildID)
	config.Bump.Reminder = enable
	config.Bump.ChannelID = channelID
	config.Bump.RoleID = roleID
	c.Store.Save()

	content := fmt.Sprintf("✅ BUMPリマインダー設定を更新しました。\n- 有効: `%v`\n- チャンネル: <#%s>\n- ロール: <@&%s>", enable, channelID, roleID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *ConfigCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ConfigCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ConfigCommand) GetComponentIDs() []string                                            { return []string{} }
