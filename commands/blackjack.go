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
	BlackjackHitButton       = "bj_hit"
	BlackjackStandButton     = "bj_stand"
	BlackjackDoubleDownButton  = "bj_double_down"
	BlackjackSplitButton     = "bj_split"
	BlackjackInsuranceButton = "bj_insurance"
	BlackjackSurrenderButton = "bj_surrender"
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
	State         BlackjackGameState
	PlayerID      string
	Interaction   *discordgo.Interaction
	Deck          []Card
	PlayerHand    []Card
	PlayerHand2   []Card // For split
	DealerHand    []Card
	BetAmount     int64
	BetAmount2    int64 // For split
	InsuranceBet  int64
	CurrentHand   int // 1 or 2, for split
	CanDoubleDown bool
	CanSplit      bool
	CanSurrender  bool
	rand          *rand.Rand // Game-specific random source
}

// BlackjackCommand handles the /blackjack command.
type BlackjackCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
	games map[string]*BlackjackGame // userID -> game
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
	userID := i.Member.User.ID

	c.mu.Lock()
	// Check for existing game for the user
	if _, exists := c.games[userID]; exists {
		c.mu.Unlock()
		sendBlackjackErrorResponse(s, i, "既にブラックジャックのゲームが進行中です。まずはそれを終了してください。")
		return
	}
	c.mu.Unlock()

	betAmount := i.ApplicationCommandData().Options[0].IntValue()

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
	gameRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	deck := NewDeck()
	ShuffleDeck(deck, gameRand)

	game := &BlackjackGame{
		State:         BJStatePlayerTurn,
		PlayerID:      userID,
		Interaction:   i.Interaction,
		Deck:          deck,
		PlayerHand:    make([]Card, 0, 5),
		DealerHand:    make([]Card, 0, 5),
		BetAmount:     betAmount,
		CurrentHand:   1,
		rand:          gameRand,
	}

	// Deal initial cards
	dealCard(game, &game.PlayerHand)
	dealCard(game, &game.DealerHand)
	dealCard(game, &game.PlayerHand)
	dealCard(game, &game.DealerHand)

	// Check for split and double down options
	game.CanSplit = len(game.PlayerHand) == 2 && game.PlayerHand[0].Rank == game.PlayerHand[1].Rank && casinoData.Chips >= betAmount
	playerValue, _ := CalculateHandValue(game.PlayerHand)
	game.CanDoubleDown = len(game.PlayerHand) == 2 && (playerValue == 9 || playerValue == 10 || playerValue == 11) && casinoData.Chips >= betAmount
	game.CanSurrender = len(game.PlayerHand) == 2

	c.mu.Lock()
	c.games[userID] = game
	c.mu.Unlock()

	// Send initial game embed as a public message
	embed := c.buildGameEmbed(game, "あなたのターン")
	components := c.buildGameComponents(game)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			// Flags:      discordgo.MessageFlagsEphemeral, // Removed this line to make it public
		},
	})
	if err != nil {
		c.Log.Error("Failed to send blackjack initial message", "error", err)
		// Rollback bet if initial message fails
		casinoData.Chips += betAmount
		c.Store.UpdateCasinoData(casinoData)
		return
	}

	// Check for insurance option or initial blackjacks
	dealerUpCardIsAce := game.DealerHand[1].Rank == "A"
	_, playerBlackjack := CalculateHandValue(game.PlayerHand)
	_, dealerBlackjack := CalculateHandValue(game.DealerHand)

	if !dealerUpCardIsAce && (playerBlackjack || dealerBlackjack) {
		// If no insurance is offered and someone has blackjack, end the game immediately.
		time.AfterFunc(1*time.Second, func() {
			c.determineWinner(s, game)
		})
	} else if dealerUpCardIsAce {
		// If dealer has an Ace up, the game continues to allow for insurance bets.
		// If the player also has blackjack, it's an even money situation, but we handle that in determineWinner.
	}
}

