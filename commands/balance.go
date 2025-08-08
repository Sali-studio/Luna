package commands

import (
	"fmt"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// BalanceCommand handles the /balance command.
type BalanceCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *BalanceCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "balance",
		Description: "チップの残高を表示します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "残高を確認したいユーザー（任意）",
				Required:    false,
			},
		},
	}
}

func (c *BalanceCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var targetUser *discordgo.User

	if len(options) > 0 {
		targetUser = options[0].UserValue(s)
	} else {
		targetUser = i.Member.User
	}

	casinoData, err := c.Store.GetCasinoData(i.GuildID, targetUser.ID)
	if err != nil {
		c.Log.Error("Failed to get casino data for balance command", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。後でもう一度お試しください。")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("💰 %s のチップ残高", targetUser.Username),
		Description: fmt.Sprintf("現在のチップ: **%d**\n🐸 **PepeCoin (PPC)**: `%d`", casinoData.Chips, casinoData.PepeCoinBalance),
		Color:       0x3498db, // Blue
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL(""),
		},
	}
	sendEmbedResponse(s, i, embed)
}

func (c *BalanceCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *BalanceCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *BalanceCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *BalanceCommand) GetCategory() string                                                  { return "カジノ" }
