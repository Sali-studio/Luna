package handlers

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
)

// EventHandler はコマンド以外のイベントを処理します
type EventHandler struct {
	Store *storage.ConfigStore
}

func NewEventHandler(store *storage.ConfigStore) *EventHandler {
	return &EventHandler{Store: store}
}

// RegisterAllHandlers はこのハンドラが処理する全てのイベントをdiscordgoセッションに登録します
func (h *EventHandler) RegisterAllHandlers(s *discordgo.Session) {
	s.AddHandler(h.handleMessageDelete)
	s.AddHandler(h.handleGuildBanAdd)
	s.AddHandler(h.handleGuildMemberAdd)
	s.AddHandler(h.handleGuildMemberRemove)
	s.AddHandler(h.handleVoiceStateUpdate)
	s.AddHandler(h.handleReactionAdd)
	s.AddHandler(h.handleReactionRemove)
}

// --- Logging ---

func (h *EventHandler) logEvent(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	config := h.Store.GetGuildConfig(guildID)
	if config.Log.ChannelID == "" {
		return
	}
	s.ChannelMessageSendEmbed(config.Log.ChannelID, embed)
}

func (h *EventHandler) handleMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	embed := &discordgo.MessageEmbed{
		Title:       "メッセージ削除",
		Description: fmt.Sprintf("メッセージが削除されました。\n**チャンネル:** <#%s>", e.ChannelID),
		Color:       0xffa500, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	embed := &discordgo.MessageEmbed{
		Title:       "ユーザーがBANされました",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	embed := &discordgo.MessageEmbed{
		Title:       "メンバー参加",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

func (h *EventHandler) handleGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	embed := &discordgo.MessageEmbed{
		Title:       "メンバー退出",
		Description: fmt.Sprintf("**ユーザー:** %s (`%s`)", e.User.String(), e.User.ID),
		Color:       0xaaaaaa, // Grey
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

// --- Reaction Roles ---

func (h *EventHandler) handleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}
	config := h.Store.GetGuildConfig(r.GuildID)
	key := fmt.Sprintf("%s_%s", r.MessageID, r.Emoji.APIName())
	roleID, ok := config.ReactionRoles[key]
	if !ok {
		return
	}
	err := s.GuildMemberRoleAdd(r.GuildID, r.UserID, roleID)
	if err != nil {
		logger.Error.Printf("ロールの付与に失敗 (User: %s, Role: %s): %v", r.UserID, roleID, err)
	}
}

func (h *EventHandler) handleReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}
	config := h.Store.GetGuildConfig(r.GuildID)
	key := fmt.Sprintf("%s_%s", r.MessageID, r.Emoji.APIName())
	roleID, ok := config.ReactionRoles[key]
	if !ok {
		return
	}
	err := s.GuildMemberRoleRemove(r.GuildID, r.UserID, roleID)
	if err != nil {
		logger.Error.Printf("ロールの削除に失敗 (User: %s, Role: %s): %v", r.UserID, roleID, err)
	}
}

// --- Temp Voice Channel ---

func (h *EventHandler) handleVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	config := h.Store.GetGuildConfig(e.GuildID)
	if config.TempVC.LobbyID == "" {
		return
	}

	// ロビーチャンネルに入室した
	if e.ChannelID == config.TempVC.LobbyID {
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
			ParentID: config.TempVC.CategoryID,
		})
		if err != nil {
			logger.Error.Printf("一時VCの作成に失敗: %v", err)
			return
		}

		s.GuildMemberMove(e.GuildID, e.UserID, &newChannel.ID)
	}

	// 古いチャンネルが一時VCで、誰もいなくなったかチェック
	if e.BeforeUpdate != nil && e.BeforeUpdate.ChannelID != config.TempVC.LobbyID {
		oldChannel, err := s.Channel(e.BeforeUpdate.ChannelID)
		if err != nil {
			return
		}

		// カテゴリで一時VCか判断
		if oldChannel.ParentID == config.TempVC.CategoryID {
			members, _ := s.State.VoiceState(e.GuildID, oldChannel.ID)
			if members == nil || len(members.Members) == 0 {
				_, err := s.ChannelDelete(oldChannel.ID)
				if err != nil {
					logger.Error.Printf("一時VCの削除に失敗: %v", err)
				}
			}
		}
	}
}