func (c *BlackjackCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// We need to find which game this component interaction belongs to.
	// Since multiple games can be active in a channel, we can't just use the channel ID.
	// We will find the game based on the user who is interacting.
	c.mu.Lock()
	game, exists := c.games[i.Member.User.ID]
	c.mu.Unlock()

	// If the game doesn't exist for this user, it might be another user's game.
	if !exists {
		// Check if this interaction is for ANY active game by checking the original message author.
		// This is a bit complex, so for now, we will just send an ephemeral error.
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "これはあなたのゲームではありません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Defer the response to avoid timeout
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})

	customID := i.MessageComponentData().CustomID

	switch customID {
	case BlackjackHitButton:
		c.handleHit(s, game)
	case BlackjackStandButton:
		c.handleStand(s, game)
	case BlackjackDoubleDownButton:
		c.handleDoubleDown(s, game)
	case BlackjackSplitButton:
		c.handleSplit(s, game)
	case BlackjackInsuranceButton:
		c.handleInsurance(s, game)
	case BlackjackSurrenderButton:
		c.handleSurrender(s, game)
	}
}

func (c *BlackjackCommand) handleHit(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != BJStatePlayerTurn {
		return
	}

	// Disable special moves after hitting
	game.CanDoubleDown = false
	game.CanSplit = false
	game.CanSurrender = false

	// Determine which hand to hit
	hand := &game.PlayerHand
	if game.CurrentHand == 2 {
		hand = &game.PlayerHand2
	}

	// Deal a new card
	dealCard(game, hand)

	playerValue, _ := CalculateHandValue(*hand)

	// Check for bust
	if playerValue > 21 {
		// If it was the first hand of a split, move to the next hand
		if game.CurrentHand == 1 && len(game.PlayerHand2) > 0 {
			game.CurrentHand = 2
			// Update message and return
			embed := c.buildGameEmbed(game, "あなたのターン (2つ目の手)")
			components := c.buildGameComponents(game)
			_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
				Embeds:     &[]*discordgo.MessageEmbed{embed},
				Components: &components,
			})
			if err != nil {
				c.Log.Error("Failed to edit blackjack message on split bust", "error", err)
			}
			return
		} else {
			// If not a split or it was the second hand, end the game
			time.AfterFunc(1*time.Second, func() {
				c.determineWinner(s, game)
			})
			return // Return to prevent updating the message twice
		}
	}

	// Update message
	title := "あなたのターン"
	if len(game.PlayerHand2) > 0 {
		title = fmt.Sprintf("あなたのターン (%dつ目の手)", game.CurrentHand)
	}
	embed := c.buildGameEmbed(game, title)
	components := c.buildGameComponents(game)
	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit blackjack message on hit", "error", err)
	}
}

func (c *BlackjackCommand) handleStand(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()

	if game.State != BJStatePlayerTurn {
		c.mu.Unlock()
		return
	}

	// If it's the first hand of a split, move to the second hand
	if game.CurrentHand == 1 && len(game.PlayerHand2) > 0 {
		game.CurrentHand = 2
		c.mu.Unlock()

		// Update the UI for the second hand
		embed := c.buildGameEmbed(game, "あなたのターン (2つ目の手)")
		components := c.buildGameComponents(game)
		_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		if err != nil {
			c.Log.Error("Failed to edit blackjack message on split stand", "error", err)
		}
		return
	}

	// If not a split or it's the second hand, proceed to the dealer's turn
	game.State = BJStateDealerTurn
	c.mu.Unlock()

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
			if game.State == BJStateFinished { // Check if game ended while sleeping
				c.mu.Unlock()
				return
			}
			dealCard(game, &game.DealerHand)
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
		c.determineWinner(s, game)
	}()
}

func (c *BlackjackCommand) handleDoubleDown(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != BJStatePlayerTurn || !game.CanDoubleDown {
		return
	}

	// Double the bet
	casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, game.PlayerID)
	if err != nil || casinoData.Chips < game.BetAmount {
		// Not enough chips, can't double down. Silently ignore.
		return
	}
	casinoData.Chips -= game.BetAmount
	c.Store.UpdateCasinoData(casinoData)
	game.BetAmount *= 2

	// Deal one more card
	dealCard(game, &game.PlayerHand)

	// End player's turn
	game.State = BJStateDealerTurn

	// Update UI and start dealer's turn after a delay
	embed := c.buildGameEmbed(game, "ダブルダウン！ディーラーのターン")
	components := c.buildGameComponents(game)
	_, err = s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit message on double down", "error", err)
	}

	go func() {
		time.Sleep(2 * time.Second)
		c.determineWinner(s, game)
	}()
}

