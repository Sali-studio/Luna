package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	BetButtonPrefix      = "hr_bet_"
	StartRaceButtonID    = "hr_start_race"
	BetModalCustomID     = "hr_bet_modal"
	RaceTrackLength      = 20
)

// レースの状態
type RaceState int

const (
	StateBetting RaceState = iota
	StateRacing
	StateFinished
)

// 馬の情報
type Horse struct {
	Name  string
	Odds  float64
	Emoji string
}

// ベット情報
type Bet struct {
	UserID     string
	HorseIndex int
	Amount     int64
}

// レースゲーム全体の管理	ype HorseRaceGame struct {
	State         RaceState
	Horses        []Horse
	Bets          []Bet
	MessageID     string
	ChannelID     string
	Interaction   *discordgo.Interaction
	CreatorID     string // レースを開始した人のID
}

// HorseRaceCommand は /horserace コマンドを処理します。
type HorseRaceCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
	races map[string]*HorseRaceGame // channelID -> game
	mu    sync.Mutex
}

// NewHorseRaceCommand は新しいHorseRaceCommandを返します。
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

// ここにHandle, HandleComponent, HandleModalが続く

// レース情報を保存
	c.traces[i.ChannelID] = game
}

func (c *HorseRaceCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	game, exists := c.traces[i.ChannelID]
	if !exists {
		return // No active race in this channel
	}

	customID := i.MessageComponentData().CustomID

	if strings.HasPrefix(customID, BetButtonPrefix) {
		if game.State != StateBetting {
			sendErrorResponse(s, i, "ベット受付は終了しました。")
			return
		}

		horseIndexStr := strings.TrimPrefix(customID, BetButtonPrefix)
		horseIndex, _ := strconv.Atoi(horseIndexStr)

		modal := discordgo.InteractionResponseModal{
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
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{CustomID: modal.CustomID, Title: modal.Title, Components: modal.Components}})

	} else if customID == StartRaceButtonID {
		if i.Member.User.ID != game.CreatorID {
			sendErrorResponse(s, i, "レースを開始できるのは、レースを開始した本人だけです。")
			return
		}
		go c.startRace(s, game)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	}
}

