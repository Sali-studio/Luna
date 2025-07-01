package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type AvatarCommand struct{}

func (c *AvatarCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "avatar",
		Description: "ユーザーのアバターを表示します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "アバターを表示するユーザー",
				Required:    false,
			},
		},
	}
}

func (c *AvatarCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var targetUser *discordgo.User
	if len(options) > 0 {
		targetUser = options[0].UserValue(s)
	} else {
		targetUser = i.Member.User
	}

	avatarURL := targetUser.AvatarURL("1024")

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s のアバター", targetUser.Username),
		Color: 0x7289da,
		Image: &discordgo.MessageEmbedImage{
			URL: avatarURL,
		},
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
