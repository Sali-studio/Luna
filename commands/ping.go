package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

var StartTime time.Time

type PingCommand struct {
	// main.goから起動時刻を受け取る
	StartTime time.Time
}

func (c *PingCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "ボットのレイテンシと稼働時間を測定します",
	}
}

func (c *PingCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. API応答時間を測定開始
	beforeAPICall := time.Now()

	// 2. 「測定中...」という一時的なメッセージを送信
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "🏓 Pong!",
					Color: 0x7289da, // Discord Blue
					Fields: []*discordgo.MessageEmbedField{
						{Name: "ゲートウェイ", Value: "測定中...", Inline: true},
						{Name: "API応答", Value: "測定中...", Inline: true},
						{Name: "稼働時間", Value: "測定中...", Inline: true},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error("pingコマンドの初期応答に失敗", "error", err)
		return
	}

	// 3. API応答時間を計算
	apiLatency := time.Since(beforeAPICall)

	// 4. ゲートウェイの応答時間を取得
	gatewayLatency := s.HeartbeatLatency()

	// 5. 稼働時間を計算
	uptime := time.Since(c.StartTime)
	uptimeStr := formatUptime(uptime)

	// 6. 結果を埋め込みメッセージで編集して表示
	latencyColor := 0x43b581 // Green
	if gatewayLatency.Milliseconds() > 200 {
		latencyColor = 0xfaa61a // Yellow
	}
	if gatewayLatency.Milliseconds() > 500 {
		latencyColor = 0xf04747 // Red
	}

	embed := &discordgo.MessageEmbed{
		Title: "🏓 Pong!",
		Color: latencyColor, // レイテンシに応じて色が変わる
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ゲートウェイ", Value: fmt.Sprintf("```%s```", gatewayLatency.String()), Inline: true},
			{Name: "API応答", Value: fmt.Sprintf("```%s```", apiLatency.String()), Inline: true},
			{Name: "稼働時間", Value: fmt.Sprintf("```%s```", uptimeStr), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
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
