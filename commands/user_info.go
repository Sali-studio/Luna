package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

type UserInfoCommand struct{}

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
	var user *discordgo.User
	if len(options) > 0 {
		user = options[0].UserValue(s)
	} else {
		user = i.Member.User
	}

	member, err := s.State.Member(i.GuildID, user.ID)
	if err != nil {
		member, err = s.GuildMember(i.GuildID, user.ID)
		if err != nil {
			logger.Error.Printf("メンバー情報の取得に失敗 (User: %s): %v", user.ID, err)
			return
		}
	}

	// サーバー参加日時を取得
	// エラーメッセージに基づき、.Parse() を削除
	joinedAt := member.JoinedAt

	// アカウント作成日時を取得
	// エラーメッセージに基づき、戻り値を2つ受け取るように修正
	createdAt, err := discordgo.SnowflakeTimestamp(user.ID)
	if err != nil {
		logger.Error.Printf("アカウント作成日時の取得に失敗: %v", err)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("%s の情報", user.Username),
		Color:     0x7289da,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: user.AvatarURL("")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ユーザー名", Value: user.String(), Inline: true},
			{Name: "ID", Value: user.ID, Inline: true},
			{Name: "Bot", Value: fmt.Sprintf("%v", user.Bot), Inline: true},
			{Name: "アカウント作成日時", Value: fmt.Sprintf("<t:%d:F>", createdAt.Unix()), Inline: false},
			{Name: "サーバー参加日時", Value: fmt.Sprintf("<t:%d:F>", joinedAt.Unix()), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func (c *UserInfoCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *UserInfoCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *UserInfoCommand) GetComponentIDs() []string                                            { return []string{} }
