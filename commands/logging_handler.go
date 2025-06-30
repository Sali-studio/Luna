package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

// HandleGuildBanAdd はユーザーがBANされたときのイベントを処理します
func HandleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	logChannel, ok := logChannelID[e.GuildID]
	if !ok {
		return
	}

	logger.Info.Printf("User %s was banned from guild %s", e.User.Username, e.GuildID)

	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberBanAdd), 1)
	executor := "不明"
	reason := "理由の記載なし"
	if len(auditLog.AuditLogEntries) > 0 {
		entry := auditLog.AuditLogEntries[0]
		if entry.TargetID == e.User.ID {
			executor = entry.UserID
			if entry.Reason != "" {
				reason = entry.Reason
			}
		}
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "メンバーがBANされました",
			IconURL: "https://cdn.discordapp.com/emojis/944622224769351700.png", // ハンマーの絵文字
		},
		Color: 0xED4245, // 赤色
		Fields: []*discordgo.MessageEmbedField{
			{Name: "対象ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
			{Name: "実行者", Value: fmt.Sprintf("<@%s>", executor), Inline: true},
			{Name: "理由", Value: reason, Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}

	s.ChannelMessageSendEmbed(logChannel, embed)
}

// ★★★ ここからが新しく追加する機能 ★★★

// HandleGuildMemberRemove はユーザーがKickされたか、自ら退出したときのイベントを処理します
func HandleGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	logChannel, ok := logChannelID[e.GuildID]
	if !ok {
		return
	}

	// Kickかどうかを監査ログで確認
	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberKick), 1)
	if len(auditLog.AuditLogEntries) > 0 {
		entry := auditLog.AuditLogEntries[0]
		// 監査ログの最新のKick対象が、退出したユーザーと一致するか確認
		if entry.TargetID == e.User.ID {
			logger.Info.Printf("User %s was kicked", e.User.Username)
			executor := entry.UserID
			reason := "理由の記載なし"
			if entry.Reason != "" {
				reason = entry.Reason
			}

			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{Name: "メンバーがKickされました", IconURL: "https://cdn.discordapp.com/emojis/944622224769351700.png"},
				Color:  0xFEE75C, // 黄色
				Fields: []*discordgo.MessageEmbedField{
					{Name: "対象ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
					{Name: "実行者", Value: fmt.Sprintf("<@%s>", executor), Inline: true},
					{Name: "理由", Value: reason, Inline: true},
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
			}
			s.ChannelMessageSendEmbed(logChannel, embed)
			return // Kick処理が終わったので、この後の退出処理はしない
		}
	}

	// Kickではなかった場合、通常の退出としてログを残す
	logger.Info.Printf("User %s left", e.User.Username)
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "メンバーが退出しました", IconURL: e.User.AvatarURL("")},
		Color:  0x34363C, // グレー
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}

// HandleGuildMemberUpdate はメンバーのロールやニックネーム、タイムアウトが変更されたときのイベントを処理します
func HandleGuildMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	logChannel, ok := logChannelID[e.GuildID]
	if !ok {
		return
	}

	// タイムアウトが設定されたか、解除されたかを確認
	// e.BeforeUpdate は更新前の情報。nilの場合もあるので注意
	if e.BeforeUpdate != nil && e.CommunicationDisabledUntil != e.BeforeUpdate.CommunicationDisabledUntil {
		// タイムアウトが設定された場合
		if e.CommunicationDisabledUntil != nil && e.CommunicationDisabledUntil.After(time.Now()) {
			logger.Info.Printf("User %s was timed out", e.User.Username)

			auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberUpdate), 1)
			executor := "不明"
			if len(auditLog.AuditLogEntries) > 0 {
				executor = auditLog.AuditLogEntries[0].UserID
			}

			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{Name: "メンバーがタイムアウトされました", IconURL: "https://cdn.discordapp.com/emojis/944622224769351700.png"},
				Color:  0xFEE75C, // 黄色
				Fields: []*discordgo.MessageEmbedField{
					{Name: "対象ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
					{Name: "実行者", Value: fmt.Sprintf("<@%s>", executor), Inline: true},
					{Name: "解除日時", Value: fmt.Sprintf("<t:%d:F>", e.CommunicationDisabledUntil.Unix()), Inline: true},
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
			}
			s.ChannelMessageSendEmbed(logChannel, embed)
		} else { // タイムアウトが解除された場合
			logger.Info.Printf("Timeout for user %s was removed", e.User.Username)
			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{Name: "メンバーのタイムアウトが解除されました", IconURL: e.User.AvatarURL("")},
				Color:  0x57F287, // 緑色
				Fields: []*discordgo.MessageEmbedField{
					{Name: "対象ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
			}
			s.ChannelMessageSendEmbed(logChannel, embed)
		}
	}
}

// HandleChannelCreate はチャンネルが作成されたときのイベントを処理します
func HandleChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) {
	logChannel, ok := logChannelID[e.GuildID]
	if !ok {
		return
	}

	logger.Info.Printf("Channel %s was created", e.Name)
	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionChannelCreate), 1)
	executor := "不明"
	if len(auditLog.AuditLogEntries) > 0 {
		executor = auditLog.AuditLogEntries[0].UserID
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "チャンネルが作成されました", IconURL: "https://cdn.discordapp.com/emojis/860602497069154364.png"}, // プラスの絵文字
		Color:  0x57F287,                                                                                                                  // 緑色
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s> (`%s`)", e.ID, e.Name), Inline: false},
			{Name: "作成者", Value: fmt.Sprintf("<@%s>", executor), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}

// HandleChannelDelete はチャンネルが削除されたときのイベントを処理します
func HandleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	logChannel, ok := logChannelID[e.GuildID]
	if !ok {
		return
	}

	logger.Info.Printf("Channel %s was deleted", e.Name)
	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionChannelDelete), 1)
	executor := "不明"
	if len(auditLog.AuditLogEntries) > 0 {
		executor = auditLog.AuditLogEntries[0].UserID
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "チャンネルが削除されました", IconURL: "https://cdn.discordapp.com/emojis/864921522055741440.png"}, // ゴミ箱の絵文字
		Color:  0xED4245,                                                                                                                  // 赤色
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル名", Value: fmt.Sprintf("`%s`", e.Name), Inline: false},
			{Name: "削除者", Value: fmt.Sprintf("<@%s>", executor), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}
