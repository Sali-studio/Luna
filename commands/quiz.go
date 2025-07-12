package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"luna/interfaces"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type QuizCommand struct{
	Log interfaces.Logger
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
		c.handleTrivia(s, i)
	}
}

func (c *QuizCommand) handleQuiz(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var topic string
	if len(i.ApplicationCommandData().Options[0].Options) > 0 {
		topic = i.ApplicationCommandData().Options[0].Options[0].StringValue()
	} else {
		topic = "ランダムなトピック"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	persona := "あなたはクイズマスターです。ユーザーに楽しくてためになるクイズを出題してください。"
	prompt := fmt.Sprintf("システムインストラクション（あなたの役割）: %s\n\n[ユーザーからのリクエスト]\n「%s」に関する面白い4択クイズを1問出題してください。答えと簡単な解説はDiscordのネタバレ機能(`||hoge||`)を使って隠してください。", persona, topic)

	c.generateAndSend(s, i, prompt)
}

func (c *QuizCommand) handleTrivia(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var topic string
	if len(i.ApplicationCommandData().Options[0].Options) > 0 {
		topic = i.ApplicationCommandData().Options[0].Options[0].StringValue()
	} else {
		topic = "何か"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	persona := "あなたは知識の泉です。ユザーが「へぇ！」と驚くような面白い豆知識を教えてあげてください。"
	prompt := fmt.Sprintf("システムインストラクション（あなたの役割）: %s\n\n[ユーザーからのリクエスト]\n「%s」に関する面白い豆知識を一つ、簡潔に教えてください。", persona, topic)

	c.generateAndSend(s, i, prompt)
}

func (c *QuizCommand) generateAndSend(s *discordgo.Session, i *discordgo.InteractionCreate, prompt string) {
	// 既存のTextRequest構造体を再利用
	reqData := TextRequest{Prompt: prompt}
	reqJson, _ := json.Marshal(reqData)

	resp, err := http.Post("http://localhost:5001/generate-text", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("Luna Assistantサーバーへの接続に失敗", "error", err)
		content := "エラー: Luna Assistantサーバーへの接続に失敗しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var textResp TextResponse
	json.Unmarshal(body, &textResp)

	if textResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("Luna Assistantからの応答取得に失敗", "error", textResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: Luna Assistantからの応答取得に失敗しました。\n`%s`", textResp.Error)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		return
	}

	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: "Luna Assistant", IconURL: s.State.User.AvatarURL("")},
		Description: textResp.Text,
		Color:       0x4a8cf7,
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func (c *QuizCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *QuizCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *QuizCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *QuizCommand) GetCategory() string                                                  { return "AI" }