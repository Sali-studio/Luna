package commands

import (
	"fmt"
	"luna/logger"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "info",
		Description: "サーバーまたはユーザーの情報を表示します。",
		Options: []*discordgo.ApplicationCommandOption{
			// --- server サブコマンド ---
			{
				Name:        "server",
				Description: "このサーバーの情報を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			// --- user サブコマンド ---
			{
				Name:        "user",
				Description: "指定したユーザーの情報を表示します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "target",
						Description: "情報を表示するユーザー（未指定の場合は自分）",
						Required:    false,
					},
				},
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// 実行されたサブコマンドによって処理を分岐
		switch i.ApplicationCommandData().Options[0].Name {
		case "server":
			handleServerInfo(s, i)
		case "user":
			handleUserInfo(s, i)
		}
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// handleServerInfo はサーバー情報を表示します
func handleServerInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Info.Println("info server command received")

	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		logger.Error.Printf("Failed to get guild info: %v", err)
		return
	}

	createdAt, _ := discordgo.SnowflakeTimestamp(guild.ID)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("サーバー情報: %s", guild.Name),
		Description: guild.Description,
		Color:       0x7289DA,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: guild.IconURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "👑 オーナー", Value: fmt.Sprintf("<@%s>", guild.OwnerID), Inline: true},
			{Name: "👥 メンバー数", Value: fmt.Sprintf("%d人", guild.MemberCount), Inline: true},
			{Name: "📅 作成日", Value: fmt.Sprintf("<t:%d:F>", createdAt.Unix()), Inline: false},
			{Name: "📜 ロール数", Value: fmt.Sprintf("%d個", len(guild.Roles)), Inline: true},
			{Name: "😀 絵文字数", Value: fmt.Sprintf("%d個", len(guild.Emojis)), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// handleUserInfo はユーザー情報を表示します
func handleUserInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Info.Println("info user command received")

	var targetUser *discordgo.User
	if len(i.ApplicationCommandData().Options[0].Options) > 0 {
		targetUser = i.ApplicationCommandData().Options[0].Options[0].UserValue(s)
	} else {
		targetUser = i.Member.User
	}

	member, err := s.State.Member(i.GuildID, targetUser.ID)
	if err != nil {
		logger.Error.Printf("Failed to get member info: %v", err)
		return
	}

	userCreatedAt, _ := discordgo.SnowflakeTimestamp(member.User.ID)
	// ★★★ ここが修正箇所です ★★★
	// .Parse() は不要なので削除します
	joinedAt := member.JoinedAt

	var roles []string
	for _, roleID := range member.Roles {
		role, _ := s.State.Role(i.GuildID, roleID)
		if role.Name != "@everyone" {
			roles = append(roles, role.Mention())
		}
	}
	rolesStr := "なし"
	if len(roles) > 0 {
		rolesStr = strings.Join(roles, " ")
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    targetUser.Username,
			IconURL: targetUser.AvatarURL(""),
		},
		Color: 0x2ECC71,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📛 名前", Value: targetUser.Mention(), Inline: true},
			{Name: "🆔 ユーザーID", Value: fmt.Sprintf("`%s`", targetUser.ID), Inline: true},
			{Name: "📅 アカウント作成日", Value: fmt.Sprintf("<t:%d:f>", userCreatedAt.Unix()), Inline: false},
			{Name: "👋 サーバー参加日", Value: fmt.Sprintf("<t:%d:f>", joinedAt.Unix()), Inline: false},
			{Name: "🎭 ロール", Value: rolesStr, Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
