package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

// AutoRoleCommand は自動ロール付与の設定を管理します。
type AutoRoleCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *AutoRoleCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "autorole",
		Description:              "ユーザー参加時の自動ロール付与を設定します。",
		DefaultMemberPermissions: &[]int64{int64(discordgo.PermissionManageGuild)}[0],
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "set",
				Description: "参加時に付与するロールを設定します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionRole,
						Name:        "role",
						Description: "付与するロール",
						Required:    true,
					},
				},
			},
			{
				Name:        "disable",
				Description: "自動ロール付与を無効にします。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "status",
				Description: "現在の設定状況を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *AutoRoleCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Options[0].Name {
	case "set":
		c.handleSet(s, i)
	case "disable":
		c.handleDisable(s, i)
	case "status":
		c.handleStatus(s, i)
	}
}

func (c *AutoRoleCommand) handleSet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	role := i.ApplicationCommandData().Options[0].Options[0].RoleValue(s, i.GuildID)

	config := storage.AutoRoleConfig{
		Enabled: true,
		RoleID:  role.ID,
	}

	err := c.Store.SaveConfig(i.GuildID, "autorole_config", &config)
	if err != nil {
		c.Log.Error("Failed to save autorole config", "error", err)
		sendErrorResponse(s, i, "設定の保存に失敗しました。）")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ 設定を更新しました",
		Description: fmt.Sprintf("新しいメンバーには自動的に <@&%s> ロールが付与されます。", role.ID),
		Color:       0x77b255, // Green
	}
	sendEmbedResponse(s, i, embed)
}

func (c *AutoRoleCommand) handleDisable(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := storage.AutoRoleConfig{
		Enabled: false,
		RoleID:  "",
	}

	err := c.Store.SaveConfig(i.GuildID, "autorole_config", &config)
	if err != nil {
		c.Log.Error("Failed to disable autorole", "error", err)
		sendErrorResponse(s, i, "設定の無効化に失敗しました。）")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🗑️ 設定を無効化しました",
		Description: "自動ロール付与は現在無効です。",
		Color:       0xe74c3c, // Red
	}
	sendEmbedResponse(s, i, embed)
}

func (c *AutoRoleCommand) handleStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var config storage.AutoRoleConfig
	err := c.Store.GetConfig(i.GuildID, "autorole_config", &config)
	if err != nil {
		c.Log.Error("Failed to get autorole config", "error", err)
		sendErrorResponse(s, i, "設定の取得に失敗しました。）")
		return
	}

	var description string
	if config.Enabled && config.RoleID != "" {
		description = fmt.Sprintf("現在、新しいメンバーには <@&%s> ロールが自動的に付与されます。", config.RoleID)
	} else {
		description = "自動ロール付与は現在無効です。"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ 自動ロール設定",
		Description: description,
		Color:       0x3498db, // Blue
	}
	sendEmbedResponse(s, i, embed)
}

func (c *AutoRoleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AutoRoleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AutoRoleCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *AutoRoleCommand) GetCategory() string                                                  { return "管理" }
