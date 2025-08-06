package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// --- Constants ---
const (
	BlackjackHitButton   = "bj_hit"
	BlackjackStandButton = "bj_stand"
)

// --- Data Structures ---

// Card represents a single playing card.
type Card struct {
	Suit string
	Rank string
}

// BlackjackGameState represents the state of a blackjack game.
type BlackjackGameState int

const (
	BJStatePlayerTurn BlackjackGameState = iota
	BJStateDealerTurn
	BJStateFinished
)

// BlackjackGame holds the state of a single game.
type BlackjackGame struct {
	State       BlackjackGameState
	PlayerID    string
	Interaction *discordgo.Interaction
	Deck        []Card
	PlayerHand  []Card
	DealerHand  []Card
	BetAmount   int64
}

// BlackjackCommand handles the /blackjack command.
type BlackjackCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
	games map[string]*BlackjackGame // channelID -> game
	mu    sync.Mutex
}

// --- Constructor ---

func NewBlackjackCommand(store interfaces.DataStore, log interfaces.Logger) *BlackjackCommand {
	return &BlackjackCommand{
		Store: store,
		Log:   log,
		games: make(map[string]*BlackjackGame),
	}
}

// --- Command Definition ---

func (c *BlackjackCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "blackjack",
		Description: "ディーラーとブラックジャックで勝負します。",
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

// --- Handlers ---

func (c *BlackjackCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	// Check for existing game in channel
	if _, exists := c.games[i.ChannelID]; exists {
		c.mu.Unlock()
		sendBlackjackErrorResponse(s, i, "このチャンネルでは既にブラックジャックが進行中です。")
		return
	}
	c.mu.Unlock()

	betAmount := i.ApplicationCommandData().Options[0].IntValue()
	userID := i.Member.User.ID

	// Check user's balance
	casinoData, err := c.Store.GetCasinoData(i.GuildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for blackjack", "error", err)
		sendBlackjackErrorResponse(s, i, "エラーが発生しました。")
		return
	}
	if casinoData.Chips < betAmount {
		sendBlackjackErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", casinoData.Chips))
		return
	}

	// Deduct bet amount
	casinoData.Chips -= betAmount
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data on bet", "error", err)
		sendBlackjackErrorResponse(s, i, "ベット処理中にエラーが発生しました。")
		return
	}

	// Create a new game
	deck := NewDeck()
	ShuffleDeck(deck)

	game := &BlackjackGame{
		State:       BJStatePlayerTurn,
		PlayerID:    userID,
		Interaction: i.Interaction,
		Deck:        deck,
		PlayerHand:  []Card{deck[0], deck[2]},
		DealerHand:  []Card{deck[1], deck[3]},
		BetAmount:   betAmount,
	}

	c.mu.Lock()
	c.games[i.ChannelID] = game
	c.mu.Unlock()

	// Send initial game embed
	embed := c.buildGameEmbed(game, "あなたのターン")
	components := c.buildGameComponents(game)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		c.Log.Error("Failed to send blackjack initial message", "error", err)
		// Rollback bet if initial message fails
		casinoData.Chips += betAmount
		c.Store.UpdateCasinoData(casinoData)
		return
	}

	// Check for initial blackjack
	playerValue, playerBlackjack := CalculateHandValue(game.PlayerHand)
	dealerValue, dealerBlackjack := CalculateHandValue(game.DealerHand)

	if playerBlackjack || dealerBlackjack {
		time.AfterFunc(1*time.Second, func() {
			c.determineWinner(s, game, playerValue, dealerValue)
		})
	}
}

func (c *BlackjackCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	game, exists := c.games[i.ChannelID]
	if !exists || i.Member.User.ID != game.PlayerID {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	// Defer the response to avoid timeout
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})

	customID := i.MessageComponentData().CustomID

	switch customID {
	case BlackjackHitButton:
		c.handleHit(s, game)
	case BlackjackStandButton:
		c.handleStand(s, game)
	}
}

func (c *BlackjackCommand) handleHit(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != BJStatePlayerTurn {
		return
	}

	// Deal a new card
	game.PlayerHand = append(game.PlayerHand, game.Deck[0])
	game.Deck = game.Deck[1:]

	playerValue, _ := CalculateHandValue(game.PlayerHand)

	// Update message
	embed := c.buildGameEmbed(game, "あなたのターン")
	components := c.buildGameComponents(game)
	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit blackjack message on hit", "error", err)
	}

	// Check for bust
	if playerValue > 21 {
		time.AfterFunc(1*time.Second, func() {
			c.determineWinner(s, game, playerValue, 0)
		})
	}
}

func (c *BlackjackCommand) handleStand(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != BJStatePlayerTurn {
		return
	}

	game.State = BJStateDealerTurn

	// Reveal dealer's hand and start their turn
	embed := c.buildGameEmbed(game, "ディーラーのターン")
	components := c.buildGameComponents(game) // Disable buttons
	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit blackjack message on stand", "error", err)
	}

	// Dealer plays
	go func() {
		time.Sleep(1 * time.Second)
		dealerValue, _ := CalculateHandValue(game.DealerHand)
		for dealerValue < 17 {
			time.Sleep(1 * time.Second)
			c.mu.Lock()
			game.DealerHand = append(game.DealerHand, game.Deck[0])
			game.Deck = game.Deck[1:]
			dealerValue, _ = CalculateHandValue(game.DealerHand)
			embed := c.buildGameEmbed(game, "ディーラーのターン")
			_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				c.Log.Error("Failed to edit blackjack message on dealer hit", "error", err)
			}
			c.mu.Unlock()
		}

		time.Sleep(1 * time.Second)
		playerValue, _ := CalculateHandValue(game.PlayerHand)
		c.determineWinner(s, game, playerValue, dealerValue)
	}()
}

