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
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message_id",
				Description: "対象メッセージのID",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "emoji",
				Description: "対象の絵文字 (例: 👍 や カスタム絵文字ID)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "role",
				Description: "付与するロール",
				Required:    true,
			},
		},
	}
}

func (c *ReactionRoleCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	messageID := options[0].StringValue()
	emoji := options[1].StringValue()
	role := options[2].RoleValue(s, i.GuildID)

	guildID := i.GuildID
	config := c.Store.GetGuildConfig(guildID)

	// config.ReactionRole が nil の場合は初期化
	if config.ReactionRole == nil {
		config.ReactionRole = make(map[string]string)
	}

	// キーを作成 (メッセージID_絵文字ID)
	key := fmt.Sprintf("%s_%s", messageID, emoji)
	config.ReactionRole[key] = role.ID

	if err := c.Store.SetGuildConfig(guildID, config); err != nil {
		// ...エラー処理...
		return
	}
	if err := c.Store.Save(); err != nil {
		// ...エラー処理...
		return
	}

	// 確認メッセージを送信
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 設定完了！メッセージ `%s` の絵文字 `%s` にロール <@&%s> を紐付けました。", messageID, emoji, role.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// Bot自身も対象のメッセージにリアクションを付けておく
	s.MessageReactionAdd(i.ChannelID, messageID, emoji)
}

// HandleReactionAdd はリアクションが追加されたときの処理です (main.goから呼び出される)
func (c *ReactionRoleCommand) HandleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	guildID := r.GuildID
	config := c.Store.GetGuildConfig(guildID)

	key := fmt.Sprintf("%s_%s", r.MessageID, r.Emoji.APIName())
	roleID, ok := config.ReactionRole[key]
	if !ok {
		return // 設定されていないリアクションなら何もしない
	}

	err := s.GuildMemberRoleAdd(guildID, r.UserID, roleID)
	if err != nil {
		logger.Error.Printf("ロールの付与に失敗 (User: %s, Role: %s): %v", r.UserID, roleID, err)
	}
}

// HandleReactionRemove はリアクションが削除されたときの処理です (main.goから呼び出される)
func (c *ReactionRoleCommand) HandleReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	guildID := r.GuildID
	config := c.Store.GetGuildConfig(guildID)

	key := fmt.Sprintf("%s_%s", r.MessageID, r.Emoji.APIName())
	roleID, ok := config.ReactionRole[key]
	if !ok {
		return
	}

	err := s.GuildMemberRoleRemove(guildID, r.UserID, roleID)
	if err != nil {
		logger.Error.Printf("ロールの削除に失敗 (User: %s, Role: %s): %v", r.UserID, roleID, err)
	}
}

func (c *ReactionRoleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ReactionRoleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
