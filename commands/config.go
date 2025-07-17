package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

func int64Ptr(v int64) *int64 {
	return &v
}

type ConfigCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *ConfigCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "config",
		Description:              "サーバー固有の設定を管理します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
		Options: []*discordgo.ApplicationCommandOption{
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
	options := i.ApplicationCommandData().Options[0].Options
	switch i.ApplicationCommandData().Options[0].Name {
	case "logging":
		c.handleLoggingConfig(s, i, options)
	case "temp-vc":
		c.handleTempVCConfig(s, i, options)
	case "bump-reminder":
		c.handleBumpConfig(s, i, options)
	}
}

func (c *ConfigCommand) handleLoggingConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var config storage.LogConfig
	config.ChannelID = options[0].ChannelValue(s).ID
	if err := c.Store.SaveConfig(i.GuildID, "log_config", config); err != nil {
		c.Log.Error("ログ設定の保存に失敗", "error", err, "guildID", i.GuildID)
		return
	}
	content := fmt.Sprintf("✅ ログチャンネルを <#%s> に設定しました。", config.ChannelID)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to respond to interaction", "error", err)
	}
}

func (c *ConfigCommand) handleTempVCConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var config storage.TempVCConfig
	config.LobbyID = options[0].ChannelValue(s).ID
	config.CategoryID = options[1].ChannelValue(s).ID
	if err := c.Store.SaveConfig(i.GuildID, "temp_vc_config", config); err != nil {
		c.Log.Error("一時VC設定の保存に失敗", "error", err, "guildID", i.GuildID)
		return
	}
	content := fmt.Sprintf("✅ 一時VC設定を更新しました.\n- ロビーチャンネル: <#%s>\n- 作成先カテゴリ: <#%s>", config.LobbyID, config.CategoryID)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to respond to interaction", "error", err)
	}
}

func (c *ConfigCommand) handleBumpConfig(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var config storage.BumpConfig
	config.Reminder = options[0].BoolValue()
	config.ChannelID = options[1].ChannelValue(s).ID
	config.RoleID = options[2].RoleValue(s, i.GuildID).ID
	if err := c.Store.SaveConfig(i.GuildID, "bump_config", config); err != nil {
		c.Log.Error("BUMPリマインダー設定の保存に失敗", "error", err, "guildID", i.GuildID)
		return
	}
	content := fmt.Sprintf("✅ BUMPリマインダー設定を更新しました.\n- 有効: `%v`\n- チャンネル: <#%s>\n- ロール: <@&%s>", config.Reminder, config.ChannelID, config.RoleID)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to respond to interaction", "error", err)
	}
}

func (c *ConfigCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ConfigCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ConfigCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *ConfigCommand) GetCategory() string {
	return "管理"
}
