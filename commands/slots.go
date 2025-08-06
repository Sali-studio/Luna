package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SlotsCommand handles the /slots command.
type SlotsCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

var (
	reels = [][]string{
		{"🍒", "🍋", "🍊", "🍉", "🍇", "🍓", "💎"}, // Reel 1
		{"🍒", "🍋", "🍊", "🍉", "🍇", "🍓", "💎"}, // Reel 2
		{"🍒", "🍋", "🍊", "🍉", "🍇", "🍓", "💎"}, // Reel 3
	}
	payouts = map[string]int{
		"💎💎💎": 50,
		"🍇🍇🍇": 20,
		"🍓🍓🍓": 15,
		"🍉🍉🍉": 10,
		"🍊🍊🍊": 8,
		"🍋🍋🍋": 5,
		"🍒🍒🍒": 3,
	}
)

func (c *SlotsCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "slots",
		Description: "スロットを回してチップを増やそう！",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "bet",
				Description: "ベットするチップの額",
				Required:    true,
				MinValue:    &[]float64{1}[0],
			},
		},
	}
}

func (c *SlotsCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bet := i.ApplicationCommandData().Options[0].IntValue()
	userID := i.Member.User.ID
	guildID := i.GuildID

	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for slots", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if casinoData.Chips < bet {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", casinoData.Chips))
		return
	}

	// Initial response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial slots response", "error", err)
		return
	}

	rand.Seed(time.Now().UnixNano())

	// --- Animation --- 
	finalResult := []string{
		reels[0][rand.Intn(len(reels[0]))],
		reels[1][rand.Intn(len(reels[1]))],
		reels[2][rand.Intn(len(reels[2]))],
	}

	// 1. Fast spinning animation
	animationEmbed := &discordgo.MessageEmbed{
		Title: "🎰 スロット回転中...",
		Color: 0x3498db, // Blue
	}
	for j := 0; j < 5; j++ { // Spin for a short duration
		r1 := reels[0][rand.Intn(len(reels[0]))]
		r2 := reels[1][rand.Intn(len(reels[1]))]
		r3 := reels[2][rand.Intn(len(reels[2]))]
		animationEmbed.Description = fmt.Sprintf("**[ %s | %s | %s ]**", r1, r2, r3)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{animationEmbed}}); err != nil {
			c.Log.Error("Failed to edit animation embed", "error", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 2. Reach animation (stop one by one)
	stoppedReels := []string{"❓", "❓", "❓"}
	for reelIndex := 0; reelIndex < 3; reelIndex++ {
		stoppedReels[reelIndex] = finalResult[reelIndex]
		animationEmbed.Description = fmt.Sprintf("**[ %s | %s | %s ]**", stoppedReels[0], stoppedReels[1], stoppedReels[2])
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{animationEmbed}}); err != nil {
			c.Log.Error("Failed to edit reach embed", "error", err)
		}
		time.Sleep(1 * time.Second) // Pause for dramatic effect
	}
	// --- End Animation ---

	resultStr := strings.Join(finalResult, "")

	// Calculate winnings
	winnings := 0
	payout, won := payouts[resultStr]
	if won {
		winnings = int(bet) * payout
		casinoData.Chips += int64(winnings) - bet // Correctly add net winnings
	} else {
		casinoData.Chips -= bet
	}

	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data after slots", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	// Final result embed
	resultEmbed := &discordgo.MessageEmbed{
		Title: "🎰 スロット結果！",
		Description: fmt.Sprintf("**[ %s | %s | %s ]**", finalResult[0], finalResult[1], finalResult[2]),
	}

	if won {
		resultEmbed.Color = 0x2ecc71 // Green
		resultEmbed.Fields = []*discordgo.MessageEmbedField{
			{Name: "結果", Value: fmt.Sprintf("🎉 おめでとうございます！ **%d** チップを獲得しました！", winnings)},
			{Name: "現在のチップ", Value: fmt.Sprintf("%d", casinoData.Chips)},
		}
	} else {
		resultEmbed.Color = 0xe74c3c // Red
		resultEmbed.Fields = []*discordgo.MessageEmbedField{
			{Name: "結果", Value: "残念、ハズレです..."},
			{Name: "現在のチップ", Value: fmt.Sprintf("%d", casinoData.Chips)},
		}
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{resultEmbed}}); err != nil {
		c.Log.Error("Failed to edit final slots response", "error", err)
	}
}

func (c *SlotsCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *SlotsCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *SlotsCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *SlotsCommand) GetCategory() string                                                  { return "カジノ" }
