package events

import (
	"luna/interfaces"
	"luna/storage"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// OnGuildMemberAdd は、新しいメンバーがサーバーに参加したときにトリガーされます。
func OnGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd, db interfaces.DataStore, log interfaces.Logger) {
	// ウェルカムメッセージの送信
	var welcomeConfig storage.WelcomeConfig
	if err := db.GetConfig(m.GuildID, "welcome_config", &welcomeConfig); err == nil && welcomeConfig.Enabled {
		msg := strings.Replace(welcomeConfig.Message, "{user}", m.User.Mention(), -1)
		msg = strings.Replace(msg, "{server}", m.GuildID, -1)
		s.ChannelMessageSend(welcomeConfig.ChannelID, msg)
	}

	// 自動ロールの付与
	var autoRoleConfig storage.AutoRoleConfig
	if err := db.GetConfig(m.GuildID, "autorole_config", &autoRoleConfig); err == nil && autoRoleConfig.Enabled {
		if err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, autoRoleConfig.RoleID); err != nil {
			log.Error("Failed to add autorole", "error", err, "userID", m.User.ID)
		}
	}

	// ログの送信
	embed := &discordgo.MessageEmbed{
		Title: "📥 メンバー参加",
		Color: 0x77b255, // Green
		Author: &discordgo.MessageEmbedAuthor{
			Name:    m.User.String(),
			IconURL: m.User.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ユーザー", Value: m.User.Mention(), Inline: true},
		},
	}
	SendLog(s, m.GuildID, db, log, embed)
}

// OnGuildMemberRemove は、メンバーがサーバーから退出またはキックされたときにトリガーされます。
func OnGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove, log interfaces.Logger, db interfaces.DataStore) {
	embed := &discordgo.MessageEmbed{
		Title: "📤 メンバー退出",
		Color: 0x992d22, // Dark Red
		Author: &discordgo.MessageEmbedAuthor{
			Name:    m.User.String(),
			IconURL: m.User.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ユーザー", Value: m.User.Mention(), Inline: true},
		},
	}
	SendLog(s, m.GuildID, db, log, embed)
}
