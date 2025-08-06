package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// ProfileAnalysisRequest はPythonサーバーに送るプロフィール分析リクエストの構造体
type ProfileAnalysisRequest struct {
	Username       string   `json:"username"`
	Roles          []string `json:"roles"`
	RecentMessages []string `json:"recent_messages"`
}

type ProfileCommand struct {
	Log   interfaces.Logger
	Store interfaces.DataStore
}

func (c *ProfileCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "profile",
		Description: "AIがあなたのプロフィールを分析します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "分析したいユーザー（任意）",
				Required:    false,
			},
		},
	}
}

func (c *ProfileCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var targetUser *discordgo.User
	if len(options) > 0 {
		targetUser = options[0].UserValue(s)
	} else {
		targetUser = i.Member.User
	}

	// 「考え中...」と即時応答
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial profile response", "error", err)
		return
	}

	// 最近のメッセージを取得 (最大100件)
	recentMessages, err := c.Store.GetRecentMessagesByUser(i.GuildID, targetUser.ID, 100)
	if err != nil {
		c.Log.Error("Failed to get recent messages for profile", "error", err)
		// エラーでも続行するが、メッセージは空として扱う
		recentMessages = []string{}
	}

	// Pythonサーバーに送信するデータを作成
	reqData := ProfileAnalysisRequest{
		Username:       targetUser.Username,
		RecentMessages: recentMessages,
	}
	reqJson, _ := json.Marshal(reqData)

	// Pythonサーバーの分析エンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/analyze-profile", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		c.Log.Error("AIサーバーへの接続に失敗", "error", err)
		content := "エラー: AIサーバーへの接続に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var textResp TextResponse
	if err := json.Unmarshal(body, &textResp); err != nil {
		c.Log.Error("Failed to unmarshal AI profile response", "error", err)
		return
	}

	if textResp.Error != "" || resp.StatusCode != http.StatusOK {
		c.Log.Error("AIプロファイル分析に失敗", "error", textResp.Error, "status_code", resp.StatusCode)
		content := fmt.Sprintf("エラー: AIプロファイル分析に失敗しました。\n`%s`", textResp.Error)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🤖 Lunaによる %s のプロフィール", targetUser.Username),
		Description: textResp.Text,
		Color:       0x824ff1, // Gemini Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    targetUser.String(),
			IconURL: targetUser.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Luna AI",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final profile response", "error", err)
	}
}

func (c *ProfileCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ProfileCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ProfileCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *ProfileCommand) GetCategory() string                                                  { return "AI" }
