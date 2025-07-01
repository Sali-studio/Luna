package handlers

import (
	"fmt"
	"luna/logger"
	"luna/storage"
	"strings"
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
	// Logging
	s.AddHandler(h.handleMessageDelete)
	s.AddHandler(h.handleMessageUpdate)
	s.AddHandler(h.handleGuildBanAdd)
	s.AddHandler(h.handleGuildBanRemove)
	s.AddHandler(h.handleGuildMemberAdd)
	s.AddHandler(h.handleGuildMemberRemove)
	s.AddHandler(h.handleVoiceStateUpdate)
	// Reaction Roles
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
		Description: fmt.Sprintf("**チャンネル:** <#%s>\n**メッセージID:** `%s`", e.ChannelID, e.ID),
		Color:       0xffa500, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	h.logEvent(s, e.GuildID, embed)
}

// (handleMessageUpdate, handleGuildBanAdd... などの他のログハンドラも同様にここに実装)

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

	// ロビーチャンネルに入室したか
	if e.ChannelID == config.TempVC.LobbyID {
		// 新しいVCを作成
		newChannel, err := s.GuildChannelCreateComplex(e.GuildID, discordgo.GuildChannelCreateData{
			Name:     fmt.Sprintf("%sの部屋", e.Member.User.Username),
			Type:     discordgo.ChannelTypeGuildVoice,
			ParentID: config.TempVC.CategoryID,
			// (パーミッション設定など)
		})
		if err != nil {
			logger.Error.Printf("一時VCの作成に失敗: %v", err)
			return
		}
		// ユーザーを新しいVCに移動
		s.GuildMemberMove(e.GuildID, e.UserID, &newChannel.ID)
	}

	// 古いチャンネルが一時VCで、誰もいなくなったかチェック
	if e.BeforeUpdate != nil {
		oldChannel, err := s.Channel(e.BeforeUpdate.ChannelID)
		if err != nil {
			return
		}

		// チャンネル名やカテゴリで一時VCか判断する（より堅牢な方法が望ましい）
		if strings.HasSuffix(oldChannel.Name, "の部屋") && oldChannel.ParentID == config.TempVC.CategoryID {
			members, _ := s.State.VoiceState(e.GuildID, oldChannel.ID)
			if members == nil || len(members.Members) == 0 {
				s.ChannelDelete(oldChannel.ID)
			}
		}
	}
}
