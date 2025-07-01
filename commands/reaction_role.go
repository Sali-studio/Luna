package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"strings"

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
			{Type: discordgo.ApplicationCommandOptionString, Name: "emoji", Description: "対象の絵文字 (例: 👍 またはカスタム絵文字)", Required: true},
			{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "付与するロール", Required: true},
		},
	}
}

func (c *ReactionRoleCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	messageID := options[0].StringValue()
	emojiInput := options[1].StringValue()
	role := options[2].RoleValue(s, i.GuildID)

	// カスタム絵文字のIDを抽出
	emojiID := emojiInput
	if strings.HasPrefix(emojiInput, "<:") && strings.HasSuffix(emojiInput, ">") {
		parts := strings.Split(strings.Trim(emojiInput, "<>"), ":")
		if len(parts) == 3 {
			emojiID = parts[2]
		}
	}

	config := c.Store.GetGuildConfig(i.GuildID)
	key := fmt.Sprintf("%s_%s", messageID, emojiID) // 標準絵文字でもカスタム絵文字IDでも対応
	config.ReactionRoles[key] = role.ID

	if err := c.Store.Save(); err != nil {
		logger.Error.Printf("リアクションロール設定の保存に失敗: %v", err)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 設定完了！メッセージ `%s` の絵文字 `%s` にロール <@&%s> を紐付けました。", messageID, emojiInput, role.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	s.MessageReactionAdd(i.ChannelID, messageID, emojiInput)
}

func (c *ReactionRoleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ReactionRoleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ReactionRoleCommand) GetComponentIDs() []string                                            { return []string{} }
