package events

import (
	"database/sql"

	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

type ReactionHandler struct {
	Log   interfaces.Logger
	Store interfaces.DataStore
}

func NewReactionHandler(log interfaces.Logger, store interfaces.DataStore) *ReactionHandler {
	return &ReactionHandler{Log: log, Store: store}
}

func (h *ReactionHandler) Register(s *discordgo.Session) {
	s.AddHandler(h.onReactionAdd)
	s.AddHandler(h.onReactionRemove)
}

func (h *ReactionHandler) onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}
	rr, err := h.Store.GetReactionRole(r.GuildID, r.MessageID, r.Emoji.APIName())
	if err != nil {
		if err != sql.ErrNoRows {
			h.Log.Error("リアクションロールのDB取得に失敗", "error", err, "guildID", r.GuildID)
		}
		return
	}
	if err = s.GuildMemberRoleAdd(r.GuildID, r.UserID, rr.RoleID); err != nil {
		h.Log.Error("ロールの付与に失敗", "error", err, "userID", r.UserID, "roleID", rr.RoleID)
	}
}

func (h *ReactionHandler) onReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}
	rr, err := h.Store.GetReactionRole(r.GuildID, r.MessageID, r.Emoji.APIName())
	if err != nil {
		if err != sql.ErrNoRows {
			h.Log.Error("リアクションロールのDB取得に失敗", "error", err, "guildID", r.GuildID)
		}
		return
	}
	if err = s.GuildMemberRoleRemove(r.GuildID, r.UserID, rr.RoleID); err != nil {
		h.Log.Error("ロールの削除に失敗", "error", err, "userID", r.UserID, "roleID", rr.RoleID)
	}
}
