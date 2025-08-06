package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
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
	StatePlayerTurn BlackjackGameState = iota
	StateDealerTurn
	StateFinished
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

// --- Handlers (to be implemented) ---

func (c *BlackjackCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	// Check for existing game in channel
	if _, exists := c.games[i.ChannelID]; exists {
		c.mu.Unlock()
		sendErrorResponse(s, i, "このチャンネルでは既にブラックジャックが進行中です。")
		return
	}
	c.mu.Unlock()

	betAmount := i.ApplicationCommandData().Options[0].IntValue()
	userID := i.Member.User.ID

	// Check user's balance
	casinoData, err := c.Store.GetCasinoData(i.GuildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for blackjack", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}
	if casinoData.Chips < betAmount {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", casinoData.Chips))
		return
	}

	// Deduct bet amount
	casinoData.Chips -= betAmount
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data on bet", "error", err)
		sendErrorResponse(s, i, "ベット処理中にエラーが発生しました。")
		return
	}

	// Create a new game
	deck := NewDeck()
	ShuffleDeck(deck)

	game := &BlackjackGame{
		State:       StatePlayerTurn,
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
	// TODO: Handle Hit and Stand buttons
}

func (c *BlackjackCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) { /* No modal for now */ }

func (c *BlackjackCommand) GetCategory() string {
	return "カジノ"
}

func (c *BlackjackCommand) GetComponentIDs() []string {
	return []string{BlackjackHitButton, BlackjackStandButton}
}

// --- Game Logic (to be implemented) ---

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
		return "[ ?? ] " + hand[1].String()
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
			num, _ := fmt.Sscan(card.Rank, &value)
			value += num
		}
	}

	for value > 21 && aces > 0 {
		value -= 10
		aces--
	}

	return value, len(hand) == 2 && value == 21
}