func (c *BlackjackCommand) handleSplit(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != BJStatePlayerTurn || !game.CanSplit {
		return
	}

	// Check if user has enough chips to split
	casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, game.PlayerID)
	if err != nil || casinoData.Chips < game.BetAmount {
		// Not enough chips, can't split. Silently ignore.
		return
	}
	casinoData.Chips -= game.BetAmount
	c.Store.UpdateCasinoData(casinoData)

	// Split the hand
	game.PlayerHand2 = []Card{game.PlayerHand[1]}
	game.PlayerHand = []Card{game.PlayerHand[0]}
	game.BetAmount2 = game.BetAmount

	// Deal a new card to each hand
	dealCard(game, &game.PlayerHand)
	dealCard(game, &game.PlayerHand2)

	// Disable further splitting or doubling for now
	game.CanSplit = false
	game.CanDoubleDown = false

	// Update UI
	embed := c.buildGameEmbed(game, "スプリット！あなたのターン (1つ目の手)")
	components := c.buildGameComponents(game)
	_, err = s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit message on split", "error", err)
	}
}

func (c *BlackjackCommand) handleInsurance(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != BJStatePlayerTurn || game.DealerHand[0].Rank != "A" || game.InsuranceBet > 0 {
		return
	}

	insuranceAmount := game.BetAmount / 2
	casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, game.PlayerID)
	if err != nil || casinoData.Chips < insuranceAmount {
		// Not enough chips for insurance. Silently ignore.
		return
	}

	casinoData.Chips -= insuranceAmount
	c.Store.UpdateCasinoData(casinoData)
	game.InsuranceBet = insuranceAmount

	// Update UI to show insurance was taken
	embed := c.buildGameEmbed(game, "インシュランスを受け付けました。あなたのターン")
	components := c.buildGameComponents(game)
	_, err = s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit message on insurance", "error", err)
	}

	// Check if dealer has blackjack immediately
	_, dealerBlackjack := CalculateHandValue(game.DealerHand)
	if dealerBlackjack {
		time.AfterFunc(1*time.Second, func() {
			c.determineWinner(s, game)
		})
	}
}

func (c *BlackjackCommand) handleSurrender(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != BJStatePlayerTurn || !game.CanSurrender {
		return
	}

	game.State = BJStateFinished

	// Refund half of the bet
	refund := game.BetAmount / 2
	casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, game.PlayerID)
	if err == nil {
		casinoData.Chips += refund
		c.Store.UpdateCasinoData(casinoData)
	}

	// Update UI to show surrender result
	embed := c.buildGameEmbed(game, "サレンダー")
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:  "結果",
		Value: fmt.Sprintf("サレンダーしました。ベットの半分 (**%d** チップ) が返却されます。", refund),
	})
	components := c.buildGameComponents(game) // Disable buttons

	_, err = s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit message on surrender", "error", err)
	}

	delete(c.games, game.PlayerID)
}

func (c *BlackjackCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) { /* No modal for now */ }

func (c *BlackjackCommand) GetCategory() string {
	return "カジノ"
}

func (c *BlackjackCommand) GetComponentIDs() []string {
	return []string{BlackjackHitButton, BlackjackStandButton, BlackjackDoubleDownButton, BlackjackSplitButton, BlackjackInsuranceButton, BlackjackSurrenderButton}
}

// --- Game Logic ---

var suits = []string{"♠️", "♥️", "♦️", "♣️"}
var ranks = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

func NewDeck() []Card {
	deck := make([]Card, 0, 52*6) // Use 6 decks
	for i := 0; i < 6; i++ {
		for _, suit := range suits {
			for _, rank := range ranks {
				deck = append(deck, Card{Suit: suit, Rank: rank})
			}
		}
	}
	return deck
}

