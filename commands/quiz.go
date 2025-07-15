package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"

	"github.com/bwmarrin/discordgo" // <-- インポートを追加
)

// --- Structs for new /generate-quiz endpoint ---

// QuizRequest is the request sent to the Python server.
type QuizRequest struct {
	Topic   string   `json:"topic"`
	History []string `json:"history"`
}

// QuizResponse is the JSON response we expect from the Python server.
type QuizResponse struct {
	Question           string   `json:"question"`
	Options            []string `json:"options"`
	CorrectAnswerIndex int      `json:"correct_answer_index"`
	Explanation        string   `json:"explanation"`
	Error              string   `json:"error"`
}

// --- Original structs (now defined in models.go) ---

// TextRequest is the original request struct.
// type TextRequest struct {
// 	Prompt string `json:"prompt"`
// }

// TextResponse is the original response struct.
// type TextResponse struct {
// 	Text  string `json:"text"`
// 	Error string `json:"error"`
// }

// --- Command Definition ---

type QuizCommand struct {
	Log   interfaces.Logger
	Store interfaces.DataStore // Add DataStore to the command struct
}

func (c *QuizCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ai-game",
		Description: "Luna Assistantを使ったクイズや豆知識を出題します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "quiz",
				Description: "指定されたトピックに関するクイズを生成します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "topic",
						Description: "クイズのトピック (例: 歴史, 宇宙, 動物)",
						Required:    false,
					},
				},
			},
			{
				Name:        "trivia",
				Description: "指定されたトピックに関する豆知識を生成します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "topic",
						Description: "豆知識のトピック (例: 科学, 映画, スポーツ)",
						Required:    false,
					},
				},
			},
		},
	}
}

func (c *QuizCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subcommand := i.ApplicationCommandData().Options[0]
	switch subcommand.Name {
	case "quiz":
		c.handleQuiz(s, i)
	case "trivia":
		c.handleTrivia(s, i) // Trivia remains unchanged for now
	}
}

func (c *QuizCommand) handleQuiz(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var topic string
	if len(i.ApplicationCommandData().Options[0].Options) > 0 {
		topic = i.ApplicationCommandData().Options[0].Options[0].StringValue()
	} else {
		topic = "ランダムなトピック"
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	// 1. Get recent questions from DB
	history, err := c.Store.GetRecentQuizQuestions(i.GuildID, topic, 20)
	if err != nil {
		c.Log.Warn("Failed to get quiz history from DB", "error", err)
		// Ensure history is not nil, even if there's an error
		history = []string{}
	}

	// 2. Send request to the new /generate-quiz endpoint
	reqData := QuizRequest{Topic: topic, History: history}
	reqJson, _ := json.Marshal(reqData)

	resp, err := http.Post("http://localhost:5001/generate-quiz", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("AIサーバーへの接続に失敗", "error", err)
		content := "エラー: AIサーバーへの接続に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// 3. Parse the new JSON response
	body, _ := io.ReadAll(resp.Body)
	var quizResp QuizResponse
	if err := json.Unmarshal(body, &quizResp); err != nil {
		c.Log.Error("Failed to unmarshal AI JSON response", "error", err, "response_body", string(body))
		content := "エラー: AIからの応答を解析できませんでした。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	if quizResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("AIからの応答取得に失敗", "error", quizResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: AIからの応答取得に失敗しました。\n`%s`", quizResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	// 4. Save the new question to the DB
	if err := c.Store.SaveQuizQuestion(i.GuildID, topic, quizResp.Question); err != nil {
		c.Log.Warn("Failed to save new quiz question to DB", "error", err)
	}

	// 5. Format the JSON into a user-friendly embed
	var formattedOptions string
	for i, opt := range quizResp.Options {
		formattedOptions += fmt.Sprintf("**%d.** %s\n", i+1, opt)
	}

	answerText := fmt.Sprintf("正解は **%d. %s** です。\n\n**解説:**\n%s", quizResp.CorrectAnswerIndex+1, quizResp.Options[quizResp.CorrectAnswerIndex], quizResp.Explanation)

	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: "Luna Assistant", IconURL: s.State.User.AvatarURL("")},
		Title:       fmt.Sprintf("【%s】に関するクイズ！", topic),
		Description: fmt.Sprintf("**Q.** %s\n\n%s\n答えと解説は下のネタバレをクリック！\n||%s||", quizResp.Question, formattedOptions, answerText),
		Color:       0x4a8cf7,
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}
}

// handleTrivia remains unchanged, using the old prompt-based generation
func (c *QuizCommand) handleTrivia(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var topic string
	if len(i.ApplicationCommandData().Options[0].Options) > 0 {
		topic = i.ApplicationCommandData().Options[0].Options[0].StringValue()
	} else {
		topic = "何か"
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	persona := "あなたは知識の泉です。ユーザーが「へぇ！」と驚くような面白い豆知識を教えてあげてください。"
	prompt := fmt.Sprintf("システムインストラクション（あなたの役割）: %s\n\n[ユーザーからのリクエスト]\n「%s」に関する面白い豆知識を一つ、簡潔に教えてください。", persona, topic)

	// Using the old TextRequest/TextResponse structs for this endpoint
	reqData := TextRequest{Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	resp, err := http.Post("http://localhost:5001/generate-text", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("Luna Assistantサーバーへの接続に失敗", "error", err)
		content := "エラー: Luna Assistantサーバーへの接続に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var textResp TextResponse
	if err := json.Unmarshal(body, &textResp); err != nil {
		c.Log.Error("Failed to unmarshal AI response", "error", err)
		return
	}

	if textResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("Luna Assistantからの応答取得に失敗", "error", textResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: Luna Assistantからの応答取得に失敗しました。\n`%s`", textResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: "Luna Assistant", IconURL: s.State.User.AvatarURL("")},
		Description: textResp.Text,
		Color:       0x4a8cf7,
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}
}

func (c *QuizCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *QuizCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *QuizCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *QuizCommand) GetCategory() string                                                  { return "AI" }