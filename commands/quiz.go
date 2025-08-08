package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	QuizButtonPrefix = "q_bet_"
	QuizModalID      = "q_modal"
)

// QuizState はクイズゲームの状態を表します。
type QuizState int

const (
	QStateBetting QuizState = iota
	QStateFinished
)

// Quiz は個々のベット情報を表します。
type Quiz struct {
	UserID      string
	ChoiceIndex int
	Amount      int64
}

// QuizGame はクイズゲーム全体の管理を行います。
type QuizGame struct {
	State              QuizState
	Question           string
	Options            []string
	CorrectAnswerIndex int
	Explanation        string
	Bets               []Quiz
	MessageID          string
	ChannelID          string
	Interaction        *discordgo.Interaction
	EndTime            time.Time
}

// QuizCommand は /quiz コマンドを処理します。
type QuizCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
	games map[string]*QuizGame // channelID -> game
	mu    sync.Mutex
}

// --- Command/Component/Modal Handlers ---

func NewQuizCommand(store interfaces.DataStore, log interfaces.Logger) *QuizCommand {
	return &QuizCommand{
		Store: store,
		Log:   log,
		games: make(map[string]*QuizGame),
	}
}

func (c *QuizCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "quiz",
		Description: "AIクイズにチップを賭けて挑戦！",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "topic",
				Description: "クイズのトピック (例: 歴史, 宇宙, 動物)",
				Required:    false,
			},
		},
	}
}

func (c *QuizCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	if _, exists := c.games[i.ChannelID]; exists {
		c.mu.Unlock()
		sendErrorResponse(s, i, "このチャンネルでは既にクイズベットが進行中です。")
		return
	}
	c.mu.Unlock()

	// Immediately defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		c.Log.Error("Failed to defer quiz response", "error", err)
		return
	}

	go func() {
		var topic string
		if len(i.ApplicationCommandData().Options) > 0 {
			topic = i.ApplicationCommandData().Options[0].StringValue()
		} else {
			topic = "ランダムなトピック"
		}

		quiz, err := c.getQuizFromAI(topic)
		if err != nil {
			c.Log.Error("Failed to get quiz from AI", "error", err)
			errorContent := "クイズの取得に失敗しました。AIサーバーが起動しているか確認してください。"
			_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorContent,
			})
			if err != nil {
				c.Log.Error("Failed to send error response for quiz", "error", err)
			}
			return
		}

		c.mu.Lock()
		defer c.mu.Unlock()

		game := &QuizGame{
			State:              QBStateQuizting,
			ChannelID:          i.ChannelID,
			Interaction:        i.Interaction,
			Question:           quiz.Question,
			Options:            quiz.Options,
			CorrectAnswerIndex: quiz.CorrectAnswerIndex,
			Explanation:        quiz.Explanation,
			EndTime:            time.Now().Add(30 * time.Second),
		}

		embed := c.buildBettingEmbed(game)
		components := c.buildBettingComponents(game, false)

		msg, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		if err != nil {
			c.Log.Error("Failed to send initial quiz message", "error", err)
			return
		}

		game.MessageID = msg.ID
		c.games[i.ChannelID] = game

		go c.scheduleEndBetting(s, game)
	}()
}

func (c *QuizCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	game, exists := c.games[i.ChannelID]
	c.mu.Unlock()

	if !exists {
		return
	}

	customID := i.MessageComponentData().CustomID
	if strings.HasPrefix(customID, QuizButtonPrefix) {
		c.handleBetButton(s, i, game)
	}
}

func (c *QuizCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.mu.Lock()
	game, exists := c.games[i.ChannelID]
	c.mu.Unlock()

	if !exists {
		return
	}

	customID := i.ModalSubmitData().CustomID
	if strings.HasPrefix(customID, QuizModalID) {
		c.handleBetModalSubmit(s, i, game)
	}
}

func (c *QuizCommand) GetComponentIDs() []string {
	return []string{QuizButtonPrefix, QuizModalID}
}

func (c *QuizCommand) GetCategory() string {
	return "カジノ"
}

// --- Handler Logic ---

