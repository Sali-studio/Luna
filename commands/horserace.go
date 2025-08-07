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

const (
	BetButtonPrefix   = "hr_bet_"
	StartRaceButtonID = "hr_start_race"
	BetModalCustomID  = "hr_bet_modal"
	RaceTrackLength   = 20
)

// RaceState はレースの状態を表します。
type RaceState int

const (
	HRStateBetting RaceState = iota
	HRStateRacing
	HRStateFinished
)

// Horse は馬の情報を表します。
type Horse struct {
	Name  string
	Odds  float64
	Emoji string
}

// Bet はベット情報を表します。
type Bet struct {
	UserID     string
	HorseIndex int
	Amount     int64
}

// HorseRaceGame はレースゲーム全体の管理を行います。
type HorseRaceGame struct {
	State       RaceState
	Horses      []Horse
	Bets        []Bet
	MessageID   string
	ChannelID   string
	Interaction *discordgo.Interaction
	CreatorID   string
}

// HorseRaceCommand は /horserace コマンドを処理します。
type HorseRaceCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
	races map[string]*HorseRaceGame // channelID -> game
	mu    sync.Mutex
}

// --- Command/Component/Modal Handlers ---

func NewHorseRaceCommand(store interfaces.DataStore, log interfaces.Logger) *HorseRaceCommand {
	return &HorseRaceCommand{
		Store: store,
		Log:   log,
		races: make(map[string]*HorseRaceGame),
	}
}

func (c *HorseRaceCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "horserace",
		Description: "競馬を開始します！",
	}
}

func (c *HorseRaceCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.races[i.ChannelID]; exists {
		sendErrorResponse(s, i, "このチャンネルでは既にレースが進行中です。")
		return
	}

	game := &HorseRaceGame{
		State:       HRStateBetting,
		ChannelID:   i.ChannelID,
		Interaction: i.Interaction,
		CreatorID:   i.Member.User.ID,
	}

	game.Horses = generateHorses(5)
	embed := c.buildBettingEmbed(game)
	components := c.buildBettingComponents(game)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		c.Log.Error("Failed to send initial race message", "error", err)
		return
	}

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		c.Log.Error("Failed to get interaction response message", "error", err)
		return
	}
	game.MessageID = msg.ID
	c.races[i.ChannelID] = game
}

func (c *HorseRaceCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	game, exists := c.races[i.ChannelID]
	c.mu.Unlock()

	if !exists {
		return
	}

	customID := i.MessageComponentData().CustomID

	if strings.HasPrefix(customID, BetButtonPrefix) {
		c.handleBetButton(s, i, game)
	} else if customID == StartRaceButtonID {
		c.handleStartRaceButton(s, i, game)
	}
}

func (c *HorseRaceCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	game, exists := c.races[i.ChannelID]
	c.mu.Unlock()

	if !exists {
		return
	}

	customID := i.ModalSubmitData().CustomID
	if strings.HasPrefix(customID, BetModalCustomID) {
		c.handleBetModalSubmit(s, i, game)
	}
}

func (c *HorseRaceCommand) GetComponentIDs() []string {
	return []string{BetButtonPrefix, StartRaceButtonID, BetModalCustomID}
}

func (c *HorseRaceCommand) GetCategory() string {
	return "カジノ"
}

// --- Handler Logic ---

func (c *HorseRaceCommand) handleBetButton(s *discordgo.Session, i *discordgo.InteractionCreate, game *HorseRaceGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != HRStateBetting {
		sendErrorResponse(s, i, "ベット受付は終了しました。")
		return
	}

	horseIndexStr := strings.TrimPrefix(i.MessageComponentData().CustomID, BetButtonPrefix)
	horseIndex, _ := strconv.Atoi(horseIndexStr)

	modal := discordgo.InteractionResponseData{
		CustomID: BetModalCustomID + "_" + horseIndexStr,
		Title:    fmt.Sprintf("%s にベット", game.Horses[horseIndex].Name),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "bet_amount",
						Label:       "ベットするチップの額",
						Style:       discordgo.TextInputShort,
						Placeholder: "100",
						Required:    true,
					},
				},
			},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &modal})
}

