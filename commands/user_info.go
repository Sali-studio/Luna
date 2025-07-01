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
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "情報を表示するユーザー",
				Required:    true,
			},
		},
	}
}

func (c *UserInfoCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	user := options[0].UserValue(s)

	// ユーザーがサーバーに参加した日時を取得
	member, err := s.State.Member(i.GuildID, user.ID)
	if err != nil {
		// State にない場合はAPIから取得を試みる
		member, err = s.GuildMember(i.GuildID, user.ID)
		if err != nil {
			logger.Error.Printf("メンバー情報の取得に失敗 (User: %s): %v", user.ID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ メンバー情報の取得に失敗しました。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	joinedAt, _ := member.JoinedAt.Parse()
	createdAt, _ := user.Timestamp()

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s の情報", user.Username),
		Color: 0x7289da,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ユーザー名", Value: user.String(), Inline: true},
			{Name: "ID", Value: user.ID, Inline: true},
			{Name: "Bot", Value: fmt.Sprintf("%v", user.Bot), Inline: true},
			{Name: "アカウント作成日時", Value: createdAt.Format(time.RFC1123),