func (c *QuizCommand) handleBetButton(s *discordgo.Session, i *discordgo.InteractionCreate, game *QuizGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if game.State != QBStateQuizting {
		sendErrorResponse(s, i, "ベット受付は終了しました。")
		return
	}

	choiceIndexStr := strings.TrimPrefix(i.MessageComponentData().CustomID, QuizButtonPrefix)

	modal := discordgo.InteractionResponseData{
		CustomID: QuizModalID + "_" + choiceIndexStr,
		Title:    "ベット額の入力",
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

func (c *QuizCommand) handleBetModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, game *QuizGame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	customID := i.ModalSubmitData().CustomID
	choiceIndexStr := strings.TrimPrefix(customID, QuizModalID+"_")
	choiceIndex, _ := strconv.Atoi(choiceIndexStr)
	betAmountStr := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	betAmount, err := strconv.ParseInt(betAmountStr, 10, 64)

	if err != nil || betAmount <= 0 {
		sendErrorResponse(s, i, "有効なベット額を入力してください。")
		return
	}

	userID := i.Member.User.ID
	casinoData, err := c.Store.GetCasinoData(i.GuildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for quiz", "error", err)
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

	game.Bets = append(game.Bets, Quiz{UserID: userID, ChoiceIndex: choiceIndex, Amount: betAmount})

	content := fmt.Sprintf("✅ <@%s> が **%d番** に **%d** チップをベットしました。", userID, choiceIndex+1, betAmount)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// --- Helper functions and game logic ---

func (c *QuizCommand) getQuizFromAI(topic string) (*QuizResponse, error) {
	history, err := c.Store.GetRecentQuizQuestions("global", topic, 20)
	if err != nil {
		c.Log.Warn("Failed to get quiz history", "error", err)
		history = []string{}
	}

	reqData := QuizRequest{Topic: topic, History: history}
	reqJson, _ := json.Marshal(reqData)

	resp, err := http.Post("http://localhost:5001/generate-quiz", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var quizResp QuizResponse
	if err := json.Unmarshal(body, &quizResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quiz response: %w, body: %s", err, string(body))
	}

	if quizResp.Error != "" {
		return nil, fmt.Errorf("AI returned an error: %s", quizResp.Error)
	}

	c.Store.SaveQuizQuestion("global", topic, quizResp.Question)

	return &quizResp, nil
}

func (c *QuizCommand) buildBettingEmbed(game *QuizGame) *discordgo.MessageEmbed {
	optionsStr := ""
	for i, opt := range game.Options {
		optionsStr += fmt.Sprintf("**%d.** %s\n", i+1, opt)
	}

	footer := fmt.Sprintf("ベット受付終了まで: %d秒", int(time.Until(game.EndTime).Seconds()))
	if game.State == QBStateQuizFinished {
		footer = "ベット受付は終了しました。"
	}

	return &discordgo.MessageEmbed{
		Title:       "🧠 クイズ＆ベット！",
		Description: fmt.Sprintf("**Q.** %s\n\n%s", game.Question, optionsStr),
		Color:       0x3498db, // Blue
		Footer:      &discordgo.MessageEmbedFooter{Text: footer},
	}
}

func (c *QuizCommand) buildBettingComponents(game *QuizGame, disabled bool) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	var buttons []discordgo.MessageComponent
	for i := range game.Options {
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("%d番にベット", i+1),
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("%s%d", QuizButtonPrefix, i),
			Disabled: disabled,
		})
	}
	components = append(components, discordgo.ActionsRow{Components: buttons})
	return components
}

func (c *QuizCommand) scheduleEndBetting(s *discordgo.Session, game *QuizGame) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(time.Until(game.EndTime))
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			// ゲームがまだアクティブか確認
			if g, exists := c.games[game.ChannelID]; !exists || g.State != QBStateQuizting {
				c.mu.Unlock()
				return
			}

			// 新しい時間でEmbedを更新
			embed := c.buildBettingEmbed(game)
			// コンポーネントは変更しないようにnilのままにする
			_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				c.Log.Warn("Failed to update quiz timer", "error", err)
			}
			c.mu.Unlock()

		case <-timer.C:
			c.mu.Lock()
			defer c.mu.Unlock()
			if g, exists := c.games[game.ChannelID]; !exists || g.State != QBStateQuizting {
				return
			}
			c.endBetting(s, game)
			return
		}
	}
}

func (c *QuizCommand) endBetting(s *discordgo.Session, game *QuizGame) {
	game.State = QBStateQuizFinished

	correctOption := game.Options[game.CorrectAnswerIndex]
	resultEmbed := &discordgo.MessageEmbed{
		Title:       "ベット終了！結果発表！",
		Description: fmt.Sprintf("**Q.** %s\n\n正解は **%d. %s** でした！\n%s", game.Question, game.CorrectAnswerIndex+1, correctOption, game.Explanation),
		Color:       0x2ecc71, // Green
	}

	var winners []Quiz
	var losers []Quiz

	for _, bet := range game.Bets {
		if bet.ChoiceIndex == game.CorrectAnswerIndex {
			winner := bet
			winners = append(winners, winner)
		} else {
			losers = append(losers, bet)
		}
	}

	var resultDescription strings.Builder
	if len(winners) > 0 {
		resultDescription.WriteString("**🎉 勝者**\n")
		for _, winner := range winners {
			// Payout is 1.2x the bet amount
			payout := int64(float64(winner.Amount) * 1.2)
			casinoData, _ := c.Store.GetCasinoData(game.Interaction.GuildID, winner.UserID)
			casinoData.Chips += payout
			c.Store.UpdateCasinoData(casinoData)
			profit := payout - winner.Amount
			resultDescription.WriteString(fmt.Sprintf("<@%s> が **%d** チップをベットして **%d** チップを獲得！ (収支: **+%d**)\n", winner.UserID, winner.Amount, payout, profit))
		}
	} else {
		resultDescription.WriteString("**😥 勝者なし**\n誰も正解できなかったため、ベットしたチップは返金されます。\n")
		// Refund all bets
		for _, bet := range game.Bets {
			casinoData, _ := c.Store.GetCasinoData(game.Interaction.GuildID, bet.UserID)
			casinoData.Chips += bet.Amount
			c.Store.UpdateCasinoData(casinoData)
		}
	}

	if len(losers) > 0 {
		resultDescription.WriteString("\n**💔 敗者**\n")
		for _, loser := range losers {
			resultDescription.WriteString(fmt.Sprintf("<@%s>\n", loser.UserID))
		}
	}

	resultEmbed.Fields = []*discordgo.MessageEmbedField{{
		Name:  "ベット結果",
		Value: resultDescription.String(),
	}}

	disabledComponents := c.buildBettingComponents(game, true)

	_, err := s.InteractionResponseEdit(game.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{c.buildBettingEmbed(game)},
		Components: &disabledComponents,
	})
	if err != nil {
		c.Log.Warn("Failed to edit original quiz message", "error", err)
	}

	s.ChannelMessageSendEmbed(game.ChannelID, resultEmbed)

	delete(c.games, game.ChannelID)
}
