package handlers

import (
	"database/sql"
	"fmt"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

type EventHandler struct {
	Store *storage.DBStore
}

func NewEventHandler(store *storage.DBStore) *EventHandler {
	return &EventHandler{Store: store}
}

func (h *EventHandler) RegisterAllHandlers(s *discordgo.Session) {
	s.AddHandler(h.handleMessageDelete)
	s.AddHandler(h.handleGuildBanAdd)
	s.AddHandler(h.handleGuildMemberAdd)
	s.AddHandler(h.handleGuildMemberRemove)
	s.AddHandler(h.handleVoiceStateUpdate)
	s.AddHandler(h.handleReactionAdd)
	s.AddHandler(h.handleReactionRemove)
}

func (h *EventHandler) logEvent(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	var logConfig storage.LogConfig
	if err := h.Store.GetConfig(guildID, "log_config", &logConfig); err != nil {
		logger.Error("ログ設定の取得に失敗", "error", err, "guildID", guildID)
		return
	}
	if logConfig.ChannelID == "" {
		return
	}
	s.ChannelMessageSendEmbed(logConfig.ChannelID, embed)
}

func (h *EventHandler) handleMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	embed := &discordgo.MessageEmbed{
		Title:       "メッセージ削除",
		Description: fmt.Sprintf("メッセージが削除されました。\n**チャンネル:** <#%s>", e.ChannelID),
		Color:       0xffa500,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	embed := &discordgo.MessageEmbed{
		Title:       "ユーザーがBANされました",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0xff0000,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	embed := &discordgo.MessageEmbed{
		Title:       "メンバー参加",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0x00ff00,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	embed := &discordgo.MessageEmbed{
		Title:       "メンバー退出",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0xaaaaaa,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}
	rr, err := h.Store.GetReactionRole(r.GuildID, r.MessageID, r.Emoji.APIName())
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Error("リアクションロールのDB取得に失敗", "error", err, "guildID", r.GuildID)
		}
		return
	}
	err = s.GuildMemberRoleAdd(r.GuildID, r.UserID, rr.RoleID)
	if err != nil {
		logger.Error("ロールの付与に失敗", "error", err, "userID", r.UserID, "roleID", rr.RoleID)
	}
}

func (h *EventHandler) handleReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}
	rr, err := h.Store.GetReactionRole(r.GuildID, r.MessageID, r.Emoji.APIName())
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Error("リアクションロールのDB取得に失敗", "error", err, "guildID", r.GuildID)
		}
		return
	}
	err = s.GuildMemberRoleRemove(r.GuildID, r.UserID, rr.RoleID)
	if err != nil {
		logger.Error("ロールの削除に失敗", "error", err, "userID", r.UserID, "roleID", rr.RoleID)
	}
}

func (h *EventHandler) handleVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	var vcConfig storage.TempVCConfig
	if err := h.Store.GetConfig(e.GuildID, "temp_vc_config", &vcConfig); err != nil || vcConfig.LobbyID == "" {
		return
	}

	if e.ChannelID == vcConfig.LobbyID {
		member, err := s.State.Member(e.GuildID, e.UserID)
		if err != nil {
			member, err = s.GuildMember(e.GuildID, e.UserID)
			if err != nil {
				return
			}
		}

		newChannel, err := s.GuildChannelCreateComplex(e.GuildID, discordgo.GuildChannelCreateData{
			Name:     fmt.Sprintf("%sの部屋", member.User.Username),
			Type:     discordgo.ChannelTypeGuildVoice,
			ParentID: vcConfig.CategoryID,
		})
		if err != nil {
			logger.Error("一時VCの作成に失敗", "error", err)
			return
		}
		s.GuildMemberMove(e.GuildID, e.UserID, &newChannel.ID)
	}

	if e.BeforeUpdate != nil && e.BeforeUpdate.ChannelID != "" && e.BeforeUpdate.ChannelID != vcConfig.LobbyID {
		oldChannel, err := s.Channel(e.BeforeUpdate.ChannelID)
		if err != nil {
			return
		}

		if oldChannel.ParentID == vcConfig.CategoryID {
			guild, err := s.State.Guild(e.GuildID)
			if err != nil {
				return
			}
			found := false
			for _, vs := range guild.VoiceStates {
				if vs.ChannelID == oldChannel.ID {
					found = true
					break
				}
			}
			if !found {
				if _, err := s.ChannelDelete(oldChannel.ID); err != nil {
					logger.Error("一時VCの削除に失敗", "error", err)
				}
			}
		}
	}
}
