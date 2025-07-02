package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

type PingCommand struct {
	StartTime time.Time
	Store     *storage.DBStore
}

func (c *PingCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "ボットのパフォーマンスと稼働時間を測定します",
	}
}

func (c *PingCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. API応答時間を測定するため、最初のメッセージを送信
	apiStart := time.Now()
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "測定中...",
		},
	})
	apiLatency := time.Since(apiStart)
	if err != nil {
		logger.Error("pingコマンドの初期応答に失敗", "error", err)
		return
	}

	// 2. データベースの応答時間を測定
	dbStart := time.Now()
	err = c.Store.PingDB()
	dbLatency := time.Since(dbStart)
	dbStatus := "✅ 正常"
	if err != nil {
		dbStatus = "❌ 異常"
		dbLatency = 0 // エラー時は0
	}

	// 3. ゲートウェイの応答時間を取得
	gatewayLatency := s.HeartbeatLatency()

	// 4. 稼働時間を計算
	uptime := time.Since(c.StartTime)
	uptimeStr := formatUptime(uptime)

	// 5. 結果をEmbedにまとめて表示
	latencyColor := 0x43b581 // Green
	if gatewayLatency.Milliseconds() > 150 || apiLatency.Milliseconds() > 300 {
		latencyColor = 0xfaa61a // Yellow
	}
	if gatewayLatency.Milliseconds() > 400 || apiLatency.Milliseconds() > 600 {
		latencyColor = 0xf04747 // Red
	}
	if dbStatus == "❌ 異常" {
		latencyColor = 0xf04747 // Red
	}

	embed := &discordgo.MessageEmbed{
		Title: "🏓 Pong! - ヘルスチェック",
		Color: latencyColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ゲートウェイ", Value: fmt.Sprintf("```%s```", gatewayLatency.String()), Inline: true},
			{Name: "API応答", Value: fmt.Sprintf("```%s```", apiLatency.String()), Inline: true},
			{Name: "データベース", Value: fmt.Sprintf("```%s (%s)```", dbStatus, dbLatency.String()), Inline: true},
			{Name: "稼働時間", Value: fmt.Sprintf("```%s```", uptimeStr), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 最初に送信した「測定中...」メッセージを編集
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &[]string{""}[0], // メッセージ内容は空にする
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
}

// 稼働時間を「X日 Y時間 Z分」のような分かりやすい形式に変換するヘルパー関数
func formatUptime(d time.Duration) string {
	d = d.Round(time.Minute)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%d日 %d時間 %d分", days, h, m)
}

func (c *PingCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PingCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *PingCommand) GetComponentIDs() []string                                            { return []string{} }