func (c *HorseRaceCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	game, exists := c.traces[i.ChannelID]
	if !exists {
		return
	}

	customID := i.ModalSubmitData().CustomID
	if strings.HasPrefix(customID, BetModalCustomID) {
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
			c.Log.Error("Failed to get casino data for horse race bet", "error", err)
			sendErrorResponse(s, i, "エラーが発生しました。")
			return
		}

		if casinoData.Chips < betAmount {
			sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", casinoData.Chips))
			return
		}

		// Add the bet
		game.Bets = append(game.Bets, Bet{UserID: userID, HorseIndex: horseIndex, Amount: betAmount})

		// Confirm the bet
		content := fmt.Sprintf("✅ <@%s> が **%s** に **%d** チップをベットしました。", userID, game.Horses[horseIndex].Name, betAmount)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func (c *HorseRaceCommand) startRace(s *discordgo.Session, game *HorseRaceGame) {
	c.mu.Lock()
	// ベット受付中の場合のみレースを開始
	if game.State != StateBetting {
		c.mu.Unlock()
		return
	}
	game.State = StateRacing
	c.mu.Unlock()

	// Update message to "Racing"
	embed := &discordgo.MessageEmbed{
		Title:       "🏇 レース中！",
		Description: "各馬一斉にスタート！",
		Color:       0xf1c40f, // Yellow
	}
	// Remove buttons during the race
	// An empty slice of components will remove all of them.
	var emptyComponents []discordgo.MessageComponent
	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}, Components: &emptyComponents})
	if err != nil {
		c.Log.Error("Failed to edit message for race start", "error", err)
		return
	}

	time.Sleep(2 * time.Second)

	horsePositions := make([]int, len(game.Horses))
	rand.Seed(time.Now().UnixNano())

	// Animation loop
	for {
		// Check if the game has been forcefully finished
		c.mu.Lock()
		if game.State == StateFinished {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		// Update horse positions
		winner := -1
		for i := range horsePositions {
			// Lower odds = higher chance to move
			if rand.Float64() > (game.Horses[i].Odds/20.0) { // Adjust this logic for race balance
				horsePositions[i] += rand.Intn(3) + 1 // Move 1 to 3 steps
			}
			if horsePositions[i] >= RaceTrackLength {
				horsePositions[i] = RaceTrackLength
				winner = i
				break
			}
		}

		// Build and update the race track embed
		trackEmbed := c.buildRaceTrackEmbed(game, horsePositions)
		_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{trackEmbed}})
		if err != nil {
			c.Log.Error("Failed to edit race track embed", "error", err)
			// If editing fails, we should probably stop the race
			c.finishRace(s, game, -1, horsePositions) // -1 indicates no winner due to error
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

	// Ensure this runs only once
	if game.State == StateFinished {
		return
	}
	game.State = StateFinished

	var resultEmbed *discordgo.MessageEmbed

	if winnerIndex == -1 {
		resultEmbed = &discordgo.MessageEmbed{
			Title:       "レース中止",
			Description: "レース中にエラーが発生したため、中止されました。ベットは返金されます。",
			Color:       0x95a5a6, // Gray
		}
	} else {
		winnerHorse := game.Horses[winnerIndex]
		winningsMap := make(map[string]int64)
		var winnerMentions []string

		// Calculate winnings and update database
		for _, bet := range game.Bets {
			casinoData, err := c.Store.GetCasinoData(game.Interaction.GuildID, bet.UserID)
			if err != nil {
				c.Log.Error("Failed to get user data for payout", "error", err, "userID", bet.UserID)
				continue
			}

			if bet.HorseIndex == winnerIndex {
				payout := int64(float64(bet.Amount) * winnerHorse.Odds)
				casinoData.Chips += payout // Add the full payout
				winningsMap[bet.UserID] += payout
			} else {
				casinoData.Chips -= bet.Amount // Subtract the bet amount for losers
			}

			if err := c.Store.UpdateCasinoData(casinoData); err != nil {
				c.Log.Error("Failed to update user data after race", "error", err, "userID", bet.UserID)
			}
		}

		for userID := range winningsMap {
			winnerMentions = append(winnerMentions, fmt.Sprintf("<@%s>", userID))
		}

		// Build final result embed
		resultEmbed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("🏁 レース終了！ 優勝は %s %s！", winnerHorse.Emoji, winnerHorse.Name),
			Description: c.buildRaceTrack(game, positions),
			Color:       0x2ecc71, // Green
		}

		if len(winnerMentions) > 0 {
			resultEmbed.Fields = []*discordgo.MessageEmbedField{
				{
					Name:  "🎉 勝者",
					Value: strings.Join(winnerMentions, " "),
				},
			}
		} else {
			resultEmbed.Fields = []*discordgo.MessageEmbedField{
				{
					Name:  "🎉 勝者",
					Value: "なし",
				},
			}
		}
	}

	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{resultEmbed}})
	if err != nil {
		c.Log.Error("Failed to edit final race result", "error", err)
	}

	// Clean up the race from the map
	delete(c.traces, game.ChannelID)
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
		track.WriteString("🏁\n")
	}
	return track.String()
}

// --- Helper functions for Handle ---

var horseNames = []string{"シンボリルドルフ", "ディープインパクト", "オルフェーヴル", "キタサンブラック", "ハルウララ", "サイレンススズカ", "ウオッカ", "ダイワスカーレット", "ゴールドシップ", "メジロマックイーン"}
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
			Odds:  1.5 + rand.Float64()*15, // 1.5倍から16.5倍のランダムなオッズ
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
	var rows []discordgo.ActionsRow
	var buttons []discordgo.MessageComponent

	for i, horse := range game.Horses {
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("%d. %s", i+1, horse.Name),
			Style:    discordgo.SecondaryButton,
			CustomID: BetButtonPrefix + strconv.Itoa(i),
			Emoji:    &discordgo.ComponentEmoji{Name: horse.Emoji},
		})
		if (i+1)%5 == 0 || i == len(game.Horses)-1 {
			rows = append(rows, discordgo.ActionsRow{Components: buttons})
			buttons = []discordgo.MessageComponent{}
		}
	}

	// レース開始ボタンを追加
	rows = append(rows, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "レース開始",
				Style:    discordgo.SuccessButton,
				CustomID: StartRaceButtonID,
				Emoji:    &discordgo.ComponentEmoji{Name: "🏁"},
			},
		},
	})

	return rows
}

