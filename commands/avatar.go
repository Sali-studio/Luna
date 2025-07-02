package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type AvatarCommand struct{}

func (c *AvatarCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "avatar",
		Description: "ユーザーのアバターやバナーを表示します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "情報を表示するユーザー",
				Required:    false,
			},
		},
	}
}

func (c *AvatarCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var targetUser *discordgo.User
	var targetMember *discordgo.Member

	if len(options) > 0 {
		targetUser = options[0].UserValue(s)
		// UserオブジェクトからMemberオブジェクトを取得する必要がある
		m, err := s.State.Member(i.GuildID, targetUser.ID)
		if err != nil {
			m, err = s.GuildMember(i.GuildID, targetUser.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "❌ メンバー情報の取得に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
				})
				return
			}
		}
		targetMember = m
	} else {
		targetUser = i.Member.User
		targetMember = i.Member
	}

	// ユーザー情報を最新の状態に更新
	// バナー情報を取得するために必要
	userWithBanner, err := s.User(targetUser.ID)
	if err != nil {
		userWithBanner = targetUser // 取得失敗の場合は元の情報を使う
	}

	// --- URLの生成 ---
	avatarURL := targetUser.AvatarURL("1024")
	serverAvatarURL := targetMember.AvatarURL("1024")
	bannerURL := userWithBanner.BannerURL("1024")

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s のプロフィール画像", targetUser.Username),
		Color: 0x7289da,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    targetUser.String(),
			IconURL: avatarURL,
		},
		Fields: []*discordgo.MessageEmbedField{},
	}

	// グローバルアバターをメイン画像に設定
	embed.Image = &discordgo.MessageEmbedImage{URL: avatarURL}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "グローバルアバターURL", Value: fmt.Sprintf("[リンク](%s)", avatarURL)})

	// サーバーアバターが設定されている場合
	if targetMember.Avatar != "" {
		// サーバーアバターをサムネイルに設定
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: serverAvatarURL}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "サーバーアバターURL", Value: fmt.Sprintf("[リンク](%s)", serverAvatarURL)})
	}

	// バナーが設定されている場合
	if userWithBanner.Banner != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "バナー画像URL", Value: fmt.Sprintf("[リンク](%s)", bannerURL)})
		// 別のEmbedとしてバナー画像を送信することも可能
		// s.ChannelMessageSendEmbed(i.ChannelID, &discordgo.MessageEmbed{Image: &discordgo.MessageEmbedImage{URL: bannerURL}})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *AvatarCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AvatarCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AvatarCommand) GetComponentIDs() []string                                            { return []string{} }
