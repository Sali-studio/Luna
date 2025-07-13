package commands

import (
	"fmt"
	"luna/interfaces"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type PingCommand struct {
	StartTime time.Time
	Store     interfaces.DataStore
}

func (c *PingCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "BOTの応答速度や状態を確認します",
	}
}

func (c *PingCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. まずは即時応答し、APIレイテンシを測定する基準点を作る
	start := time.Now()
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🏓 Pinging...",
		},
	})
	if err != nil {
		return
	}
	apiLatency := time.Since(start)

	// 2. WebSocketのレイテンシを取得
	wsLatency := s.HeartbeatLatency()

	// 3. データベースの応答を確認
	dbStart := time.Now()
	dbErr := c.Store.PingDB()
	dbLatency := time.Since(dbStart)
	dbStatus := "✅ Online"
	if dbErr != nil {
		dbStatus = "❌ Offline"
	}

	// 4. アップタイムを計算
	uptime := time.Since(c.StartTime)

	// 5. 結果をEmbedにまとめて表示
	embed := &discordgo.MessageEmbed{
		Title: "Pong! - BOTステータス",
		Color: 0x7289da, // Discord Blue
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "APIレイテンシ",
				Value:  fmt.Sprintf("```%s```", apiLatency.Round(time.Millisecond).String()),
				Inline: true,
			},
			{
				Name:   "WebSocketレイテンシ",
				Value:  fmt.Sprintf("```%s```", wsLatency.Round(time.Millisecond).String()),
				Inline: true,
			},
			{
				Name:   "データベース",
				Value:  fmt.Sprintf("```%s (%s)```", dbStatus, dbLatency.Round(time.Millisecond).String()),
				Inline: true,
			},
			{
				Name:  "アップタイム",
				Value: fmt.Sprintf("```%s```", formatUptime(uptime)),
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 最初に送った "Pinging..." というメッセージを、完成したEmbedに編集する
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new(string), // テキストを空にする
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		// c.Log is not available in PingCommand.
	}
}

// アップタイムを見やすい形式にフォーマットするヘルパー関数
func formatUptime(d time.Duration) string {
	d = d.Round(time.Second)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d日", days))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%d時間", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%d分", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d秒", s))
	}

	return strings.Join(parts, " ")
}

func (c *PingCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PingCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *PingCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *PingCommand) GetCategory() string                                                  { return "ユーティリティ" }
