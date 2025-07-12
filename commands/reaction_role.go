package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	ReactionRoleSelectMenuID = "reaction_role_select:"
)

type ReactionRoleCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *ReactionRoleCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "reaction-role-setup",
		Description:              "選択したメッセージにリアクションロールを設定します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageRoles),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "対象のメッセージがあるチャンネル",
				Required:     true,
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "emoji",
				Description: "対象の絵文字 (例: 👍 またはカスタム絵文字)",
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
	channel := options[0].ChannelValue(s)
	emojiInput := options[1].StringValue()
	role := options[2].RoleValue(s, i.GuildID)

	messages, err := s.ChannelMessages(channel.ID, 25, "", "", "")
	if err != nil {
		c.Log.Error("リアクションロール用のメッセージ取得に失敗", "error", err, "channelID", channel.ID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ メッセージの取得に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if len(messages) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ そのチャンネルにはメッセージが見つかりませんでした。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	selectOptions := make([]discordgo.SelectMenuOption, 0, len(messages))
	for _, msg := range messages {
		content := msg.Content

		// ★★★ ここを修正 ★★★
		// runeスライスに変換して、文字数を正しく扱う
		runes := []rune(content)
		if len(runes) > 47 {
			content = string(runes[:47]) + "..."
		}
		// ★★★ ここまで ★★★

		if content == "" && len(msg.Embeds) > 0 {
			content = fmt.Sprintf("Embed: %s", msg.Embeds[0].Title)
		}

		selectOptions = append(selectOptions, discordgo.SelectMenuOption{
			Label:       fmt.Sprintf("%s: %s", msg.Author.Username, content),
			Description: fmt.Sprintf("ID: %s", msg.ID),
			Value:       msg.ID,
		})
	}

	customID := fmt.Sprintf("%s%s:%s", ReactionRoleSelectMenuID, role.ID, emojiInput)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "どのメッセージにリアクションロールを設定しますか？",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    customID,
							Placeholder: "メッセージを選択してください",
							Options:     selectOptions,
						},
					},
				},
			},
		},
	})
	if err != nil {
		c.Log.Error("メッセージ選択メニューの送信に失敗", "error", err)
	}
}

func (c *ReactionRoleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	if !strings.HasPrefix(customID, ReactionRoleSelectMenuID) {
		return
	}

	parts := strings.Split(strings.TrimPrefix(customID, ReactionRoleSelectMenuID), ":")
	if len(parts) < 2 {
		return
	}
	roleID := parts[0]
	emojiInput := strings.Join(parts[1:], ":")

	messageID := i.MessageComponentData().Values[0]

	emojiToSave := emojiInput
	if strings.HasPrefix(emojiInput, "<:") && strings.HasSuffix(emojiInput, ">") {
		emojiParts := strings.Split(strings.Trim(emojiInput, "<>"), ":")
		if len(emojiParts) == 3 {
			emojiToSave = emojiParts[2]
		}
	}
	rr := storage.ReactionRole{
		MessageID: messageID,
		EmojiID:   emojiToSave,
		GuildID:   i.GuildID,
		RoleID:    roleID,
	}
	if err := c.Store.SaveReactionRole(rr); err != nil {
		c.Log.Error("リアクションロール設定の保存に失敗", "error", err)
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("✅ 設定完了！メッセージ `%s` の絵文字 `%s` にロール <@&%s> を紐付けました。", messageID, emojiInput, roleID),
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		c.Log.Error("リアクションロール設定完了メッセージの編集に失敗", "error", err)
	}

	s.MessageReactionAdd(i.ChannelID, messageID, emojiInput)
}

func (c *ReactionRoleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ReactionRoleCommand) GetComponentIDs() []string {
	return []string{ReactionRoleSelectMenuID}
}
func (c *ReactionRoleCommand) GetCategory() string {
	return "管理"
}
