package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

// HandleGuildBanAdd はユーザーがBANされたときのイベントを処理します
func HandleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	// ログチャンネルが設定されていなければ何もしない
	logChannel, ok := logChannelID[e.GuildID]
	if !ok {
		return
	}

	logger.Info.Printf("User %s was banned from guild %s", e.User.Username, e.GuildID)

	// --- ここからがUI/UXに凝ったEmbedメッセージの作成 ---

	// BANを実行した犯人を探すため、監査ログを取得
	// この処理には「監査ログの表示」権限がボットに必要
	auditLog, err := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberBanAdd), 1)
	if err != nil {
		logger.Error.Printf("Could not fetch audit log: %v", err)
		return
	}

	executor := "不明"
	reason := "理由の記載なし"
	// 監査ログのエントリから、実行者と理由を探す
	if len(auditLog.AuditLogEntries) > 0 {
		entry := auditLog.AuditLogEntries[0]
		if entry.TargetID == e.User.ID { // BANされたユーザーが監査ログの対象と一致するか確認
			executor = entry.UserID
			if entry.Reason != "" {
				reason = entry.Reason
			}
		}
	}

	// BANログ用のEmbedを作成
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "メンバーがBANされました",
			IconURL: e.User.AvatarURL(""),
		},
		Color: 0xED4245, // 赤色
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "対象ユーザー",
				Value:  fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID),
				Inline: false,
			},
			{
				Name:   "実行者",
				Value:  fmt.Sprintf("<@%s>", executor),
				Inline: true,
			},
			{
				Name:   "理由",
				Value:  reason,
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Luna Logging System",
		},
	}

	s.ChannelMessageSendEmbed(logChannel, embed)
}
