package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	HiLowButtonHigh = "hilow_high"
	HiLowButtonLow  = "hilow_low"
)

// HiLowGame holds the state of a single game.
type HiLowGame struct {
	PlayerID    string
	Interaction *discordgo.Interaction
	BetAmount   int64
	FirstCard   int
}

// HiLowCommand handles the /hilow command.
type HiLowCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
	games map[string]*HiLowGame // userID -> game
	mu    sync.Mutex
}

// NewHiLowCommand creates a new HiLowCommand.
func NewHiLowCommand(store interfaces.DataStore, log interfaces.Logger) *HiLowCommand {
	return &HiLowCommand{
		Store: store,
		Log:   log,
		games: make(map[string]*HiLowGame),
	}
}

func (c *HiLowCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "hilow",
		Description: "次のカードがハイかローかを当てる簡単なゲームです。",
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

func (c *HiLowCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	c.mu.Lock()
	if _, exists := c.games[userID]; exists {
		c.mu.Unlock()
		sendErrorResponse(s, i, "既にハイ＆ローのゲームが進行中です。まずはそれを終了してください。")
		return
	}
	c.mu.Unlock()

	betAmount := i.ApplicationCommandData().Options[0].IntValue()

	casinoData, err := c.Store.GetCasinoData(i.GuildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for hilow", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}
	if casinoData.Chips < betAmount {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", casinoData.Chips))
		return
	}

	casinoData.Chips -= betAmount
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data on bet", "error", err)
		sendErrorResponse(s, i, "ベット処理中にエラーが発生しました。")
		return
	}

	rand.Seed(time.Now().UnixNano())
	firstCard := rand.Intn(13) + 1

	game := &HiLowGame{
		PlayerID:    userID,
		Interaction: i.Interaction,
		BetAmount:   betAmount,
		FirstCard:   firstCard,
	}

	c.mu.Lock()
	c.games[userID] = game
	c.mu.Unlock()

	embed := &discordgo.MessageEmbed{
		Title:       "🃏 ハイ＆ロー",
		Description: fmt.Sprintf("最初のカードは **%d** です。\n次のカードはこれより高い(High)か低い(Low)か？", firstCard),
		Color:       0x3498db, // Blue
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("ベット額: %d チップ", betAmount)},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "ハイ (High)",
					Style:    discordgo.SuccessButton,
					CustomID: HiLowButtonHigh,
				},
				discordgo.Button{
					Label:    "ロー (Low)",
					Style:    discordgo.DangerButton,
					CustomID: HiLowButtonLow,
				},
			},
		},
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		c.Log.Error("Failed to send hilow initial message", "error", err)
		casinoData.Chips += betAmount
		c.Store.UpdateCasinoData(casinoData)
	}
}

func (c *HiLowCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	c.mu.Lock()
	game, exists := c.games[userID]
	if !exists {
		c.mu.Unlock()
		sendErrorResponse(s, i, "これはあなたのゲームではありません。")
		return
	}
	c.mu.Unlock()

	playerChoiceIsHigh := i.MessageComponentData().CustomID == HiLowButtonHigh

	rand.Seed(time.Now().UnixNano())
	secondCard := rand.Intn(13) + 1

	var resultText string
	payout := int64(0)
	var won bool

	if (playerChoiceIsHigh && secondCard > game.FirstCard) || (!playerChoiceIsHigh && secondCard < game.FirstCard) {
		won = true
		payout = int64(float64(game.BetAmount) * 1.8)
		resultText = fmt.Sprintf("🎉 **勝ち！** %dチップを獲得しました！", payout)
	} else if secondCard == game.FirstCard {
		won = false // Technically a push, not a win
		payout = game.BetAmount
		resultText = "😐 **引き分け。** ベット額が返却されます。"
	} else {
		won = false
		payout = 0
		resultText = "😭 **負け...** ベット額は没収されます。"
	}

	if payout > 0 {
		casinoData, err := c.Store.GetCasinoData(i.GuildID, userID)
		if err == nil {
			casinoData.Chips += payout
			c.Store.UpdateCasinoData(casinoData)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🃏 ハイ＆ロー - 結果",
		Description: fmt.Sprintf("最初のカード: **%d**\n次のカード: **%d**\n\n%s", game.FirstCard, secondCard, resultText),
		Color:       0x2ecc71, // Green for win/push, should be dynamic
	}

	if !won && payout == 0 {
		embed.Color = 0xe74c3c // Red for loss
	}

	// Disable buttons
	disabledComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "ハイ (High)", Style: discordgo.SuccessButton, CustomID: HiLowButtonHigh, Disabled: true},
				discordgo.Button{Label: "ロー (Low)", Style: discordgo.DangerButton, CustomID: HiLowButtonLow, Disabled: true},
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: disabledComponents,
		},
	})

	c.mu.Lock()
	delete(c.games, userID)
	c.mu.Unlock()
}

func (c *HiLowCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}

func (c *HiLowCommand) GetCategory() string {
	return "カジノ"
}

func (c *HiLowCommand) GetComponentIDs() []string {
	return []string{HiLowButtonHigh, HiLowButtonLow}
}
