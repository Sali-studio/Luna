package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

type ReactionRoleCommand struct {
	Store *storage.ConfigStore
}

func (c *ReactionRoleCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "reaction-role-setup",
		Description:              "指定したメッセージにリアクションロールを設定します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageRoles),
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "message_id", Description: "対象メッセージのID", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "emoji", Description: "対象の絵文字 (例: 👍)", Required: true},
			{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "付与するロール", Required: true},
		},
	}
}

func (c *ReactionRoleCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	messageID := options[0].StringValue()
	emoji := options[1].StringValue()
	role := options[2].RoleValue(s, i.GuildID)

	config := c.Store.GetGuildConfig(i.GuildID)
	key := fmt.Sprintf("%s_%s", messageID, emoji)
	config.ReactionRoles[key] = role.ID

	if err := c.Store.Save(); err != nil {
		logger.Error.Printf("リアクションロール設定の保存に失敗: %v", err)
		// ...エラーレスポンス...
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 設定完了！メッセージ `%s` の絵文字 `%s` にロール <@&%s> を紐付けました。", messageID, emoji, role.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	s.MessageReactionAdd(i.ChannelID, messageID, emoji)
}

func (c *ReactionRoleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ReactionRoleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ReactionRoleCommand) GetComponentIDs() []string                                            { return []string{} }
