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
	// Reels are now weighted. More common symbols appear more frequently.
	reels = [][]string{
		{"🍒", "🍒", "🍒", "🍋", "🍋", "🍊", "🍊", "🍉", "🍇", "🍓", "💎"}, // Reel 1
		{"🍒", "🍒", "🍋", "🍋", "🍋", "🍊", "🍊", "🍉", "🍉", "🍇", "🍓", "💎"}, // Reel 2
		{"🍒", "🍒", "🍒", "🍋", "🍋", "🍊", "🍉", "🍉", "🍇", "🍇", "🍓", "💎"}, // Reel 3
	}
	payouts = map[string]int{
		"💎💎💎": 50, // This is now the jackpot trigger, so the direct payout is less important.
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

	// --- Debit user's bet and contribute to jackpot BEFORE animation ---
	casinoData.Chips -= bet
	jackpotContribution := int64(float64(bet) * 0.01)
	if jackpotContribution < 1 {
		jackpotContribution = 1
	}

	// Update the user's balance first to prevent race conditions
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data on bet", "error", err)
		sendErrorResponse(s, i, "ベット処理中にエラーが発生しました。")
		return
	}

	// Now, add to the jackpot
	currentJackpot, err := c.Store.AddToJackpot(guildID, jackpotContribution)
	if err != nil {
		c.Log.Error("Failed to add to jackpot", "error", err)
		// If this fails, we'll just use the last known jackpot value for display
		currentJackpot, _ = c.Store.GetJackpot(guildID)
	}

	// Initial response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial slots response", "error", err)
		// Attempt to refund the bet if we can't even start the animation
		casinoData.Chips += bet
		c.Store.UpdateCasinoData(casinoData)
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

	// --- Payout Calculation ---
	var winnings int64 = 0
	won := false
	jackpotWon := false
	winDescription := ""

	// 1. Check for Jackpot
	if resultStr == "💎💎💎" {
		won = true
		jackpotWon = true
		winDescription = "👑 JACKPOT! 👑"
		winnings = int64(currentJackpot)
		casinoData.Chips += winnings
		if err := c.Store.UpdateJackpot(guildID, 0); err != nil {
			c.Log.Error("Failed to reset jackpot", "error", err)
		}
	} else if p, ok := payouts[resultStr]; ok {
		// 2. Check for 3-of-a-kind
		won = true
		winDescription = fmt.Sprintf("%s 揃い！", resultStr)
		winnings = bet * int64(p)
		casinoData.Chips += winnings
	} else {
		// 3. Check for small wins (2 cherries)
		cherryCount := 0
		for _, s := range finalResult {
			if s == "🍒" {
				cherryCount++
			}
		}

		if cherryCount == 2 {
			won = true
			winDescription = "🍒 チェリー2つ！"
			winnings = bet * 2 // 2x payout for 2 cherries
			casinoData.Chips += winnings
		}
	}

	// If there were winnings, update the database again
	if won {
		if err := c.Store.UpdateCasinoData(casinoData); err != nil {
			c.Log.Error("Failed to update casino data after slots win", "error", err)
		}
	}

	// Final result embed
	resultEmbed := &discordgo.MessageEmbed{
		Title:       "🎰 スロット結果！",
		Description: fmt.Sprintf("**[ %s | %s | %s ]**", finalResult[0], finalResult[1], finalResult[2]),
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("現在のジャックポット: %d チップ", currentJackpot)},
	}

	if jackpotWon {
		resultEmbed.Color = 0xFFD700 // Gold for Jackpot
		resultEmbed.Title = winDescription
		resultEmbed.Fields = []*discordgo.MessageEmbedField{
			{Name: "ベット", Value: fmt.Sprintf("`%d` チップ", bet), Inline: true},
			{Name: "ジャックポット獲得！", Value: fmt.Sprintf("`%d` チップ", winnings), Inline: true},
			{Name: "💰 所持チップ", Value: fmt.Sprintf("**%d**", casinoData.Chips)},
		}
	} else if won {
		profit := winnings - bet
		resultEmbed.Color = 0x2ecc71 // Green
		resultEmbed.Title = winDescription
		resultEmbed.Fields = []*discordgo.MessageEmbedField{
			{Name: "ベット", Value: fmt.Sprintf("`%d` チップ", bet), Inline: true},
			{Name: "配当", Value: fmt.Sprintf("`%d` チップ", winnings), Inline: true},
			{Name: "収支", Value: fmt.Sprintf("**`+%d`** チップ", profit), Inline: true},
			{Name: "💰 所持チップ", Value: fmt.Sprintf("**%d**", casinoData.Chips)},
		}
	} else {
		resultEmbed.Color = 0xe74c3c // Red
		resultEmbed.Fields = []*discordgo.MessageEmbedField{
			{Name: "ベット", Value: fmt.Sprintf("`%d` チップ", bet), Inline: true},
			{Name: "収支", Value: fmt.Sprintf("**`-%d`** チップ", bet), Inline: true},
			{Name: "💰 所持チップ", Value: fmt.Sprintf("**%d**", casinoData.Chips)},
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

func (c *SlotsCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *SlotsCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *SlotsCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *SlotsCommand) GetCategory() string                                                  { return "カジノ" }
