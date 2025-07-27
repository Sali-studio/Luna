package events

import (
	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

// MemberEventHandler はメンバー関連のイベントを処理します。
type MemberEventHandler struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

// NewMemberEventHandler は新しいMemberEventHandlerを返します。
func NewMemberEventHandler(store interfaces.DataStore, log interfaces.Logger) *MemberEventHandler {
	return &MemberEventHandler{Store: store, Log: log}
}

// RegisterHandlers はこのハンドラのイベントリスナーを登録します。
func (h *MemberEventHandler) RegisterHandlers(s *discordgo.Session) {
	s.AddHandler(h.onGuildMemberAdd)
}

// onGuildMemberAdd はユーザーがサーバーに参加したときにトリガーされます。
func (h *MemberEventHandler) onGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	// 自動ロール付与の処理
	var config storage.AutoRoleConfig
	err := h.Store.GetConfig(m.GuildID, "autorole_config", &config)
	if err != nil {
		h.Log.Error("Failed to get autorole config on member add", "error", err, "guild_id", m.GuildID)
		return
	}

	if config.Enabled && config.RoleID != "" {
		err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, config.RoleID)
		if err != nil {
			h.Log.Error("Failed to add autorole to member", "error", err, "user_id", m.User.ID, "role_id", config.RoleID)
		} else {
			h.Log.Info("Successfully added autorole to member", "user_id", m.User.ID, "role_id", config.RoleID)
		}
	}
}