func ShuffleDeck(deck []Card, r *rand.Rand) {
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

// dealCard deals one card from the deck to the specified hand.
func dealCard(game *BlackjackGame, hand *[]Card) {
	*hand = append(*hand, game.Deck[0])
	game.Deck = game.Deck[1:]
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
	playerValue2, _ := CalculateHandValue(game.PlayerHand2)

	var dealerHandStr string
	var dealerValue int

	if game.State == BJStatePlayerTurn {
		dealerHandStr = HandToString(game.DealerHand, true)
		if len(game.DealerHand) > 1 {
			// Show only the value of the up-card
			dealerValue, _ = CalculateHandValue([]Card{game.DealerHand[1]})
		}
	} else {
		dealerHandStr = HandToString(game.DealerHand, false)
		dealerValue, _ = CalculateHandValue(game.DealerHand)
	}

	description := fmt.Sprintf("ベット額: **%d** チップ", game.BetAmount)
	if game.InsuranceBet > 0 {
		description += fmt.Sprintf(" | インシュランス: **%d** チップ", game.InsuranceBet)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "♠️♥️ ブラックジャック ♦️♣️",
		Description: description,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   fmt.Sprintf("ディーラーの手札 (%d)", dealerValue),
				Value:  dealerHandStr,
				Inline: false,
			},
		},
		Color:  0x000000,
		Footer: &discordgo.MessageEmbedFooter{Text: title},
	}

	// Add player hand fields
	playerHandName := "あなたの手札"
	if len(game.PlayerHand2) > 0 {
		playerHandName = "あなたの手札 (1)"
		if game.CurrentHand == 1 {
			playerHandName += " ◀️"
		}
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   fmt.Sprintf("%s (%d)", playerHandName, playerValue),
		Value:  HandToString(game.PlayerHand, false),
		Inline: false,
	})

	if len(game.PlayerHand2) > 0 {
		playerHand2Name := "あなたの手札 (2)"
		if game.CurrentHand == 2 {
			playerHand2Name += " ◀️"
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (%d)", playerHand2Name, playerValue2),
			Value:  HandToString(game.PlayerHand2, false),
			Inline: false,
		})
	}

	return embed
}

func (c *BlackjackCommand) buildGameComponents(game *BlackjackGame) []discordgo.MessageComponent {
	disabled := game.State != BJStatePlayerTurn
	showInsurance := game.DealerHand[0].Rank == "A" && game.InsuranceBet == 0

	// First row of buttons: core actions
	actionsRow1 := discordgo.ActionsRow{
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
	}

	// Second row of buttons: special actions (Double Down, Split)
	var actionsRow2 *discordgo.ActionsRow
	if game.CanDoubleDown || game.CanSplit {
		var specialButtons []discordgo.MessageComponent
		if game.CanDoubleDown {
			specialButtons = append(specialButtons, discordgo.Button{
				Label:    "ダブルダウン",
				Style:    discordgo.PrimaryButton,
				CustomID: BlackjackDoubleDownButton,
				Disabled: disabled,
			})
		}
		if game.CanSplit {
			specialButtons = append(specialButtons, discordgo.Button{
				Label:    "スプリット",
				Style:    discordgo.PrimaryButton,
				CustomID: BlackjackSplitButton,
				Disabled: disabled,
			})
		}
		actionsRow2 = &discordgo.ActionsRow{Components: specialButtons}
	}

	// Third row for Insurance and Surrender
	var actionsRow3 *discordgo.ActionsRow
	var specialButtons2 []discordgo.MessageComponent
	if showInsurance {
		specialButtons2 = append(specialButtons2, discordgo.Button{
			Label:    "インシュランス",
			Style:    discordgo.SecondaryButton,
			CustomID: BlackjackInsuranceButton,
			Disabled: disabled,
		})
	}
	if game.CanSurrender {
		specialButtons2 = append(specialButtons2, discordgo.Button{
			Label:    "サレンダー",
			Style:    discordgo.SecondaryButton,
			CustomID: BlackjackSurrenderButton,
			Disabled: disabled,
		})
	}
	if len(specialButtons2) > 0 {
		actionsRow3 = &discordgo.ActionsRow{Components: specialButtons2}
	}

	var components []discordgo.MessageComponent
	components = append(components, actionsRow1)
	if actionsRow2 != nil {
		components = append(components, *actionsRow2)
	}
	if actionsRow3 != nil {
		components = append(components, *actionsRow3)
	}

	return components
}

