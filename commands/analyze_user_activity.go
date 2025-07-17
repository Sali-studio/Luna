package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

type AnalyzeUserActivityCommand struct {
	Log interfaces.Logger
}

func (c *AnalyzeUserActivityCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name: "活動傾向を分析",
		Type: discordgo.UserApplicationCommand, // ユーザーコンテキストメニューとして定義
	}
}

// UserActivityRequest はPythonサーバーに送るユーザー活動分析リクエストの構造体
type UserActivityRequest struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	JoinedAt  string `json:"joined_at"`
	Roles     []string `json:"roles"`
	// 他にも、ボットがアクセスできる範囲でメッセージ数やアクティビティ情報などを追加可能
}

func (c *AnalyzeUserActivityCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 対象のユーザー情報を取得
	targetUser := i.ApplicationCommandData().Resolved.Users[i.ApplicationCommandData().TargetID]

	// ユーザーのギルドメンバー情報を取得
	member, err := s.GuildMember(i.GuildID, targetUser.ID)
	if err != nil {
		c.Log.Error("Failed to get guild member info", "error", err)
		content := "エラー: ユーザー情報を取得できませんでした。"
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			c.Log.Error("Failed to send error response", "error", err)
		}
		return
	}

	// ロール名を収集
	var roleNames []string
	for _, roleID := range member.Roles {
		role, err := s.State.Role(i.GuildID, roleID)
		if err == nil {
			roleNames = append(roleNames, role.Name)
		}
	}

	// 「考え中...」と即時応答
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		c.Log.Error("Failed to send initial response", "error", err)
		return
	}

	// Pythonサーバーに送信するデータを作成
	reqData := UserActivityRequest{
		UserID:   targetUser.ID,
		Username: targetUser.Username,
		JoinedAt: member.JoinedAt.Format(time.RFC3339),
		Roles:    roleNames,
	}
	reqJson, _ := json.Marshal(reqData)

	// Pythonサーバーの分析エンドポイントにリクエストを送信
	resp, err := http.Post("http://localhost:5001/analyze-user-activity", "application/json", bytes.NewBuffer(reqJson))

	// エラーハンドリング
	if err != nil {
		c.Log.Error("AIサーバーへの接続に失敗", "error", err)
		content := "エラー: AIサーバーへの接続に失敗しました。"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			c.Log.Error("Failed to edit error response", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	// レスポンスを読み取りJSONをパース
	body, _ := io.ReadAll(resp.Body)
	var textResp TextResponse // common.goで定義されたTextResponseを使用
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
		Title:       fmt.Sprintf("📊 %s さんの活動傾向", targetUser.Username),
		Description: textResp.Text,
		Color:       0x824ff1, // Gemini Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    targetUser.String(),
			IconURL: targetUser.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by Gemini",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		c.Log.Error("Failed to edit final response", "error", err)
	}
}

func (c *AnalyzeUserActivityCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *AnalyzeUserActivityCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *AnalyzeUserActivityCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *AnalyzeUserActivityCommand) GetCategory() string                                                  { return "Utility" }