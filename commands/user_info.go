package commands

import (
	"fmt"
	"luna/interfaces"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type UserInfoCommand struct {
	Log interfaces.Logger
}

func (c *UserInfoCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "user-info",
		Description: "指定したユーザーの情報を表示します",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "情報を表示するユーザー", Required: false},
		},
	}
}

func (c *UserInfoCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var targetUser *discordgo.User
	if len(options) > 0 {
		targetUser = options[0].UserValue(s)
	} else {
		targetUser = i.Member.User
	}

	member, err := s.State.Member(i.GuildID, targetUser.ID)
	if err != nil {
		member, err = s.GuildMember(i.GuildID, targetUser.ID)
		if err != nil {
			c.Log.Error("メンバー情報の取得に失敗", "error", err, "userID", targetUser.ID)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "❌ メンバー情報の取得に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}
	}

	// --- 情報の整形 ---

	// 1. 日時
	joinedAt := member.JoinedAt
	createdAt, _ := discordgo.SnowflakeTimestamp(targetUser.ID)

	// 2. ロール
	roles := make([]string, 0)
	guildRoles, _ := s.GuildRoles(i.GuildID)
	for _, roleID := range member.Roles {
		for _, role := range guildRoles {
			if role.ID == roleID {
				roles = append(roles, fmt.Sprintf("<@&%s>", role.ID))
				break
			}
		}
	}
	rolesStr := "なし"
	if len(roles) > 0 {
		rolesStr = strings.Join(roles, " ")
	}

	// 3. ステータスとアクティビティ
	presence, err := s.State.Presence(i.GuildID, targetUser.ID)
	statusStr := "オフライン"
	activityStr := "なし"
	if err == nil {
		statusMap := map[discordgo.Status]string{
			discordgo.StatusOnline:       "🟢 オンライン",
			discordgo.StatusIdle:         "🟡 離席中",
			discordgo.StatusDoNotDisturb: "🔴 取り込み中",
			discordgo.StatusInvisible:    "⚪ 不可視",
			discordgo.StatusOffline:      "⚫ オフライン",
		}
		statusStr = statusMap[presence.Status]

		if len(presence.Activities) > 0 {
			activity := presence.Activities[0]
			activityStr = fmt.Sprintf("%s: %s", activityTypeToString(activity.Type), activity.Name)
		}
	}

	// --- Embedの作成 ---
	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("%s の情報", targetUser.Username),
		Color:     s.State.UserColor(targetUser.ID, i.ChannelID),
		Timestamp: time.Now().Format(time.RFC3339),
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: member.AvatarURL("1024")},
		Author: &discordgo.MessageEmbedAuthor{
			Name:    targetUser.String(),
			IconURL: targetUser.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "基本情報", Value: fmt.Sprintf("**ID:** `%s`\n**Bot:** %v", targetUser.ID, targetUser.Bot), Inline: false},
			{Name: "日時", Value: fmt.Sprintf("**アカウント作成:** <t:%d:R>\n**サーバー参加:** <t:%d:R>", createdAt.Unix(), joinedAt.Unix()), Inline: false},
			{Name: "ステータス", Value: statusStr, Inline: true},
			{Name: "アクティビティ", Value: activityStr, Inline: true},
			{Name: fmt.Sprintf("役割 (%d)", len(roles)), Value: rolesStr, Inline: false},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func activityTypeToString(at discordgo.ActivityType) string {
	switch at {
	case discordgo.ActivityTypeGame:
		return "プレイ中"
	case discordgo.ActivityTypeStreaming:
		return "配信中"
	case discordgo.ActivityTypeListening:
		return "リスニング中"
	case discordgo.ActivityTypeWatching:
		return "視聴中"
	case discordgo.ActivityTypeCustom:
		return "カスタムステータス"
	case discordgo.ActivityTypeCompeting:
		return "競争中"
	default:
		return "不明"
	}
}

func (c *UserInfoCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *UserInfoCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *UserInfoCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *UserInfoCommand) GetCategory() string {
	return "ユーティリティ"
}