package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

// CoinflipCommand handles the /coinflip command.
type CoinflipCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *CoinflipCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "coinflip",
		Description: "コインを投げて運試し！",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "bet",
				Description: "ベットするチップの額",
				Required:    true,
				MinValue:    &[]float64{1}[0],
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "choice",
				Description: "表か裏かを選択",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "表", Value: "heads"},
					{Name: "裏", Value: "tails"},
				},
			},
		},
	}
}

func (c *CoinflipCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bet := i.ApplicationCommandData().Options[0].IntValue()
	choice := i.ApplicationCommandData().Options[1].StringValue()
	userID := i.Member.User.ID
	guildID := i.GuildID

	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for coinflip", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if casinoData.Chips < bet {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", casinoData.Chips))
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial coinflip response", "error", err)
		return
	}

	// Animation
	animationEmbed := &discordgo.MessageEmbed{
		Title:       "🪙 コインをトス！",
		Description: "コインは空高く舞い上がった...",
		Color:       0x3498db, // Blue
		Image: &discordgo.MessageEmbedImage{
			URL: "https://i.gifer.com/origin/29/29c53ea25c31332a80a8035463734a18_w200.gif",
		},
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{animationEmbed}}); err != nil {
		c.Log.Error("Failed to edit animation embed", "error", err)
	}
	time.Sleep(3 * time.Second)

	rand.Seed(time.Now().UnixNano())
	result := "tails"
	if rand.Intn(2) == 0 {
		result = "heads"
	}

	// First, subtract the bet amount
	casinoData.Chips -= bet

	won := result == choice

	if won {
		// On win, add double the bet (bet back + winnings)
		casinoData.Chips += bet * 2
	}

	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data after coinflip", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	resultEmbed := &discordgo.MessageEmbed{}
	if won {
		resultEmbed.Title = "🎉 勝利！"
		resultEmbed.Description = fmt.Sprintf("コインは **%s** でした！\n**%d** チップを獲得しました！", translateChoice(result), bet)
		resultEmbed.Color = 0x2ecc71 // Green
	} else {
		resultEmbed.Title = "😥 敗北..."
		resultEmbed.Description = fmt.Sprintf("コインは **%s** でした...\n**%d** チップを失いました。", translateChoice(result), bet)
		resultEmbed.Color = 0xe74c3c // Red
	}
	resultEmbed.Fields = []*discordgo.MessageEmbedField{
		{Name: "現在のチップ", Value: fmt.Sprintf("%d", casinoData.Chips)},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{resultEmbed}}); err != nil {
		c.Log.Error("Failed to edit final coinflip response", "error", err)
	}
}

func translateChoice(choice string) string {
	if choice == "heads" {
		return "表"
	}
	return "裏"
}

func (c *CoinflipCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *CoinflipCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *CoinflipCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *CoinflipCommand) GetCategory() string                                                  { return "カジノ" }