func (c *HorseRaceCommand) handleStartRaceButton(s *discordgo.Session, i *discordgo.InteractionCreate, game *HorseRaceGame) {
	if i.Member.User.ID != game.CreatorID {
		sendErrorResponse(s, i, "レースを開始できるのは、レースを開始した本人だけです。")
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	go c.startRace(s, game)
}

func (c *HorseRaceCommand) handleBetModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, game *HorseRaceGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	customID := i.ModalSubmitData().CustomID
	horseIndexStr := strings.TrimPrefix(customID, BetModalCustomID+"_")
	horseIndex, _ := strconv.Atoi(horseIndexStr)
	betAmountStr := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	betAmount, err := strconv.ParseInt(betAmountStr, 10, 64)

	if err != nil || betAmount <= 0 {
		sendErrorResponse(s, i, "有効なベット額を入力してください。")
		return
	}

	userID := i.Member.User.ID
	casinoData, err := c.Store.GetCasinoData(i.GuildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for bet", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if casinoData.Chips < betAmount {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", casinoData.Chips))
		return
	}

	// Subtract the bet amount from the user's balance first
	casinoData.Chips -= betAmount
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data on bet", "error", err)
		sendErrorResponse(s, i, "ベット処理中にエラーが発生しました。")
		return
	}

	game.Bets = append(game.Bets, Bet{UserID: userID, HorseIndex: horseIndex, Amount: betAmount})

	content := fmt.Sprintf("✅ <@%s> が **%s** に **%d** チップをベットしました。", userID, game.Horses[horseIndex].Name, betAmount)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// --- Race Logic ---

func (c *HorseRaceCommand) startRace(s *discordgo.Session, game *HorseRaceGame) {
	c.mu.Lock()
	if game.State != HRStateBetting {
		c.mu.Unlock()
		return
	}
	game.State = HRStateRacing
	c.mu.Unlock()

	embed := &discordgo.MessageEmbed{
		Title:       "🏇 レース中！",
		Description: "各馬一斉にスタート！",
		Color:       0xf1c40f, // Yellow
	}
	var emptyComponents []discordgo.MessageComponent
	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}, Components: &emptyComponents})
	if err != nil {
		c.Log.Error("Failed to edit message for race start", "error", err)
		return
	}

	time.Sleep(2 * time.Second)

	horsePositions := make([]int, len(game.Horses))
	rand.Seed(time.Now().UnixNano())

	for {
		c.mu.Lock()
		if game.State == HRStateFinished {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		winner := -1
		for i := range horsePositions {
			if rand.Float64() > (game.Horses[i].Odds / 20.0) {
				horsePositions[i] += rand.Intn(3) + 1
			}
			if horsePositions[i] >= RaceTrackLength {
				horsePositions[i] = RaceTrackLength
				winner = i
				break
			}
		}

		trackEmbed := c.buildRaceTrackEmbed(game, horsePositions)
		_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{trackEmbed}})
		if err != nil {
			c.Log.Error("Failed to edit race track embed", "error", err)
			c.finishRace(s, game, -1, horsePositions)
			return
		}

		if winner != -1 {
			c.finishRace(s, game, winner, horsePositions)
			return
		}

		time.Sleep(1500 * time.Millisecond)
	}
}

func (c *HorseRaceCommand) finishRace(s *discordgo.Session, game *HorseRaceGame, winnerIndex int, positions []int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State == HRStateFinished {
		return
	}
	game.State = HRStateFinished

	var resultEmbed *discordgo.MessageEmbed

	if winnerIndex == -1 {
		resultEmbed = &discordgo.MessageEmbed{
			Title:       "レース中止",
			Description: "レース中にエラーが発生したため、中止されました。ベットは返金されます。",
			Color:       0x95a5a6, // Gray
		}
		// Refund all bets
		for _, bet := range game.Bets {
			casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, bet.UserID)
			if err != nil {
				c.Log.Error("Failed to get user data for refund", "error", err, "userID", bet.UserID)
				continue
			}
			casinoData.Chips += bet.Amount
			if err := c.Store.UpdateCasinoData(casinoData); err != nil {
				c.Log.Error("Failed to update user data for refund", "error", err, "userID", bet.UserID)
			}
		}
	} else {
		winnerHorse := game.Horses[winnerIndex]
		winningsMap := make(map[string]int64)
		var winnerMentions []string

		for _, bet := range game.Bets {
			if bet.HorseIndex == winnerIndex {
				casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, bet.UserID)
				if err != nil {
					c.Log.Error("Failed to get winner data for payout", "error", err, "userID", bet.UserID)
					continue
				}
				payout := int64(float64(bet.Amount) * winnerHorse.Odds)
				casinoData.Chips += payout // Add the full payout
				winningsMap[bet.UserID] += payout

				if err := c.Store.UpdateCasinoData(casinoData); err != nil {
					c.Log.Error("Failed to update winner data after race", "error", err, "userID", bet.UserID)
				}
			}
		}

		for userID := range winningsMap {
			winnerMentions = append(winnerMentions, fmt.Sprintf("<@%s>", userID))
		}

		resultEmbed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("🏁 レース終了！ 優勝は %s %s！", winnerHorse.Emoji, winnerHorse.Name),
			Description: c.buildRaceTrack(game, positions),
			Color:       0x2ecc71, // Green
		}

		var resultDescription strings.Builder
		for _, bet := range game.Bets {
			if bet.HorseIndex == winnerIndex {
				payout := winningsMap[bet.UserID]
				profit := payout - bet.Amount
				resultDescription.WriteString(fmt.Sprintf("👑 <@%s> は **%d** チップをベットして **%d** チップの配当を獲得！ (収支: **+%d**)\n", bet.UserID, bet.Amount, payout, profit))
			} else {
				resultDescription.WriteString(fmt.Sprintf("💔 <@%s> は **%d** チップを失いました...\n", bet.UserID, bet.Amount))
			}
		}

		if len(game.Bets) > 0 {
			resultEmbed.Fields = []*discordgo.MessageEmbedField{{
				Name:  "ベット結果",
				Value: resultDescription.String(),
			}}
		} else {
			resultEmbed.Fields = []*discordgo.MessageEmbedField{{
				Name:  "ベット結果",
				Value: "誰もベットしていませんでした。",
			}}
		}
	}

	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{resultEmbed}})
	if err != nil {
		c.Log.Error("Failed to edit final race result", "error", err)
	}

	delete(c.races, game.ChannelID)
}

