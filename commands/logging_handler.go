package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

func HandleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
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
			IconURL: "https://cdn.discordapp.com/emojis/944622224769351700.png",
		},
		Color: 0xED4245,
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

func HandleGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
		return
	}

	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberKick), 1)
	if len(auditLog.AuditLogEntries) > 0 {
		entry := auditLog.AuditLogEntries[0]
		if entry.TargetID == e.User.ID {
			logger.Info.Printf("User %s was kicked", e.User.Username)
			executor := entry.UserID
			reason := "理由の記載なし"
			if entry.Reason != "" {
				reason = entry.Reason
			}

			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{Name: "メンバーがKickされました", IconURL: "https://cdn.discordapp.com/emojis/944622224769351700.png"},
				Color:  0xFEE75C,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "対象ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
					{Name: "実行者", Value: fmt.Sprintf("<@%s>", executor), Inline: true},
					{Name: "理由", Value: reason, Inline: true},
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
			}
			s.ChannelMessageSendEmbed(logChannel, embed)
			return
		}
	}

	logger.Info.Printf("User %s left", e.User.Username)
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "メンバーが退出しました", IconURL: e.User.AvatarURL("")},
		Color:  0x34363C,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}

func HandleGuildMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
		return
	}

	if e.BeforeUpdate != nil && e.CommunicationDisabledUntil != e.BeforeUpdate.CommunicationDisabledUntil {
		if e.CommunicationDisabledUntil != nil && e.CommunicationDisabledUntil.After(time.Now()) {
			logger.Info.Printf("User %s was timed out", e.User.Username)

			auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMemberUpdate), 1)
			executor := "不明"
			if len(auditLog.AuditLogEntries) > 0 {
				executor = auditLog.AuditLogEntries[0].UserID
			}

			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{Name: "メンバーがタイムアウトされました", IconURL: "https://cdn.discordapp.com/emojis/944622224769351700.png"},
				Color:  0xFEE75C,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "対象ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
					{Name: "実行者", Value: fmt.Sprintf("<@%s>", executor), Inline: true},
					{Name: "解除日時", Value: fmt.Sprintf("<t:%d:F>", e.CommunicationDisabledUntil.Unix()), Inline: true},
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
			}
			s.ChannelMessageSendEmbed(logChannel, embed)
		} else {
			logger.Info.Printf("Timeout for user %s was removed", e.User.Username)
			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{Name: "メンバーのタイムアウトが解除されました", IconURL: e.User.AvatarURL("")},
				Color:  0x57F287,
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

func HandleChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
		return
	}

	logger.Info.Printf("Channel %s was created", e.Name)
	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionChannelCreate), 1)
	executor := "不明"
	if len(auditLog.AuditLogEntries) > 0 {
		executor = auditLog.AuditLogEntries[0].UserID
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "チャンネルが作成されました", IconURL: "https://cdn.discordapp.com/emojis/860602497069154364.png"},
		Color:  0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s> (`%s`)", e.ID, e.Name), Inline: false},
			{Name: "作成者", Value: fmt.Sprintf("<@%s>", executor), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}

func HandleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
		return
	}

	logger.Info.Printf("Channel %s was deleted", e.Name)
	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionChannelDelete), 1)
	executor := "不明"
	if len(auditLog.AuditLogEntries) > 0 {
		executor = auditLog.AuditLogEntries[0].UserID
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "チャンネルが削除されました", IconURL: "https://cdn.discordapp.com/emojis/864921522055741440.png"},
		Color:  0xED4245,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル名", Value: fmt.Sprintf("`%s`", e.Name), Inline: false},
			{Name: "削除者", Value: fmt.Sprintf("<@%s>", executor), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}

// HandleGuildMemberAddLog はユーザーがサーバーに参加したときのイベントを処理します
func HandleGuildMemberAddLog(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
		return
	}

	logger.Info.Printf("User %s joined guild %s", e.User.Username, e.GuildID)

	createdAt, _ := discordgo.SnowflakeTimestamp(e.User.ID)

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "メンバーが参加しました",
			IconURL: e.User.AvatarURL(""),
		},
		Color: 0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ユーザー", Value: fmt.Sprintf("%s (`%s`)", e.User.Mention(), e.User.ID), Inline: false},
			{Name: "アカウント作成日", Value: fmt.Sprintf("<t:%d:F>", createdAt.Unix()), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}

	s.ChannelMessageSendEmbed(logChannel, embed)
}

// HandleMessageDelete はメッセージが削除されたときのイベントを処理します
func HandleMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
		return
	}

	logger.Info.Printf("Message %s was deleted from channel %s", e.ID, e.ChannelID)

	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", int(discordgo.AuditLogActionMessageDelete), 1)
	executor := "不明"
	if len(auditLog.AuditLogEntries) > 0 {
		entry := auditLog.AuditLogEntries[0]
		if entry.Options != nil && entry.Options.ChannelID == e.ChannelID {
			executor = entry.UserID
		}
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "メッセージが削除されました", IconURL: "https://cdn.discordapp.com/emojis/864921522055741440.png"},
		Color:  0xED4245,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: false},
			{Name: "メッセージID", Value: e.ID, Inline: false},
			{Name: "削除実行者", Value: fmt.Sprintf("<@%s>", executor), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}

// HandleWebhooksUpdate はWebhookが作成・更新・削除されたときのイベントを処理します
func HandleWebhooksUpdate(s *discordgo.Session, e *discordgo.WebhooksUpdate) {
	config := Config.GetGuildConfig(e.GuildID)
	logChannel := config.Log.ChannelID
	if logChannel == "" {
		return
	}

	logger.Info.Printf("Webhooks were updated in channel %s", e.ChannelID)

	auditLog, _ := s.GuildAuditLog(e.GuildID, "", "", -1, 5)
	executor := "不明"
	action := "更新"

	if len(auditLog.AuditLogEntries) > 0 {
		for _, entry := range auditLog.AuditLogEntries {
			// nilチェックを追加し、ポインタをデリファレンス(*)して比較
			if entry.ActionType != nil && (*entry.ActionType == discordgo.AuditLogActionWebhookCreate || *entry.ActionType == discordgo.AuditLogActionWebhookDelete || *entry.ActionType == discordgo.AuditLogActionWebhookUpdate) {
				executor = entry.UserID
				switch *entry.ActionType {
				case discordgo.AuditLogActionWebhookCreate:
					action = "作成"
				case discordgo.AuditLogActionWebhookDelete:
					action = "削除"
				case discordgo.AuditLogActionWebhookUpdate:
					action = "更新"
				}
				break
			}
		}
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: "Webhookが更新されました", IconURL: "https://cdn.discordapp.com/emojis/864921521191550997.png"},
		Color:  0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "対象チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: false},
			{Name: "操作", Value: action, Inline: true},
			{Name: "実行者", Value: fmt.Sprintf("<@%s>", executor), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: "Luna Logging System"},
	}
	s.ChannelMessageSendEmbed(logChannel, embed)
}