func (c *BlackjackCommand) determineWinner(s *discordgo.Session, game *BlackjackGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State == BJStateFinished {
		return
	}
	game.State = BJStateFinished

	_, dealerBlackjack := CalculateHandValue(game.DealerHand)
	var finalResultText strings.Builder
	var totalPayout int64 = 0

	// Handle Insurance Payout
	if game.InsuranceBet > 0 {
		if dealerBlackjack {
			insurancePayout := game.InsuranceBet * 2
			totalPayout += insurancePayout
			finalResultText.WriteString(fmt.Sprintf("✅ **インシュランス成功！** ディーラーはブラックジャックでした。配当 **%d** チップを獲得しました。\n", insurancePayout))
		} else {
			finalResultText.WriteString(fmt.Sprintf("❌ **インシュランス失敗。** ディーラーはブラックジャックではありませんでした。\n"))
		}
	}

	// Determine result for the first hand (and the only hand if not split)
	payout1, resultText1 := c.calculateHandResult(game.PlayerHand, game.DealerHand, game.BetAmount)
	totalPayout += payout1
	finalResultText.WriteString(fmt.Sprintf("**手札1:** %s\n", resultText1))

	// Determine result for the second hand if it exists
	if len(game.PlayerHand2) > 0 {
		payout2, resultText2 := c.calculateHandResult(game.PlayerHand2, game.DealerHand, game.BetAmount2)
		totalPayout += payout2
		finalResultText.WriteString(fmt.Sprintf("**手札2:** %s\n", resultText2))
	}

	// Update user's balance
	if totalPayout > 0 {
		casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, game.PlayerID)
		if err == nil {
			casinoData.Chips += totalPayout
			c.Store.UpdateCasinoData(casinoData)
			finalResultText.WriteString(fmt.Sprintf("\n**合計収支:** `+%d` チップ | **現在の所持チップ:** `%d`", totalPayout-(game.BetAmount+game.BetAmount2), casinoData.Chips))
		} else {
			c.Log.Error("Failed to get casino data for payout", "error", err)
		}
	} else {
		casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, game.PlayerID)
		if err == nil {
			finalResultText.WriteString(fmt.Sprintf("\n**合計収支:** `-%d` チップ | **現在の所持チップ:** `%d`", game.BetAmount+game.BetAmount2, casinoData.Chips))
		}
	}

	embed := c.buildGameEmbed(game, "ゲーム終了")
	// Add a field for the final results
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:  "最終結果",
		Value: finalResultText.String(),
	})

	components := c.buildGameComponents(game) // This will disable all buttons

	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		c.Log.Error("Failed to edit blackjack final message", "error", err)
	}

	delete(c.games, game.PlayerID)
}

// calculateHandResult calculates the payout and result text for a single hand.
func (c *BlackjackCommand) calculateHandResult(playerHand, dealerHand []Card, betAmount int64) (int64, string) {
	playerValue, playerBlackjack := CalculateHandValue(playerHand)
	dealerValue, dealerBlackjack := CalculateHandValue(dealerHand)

	if playerBlackjack && !dealerBlackjack {
		payout := int64(float64(betAmount) * 2.2)
		return payout, fmt.Sprintf("ブラックジャック！あなたの勝ちです！🎉 (配当: %d)", payout)
	} else if playerValue > 21 {
		return 0, "バスト！あなたの負けです...😢"
	} else if dealerBlackjack {
		return 0, "ディーラーのブラックジャック！あなたの負けです...😭"
	} else if dealerValue > 21 {
		payout := betAmount * 2
		return payout, fmt.Sprintf("ディーラーがバスト！あなたの勝ちです！🥳 (配当: %d)", payout)
	} else if playerValue > dealerValue {
		payout := betAmount * 2
		return payout, fmt.Sprintf("あなたの勝ちです！😄 (配当: %d)", payout)
	} else if playerValue < dealerValue {
		return 0, "あなたの負けです...😭"
	} else { // Push
		return betAmount, "引き分け（プッシュ）です。ベット額が返却されます。😐"
	}
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