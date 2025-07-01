package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

var rrStore *storage.ReactionRoleStore

func init() {
	var err error
	rrStore, err = storage.NewReactionRoleStore("reaction_roles.json")
	if err != nil {
		logger.Fatal.Fatalf("Failed to initialize reaction role store: %v", err)
	}

	cmd := &discordgo.ApplicationCommand{
		Name:                     "reaction-role-setup",
		Description:              "リアクションロールを設定します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageRoles),
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "message_id", Description: "対象メッセージのID", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "emoji", Description: "対象の絵文字", Required: true},
			{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "付与するロール", Required: true},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		messageID := optionMap["message_id"].StringValue()
		emoji := optionMap["emoji"].StringValue()
		role := optionMap["role"].RoleValue(s, i.GuildID)

		// メッセージに実際にリアクションを追加して、設定が正しいか確認
		err := s.MessageReactionAdd(i.ChannelID, messageID, emoji)
		if err != nil {
			logger.Error.Printf("Failed to add reaction for setup: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "❌ 絵文字のリアクションに失敗しました。絵文字が正しいか、ボットがアクセスできるか確認してください。", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		rr := &storage.ReactionRole{
			MessageID: messageID,
			Emoji:     emoji,
			RoleID:    role.ID,
		}
		if err := rrStore.Add(rr); err != nil {
			logger.Error.Printf("Failed to save reaction role: %v", err)
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ リアクションロールを設定しました！\nメッセージID: `%s`\n絵文字: %s\nロール: %s", messageID, emoji, role.Mention()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// HandleMessageReactionAdd はリアクションが追加されたときの処理
func HandleMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}

	roleID, ok := rrStore.Get(r.MessageID, r.Emoji.APIName())
	if !ok {
		return
	}

	err := s.GuildMemberRoleAdd(r.GuildID, r.UserID, roleID)
	if err != nil {
		logger.Error.Printf("Failed to add role: %v", err)
	}
}

// HandleMessageReactionRemove はリアクションが削除されたときの処理
func HandleMessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}

	roleID, ok := rrStore.Get(r.MessageID, r.Emoji.APIName())
	if !ok {
		return
	}

	err := s.GuildMemberRoleRemove(r.GuildID, r.UserID, roleID)
	if err != nil {
		logger.Error.Printf("Failed to remove role: %v", err)
	}
}