func (c *BlackjackCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) { /* No modal for now */ }

func (c *BlackjackCommand) GetCategory() string {
	return "カジノ"
}

func (c *BlackjackCommand) GetComponentIDs() []string {
	return []string{BlackjackHitButton, BlackjackStandButton}
}

// --- Game Logic ---

var suits = []string{"♠️", "♥️", "♦️", "♣️"}
var ranks = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

func NewDeck() []Card {
	deck := make([]Card, 0, 52)
	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{Suit: suit, Rank: rank})
		}
	}
	return deck
}

func ShuffleDeck(deck []Card) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

func (c *Card) String() string {
	return c.Suit + " " + c.Rank
}

func HandToString(hand []Card, hideFirst bool) string {
	if len(hand) == 0 {
		return ""
	}
	if hideFirst {
		if len(hand) > 1 {
			return "[ ?? ] | " + hand[1].String()
		}
		return "[ ?? ]"
	}
	var parts []string
	for _, card := range hand {
		parts = append(parts, card.String())
	}
	return strings.Join(parts, " | ")
}

func CalculateHandValue(hand []Card) (int, bool) {
	value := 0
	aces := 0
	for _, card := range hand {
		switch card.Rank {
		case "A":
			aces++
			value += 11
		case "K", "Q", "J":
			value += 10
		default:
			rankValue, err := strconv.Atoi(card.Rank)
			if err == nil {
				value += rankValue
			}
		}
	}

	for value > 21 && aces > 0 {
		value -= 10
		aces--
	}

	return value, len(hand) == 2 && value == 21
}

// --- Helper Functions ---

func (c *BlackjackCommand) buildGameEmbed(game *BlackjackGame, title string) *discordgo.MessageEmbed {
	playerValue, _ := CalculateHandValue(game.PlayerHand)

	var dealerHandStr string
	var dealerValue int

	if game.State == BJStatePlayerTurn {
		dealerHandStr = HandToString(game.DealerHand, true)
		if len(game.DealerHand) > 1 {
			dealerValue, _ = CalculateHandValue([]Card{game.DealerHand[1]})
		}
	} else {
		dealerHandStr = HandToString(game.DealerHand, false)
		dealerValue, _ = CalculateHandValue(game.DealerHand)
	}

	return &discordgo.MessageEmbed{
		Title:       "♠️♥️ ブラックジャック ♦️♣️",
		Description: fmt.Sprintf("ベット額: **%d** チップ", game.BetAmount),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   fmt.Sprintf("ディーラーの手札 (%d)", dealerValue),
				Value:  dealerHandStr,
				Inline: false,
			},
			{
				Name:   fmt.Sprintf("あなたの手札 (%d)", playerValue),
				Value:  HandToString(game.PlayerHand, false),
				Inline: false,
			},
		},
		Color:  0x000000,
		Footer: &discordgo.MessageEmbedFooter{Text: title},
	}
}

func (c *BlackjackCommand) buildGameComponents(game *BlackjackGame) []discordgo.MessageComponent {
	disabled := game.State != BJStatePlayerTurn
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "ヒット",
					Style:    discordgo.SuccessButton,
					CustomID: BlackjackHitButton,
					Disabled: disabled,
				},
				discordgo.Button{
					Label:    "スタンド",
					Style:    discordgo.DangerButton,
					CustomID: BlackjackStandButton,
					Disabled: disabled,
				},
			},
		},
	}
}

func (c *BlackjackCommand) determineWinner(s *discordgo.Session, game *BlackjackGame, playerValue int, dealerValue int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State == BJStateFinished {
		return
	}
	game.State = BJStateFinished

	_, playerBlackjack := CalculateHandValue(game.PlayerHand)
	_, dealerBlackjack := CalculateHandValue(game.DealerHand)

	var resultText string
	var payout int64

	if playerBlackjack && !dealerBlackjack {
		resultText = "ブラックジャック！あなたの勝ちです！🎉"
		payout = int64(float64(game.BetAmount) * 2.5)
	} else if playerValue > 21 {
		resultText = "バスト！あなたの負けです...😢"
		payout = 0
	} else if dealerValue > 21 {
		resultText = "ディーラーがバスト！あなたの勝ちです！🥳"
		payout = game.BetAmount * 2
	} else if playerValue > dealerValue {
		resultText = "あなたの勝ちです！😄"
		payout = game.BetAmount * 2
	} else if playerValue < dealerValue {
		resultText = "あなたの負けです...😭"
		payout = 0
	} else { // Push
		resultText = "引き分け（プッシュ）です。ベット額が返却されます。😐"
		payout = game.BetAmount
	}

	if payout > 0 {
		casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, game.PlayerID)
		if err == nil {
			casinoData.Chips += payout
			c.Store.UpdateCasinoData(casinoData)
		}
	}

	embed := c.buildGameEmbed(game, resultText)
	components := c.buildGameComponents(game)

	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit blackjack final message", "error", err)
	}

	delete(c.games, game.Interaction.ChannelID)
}

func sendBlackjackErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}