// --- Helper Functions ---

var horseNames = []string{"スミレバカノフ", "シンボリルドルフ", "ディープインパクト", "オルフェーヴル", "キタサンブラック", "ハルウララ", "サイレンススズカ", "ウオッカ", "ダイワスカーレット", "ゴールドシップ", "メジロマックイーン"}
var horseEmojis = []string{"🏇", "🐎", "🐴", "🦄", "🦓"}

func generateHorses(count int) []Horse {
	rand.Seed(time.Now().UnixNano())
	names := make([]string, len(horseNames))
	copy(names, horseNames)
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })

	horses := make([]Horse, count)
	for i := 0; i < count; i++ {
		horses[i] = Horse{
			Name:  names[i],
			Emoji: horseEmojis[i],
			Odds:  1.5 + rand.Float64()*15,
		}
	}
	return horses
}

func (c *HorseRaceCommand) buildBettingEmbed(game *HorseRaceGame) *discordgo.MessageEmbed {
	fields := make([]*discordgo.MessageEmbedField, len(game.Horses))
	for i, horse := range game.Horses {
		fields[i] = &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%d. %s %s", i+1, horse.Emoji, horse.Name),
			Value:  fmt.Sprintf("単勝: %.2f倍", horse.Odds),
			Inline: true,
		}
	}

	return &discordgo.MessageEmbed{
		Title:       "🏇 競馬 - ベット受付中",
		Description: "ベットしたい馬のボタンを押してください！",
		Color:       0x3498db, // Blue
		Fields:      fields,
	}
}

func (c *HorseRaceCommand) buildBettingComponents(game *HorseRaceGame) []discordgo.MessageComponent {
	var actionRows []discordgo.ActionsRow
	var buttons []discordgo.MessageComponent

	for i, horse := range game.Horses {
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("%d. %s", i+1, horse.Name),
			Style:    discordgo.SecondaryButton,
			CustomID: BetButtonPrefix + strconv.Itoa(i),
			Emoji:    &discordgo.ComponentEmoji{Name: horse.Emoji},
		})
		if (i+1)%5 == 0 || i == len(game.Horses)-1 {
			actionRows = append(actionRows, discordgo.ActionsRow{Components: buttons})
			buttons = []discordgo.MessageComponent{}
		}
	}

	actionRows = append(actionRows, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "レース開始",
				Style:    discordgo.SuccessButton,
				CustomID: StartRaceButtonID,
				Emoji:    &discordgo.ComponentEmoji{Name: "🏁"},
			},
		},
	})

	components := make([]discordgo.MessageComponent, len(actionRows))
	for i, row := range actionRows {
		components[i] = row
	}
	return components
}

func (c *HorseRaceCommand) buildRaceTrackEmbed(game *HorseRaceGame, positions []int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🏇 レース中！",
		Description: c.buildRaceTrack(game, positions),
		Color:       0xf1c40f, // Yellow
	}
}

func (c *HorseRaceCommand) buildRaceTrack(game *HorseRaceGame, positions []int) string {
	var track strings.Builder
	for i, horse := range game.Horses {
		pos := positions[i]
		track.WriteString(fmt.Sprintf("**%d.** ", i+1))
		track.WriteString(strings.Repeat("-", pos))
		track.WriteString(horse.Emoji)
		track.WriteString(strings.Repeat("-", RaceTrackLength-pos))
		track.WriteString(fmt.Sprintf("🏁 %s\n", horse.Name))
	}
	return track.String()
}