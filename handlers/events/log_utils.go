package events

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SendLog sends a log message to the configured log channel for a guild.
func SendLog(s *discordgo.Session, guildID string, store interfaces.DataStore, log interfaces.Logger, embed *discordgo.MessageEmbed) {
	var logConfig storage.LogConfig
	if err := store.GetConfig(guildID, "log_config", &logConfig); err != nil {
		// Don't log the error here to avoid potential logging loops
		return
	}
	if logConfig.ChannelID == "" {
		return
	}

	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, _ = s.Guild(guildID) // Fallback to API call
	}
	if guild != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: guild.Name}
	}
	embed.Timestamp = time.Now().Format(time.RFC3339)

	if _, err := s.ChannelMessageSendEmbed(logConfig.ChannelID, embed); err != nil {
		log.Error("Failed to send log embed", "error", err, "channelID", logConfig.ChannelID)
	}
}

// GetExecutor retrieves the user who performed an action from the audit log.
func GetExecutor(s *discordgo.Session, guildID string, targetID string, action discordgo.AuditLogAction, log interfaces.Logger) string {
	// Add a small delay to ensure audit log is populated
	time.Sleep(500 * time.Millisecond)

	auditLog, err := s.GuildAuditLog(guildID, "", "", int(action), 5)
	if err != nil {
		log.Error("Failed to get audit log", "error", err, "guildID", guildID, "action", action)
		return ""
	}
	for _, entry := range auditLog.AuditLogEntries {
		if entry.TargetID == targetID {
			logTime, _ := discordgo.SnowflakeTimestamp(entry.ID)
			// Use a slightly larger window to account for timing discrepancies
			if time.Since(logTime) < 15*time.Second {
				return entry.UserID
			}
		}
	}
	return ""
}

// GetMessageDeleteExecutor is a special version of GetExecutor for message delete events.
func GetMessageDeleteExecutor(s *discordgo.Session, guildID, authorID, channelID string, log interfaces.Logger) string {
	time.Sleep(500 * time.Millisecond)
	auditLog, err := s.GuildAuditLog(guildID, "", "", int(discordgo.AuditLogActionMessageDelete), 5)
	if err != nil {
		log.Error("Failed to get audit log for message delete", "error", err, "guildID", guildID)
		return ""
	}

	for _, entry := range auditLog.AuditLogEntries {
		// For message delete, TargetID is the user whose message was deleted.
		if entry.TargetID == authorID && entry.Options.ChannelID == channelID {
			logTime, _ := discordgo.SnowflakeTimestamp(entry.ID)
			if time.Since(logTime) < 15*time.Second {
				return entry.UserID
			}
		}
	}
	return ""
}

func ChannelTypeToString(t discordgo.ChannelType) string {
	switch t {
	case discordgo.ChannelTypeGuildText:
		return "テキストチャンネル"
	case discordgo.ChannelTypeGuildVoice:
		return "ボイスチャンネル"
	case discordgo.ChannelTypeGuildCategory:
		return "カテゴリ"
	case discordgo.ChannelTypeGuildNews:
		return "アナウンスチャンネル"
	case discordgo.ChannelTypeGuildStageVoice:
		return "ステージチャンネル"
	case discordgo.ChannelTypeGuildForum:
		return "フォーラムチャンネル"
	default:
		return fmt.Sprintf("不明な種類 (%d)", t)
	}
}
