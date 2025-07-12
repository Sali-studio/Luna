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
		m, err := s.State.Member(i.GuildID, targetUser.ID)
		if err != nil {
			m, err = s.GuildMember(i.GuildID, targetUser.ID)
			if err != nil {
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "❌ メンバー情報の取得に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
				}); err != nil {
					// Log the error, but we can't do much more
				}
				return
			}
		}
		targetMember = m
	} else {
		targetUser = i.Member.User
		targetMember = i.Member
	}

	userWithBanner, err := s.User(targetUser.ID)
	if err != nil {
		userWithBanner = targetUser
	}

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

	embed.Image = &discordgo.MessageEmbedImage{URL: avatarURL}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "グローバルアバターURL", Value: fmt.Sprintf("[リンク](%s)", avatarURL)})

	if targetMember.Avatar != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: serverAvatarURL}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "サーバーアバターURL", Value: fmt.Sprintf("[リンク](%s)", serverAvatarURL)})
	}

	if userWithBanner.Banner != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "バナー画像URL", Value: fmt.Sprintf("[リンク](%s)", bannerURL)})
	}

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	}); err != nil {
		// Log the error, but we can't do much more
	}
}

func (c *AvatarCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AvatarCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AvatarCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *AvatarCommand) GetCategory() string                                                  { return "ユーティリティ" }
