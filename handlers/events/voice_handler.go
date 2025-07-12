package events

import (
	"fmt"

	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

const (
	ConfigKeyTempVC = "temp_vc_config"
)

type VoiceHandler struct {
	Log   interfaces.Logger
	Store interfaces.DataStore
}

func NewVoiceHandler(log interfaces.Logger, store interfaces.DataStore) *VoiceHandler {
	return &VoiceHandler{Log: log, Store: store}
}

func (h *VoiceHandler) Register(s *discordgo.Session) {
	s.AddHandler(h.onVoiceStateUpdate)
}

func (h *VoiceHandler) onVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	var vcConfig storage.TempVCConfig
	if err := h.Store.GetConfig(e.GuildID, ConfigKeyTempVC, &vcConfig); err != nil || vcConfig.LobbyID == "" {
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
			h.Log.Error("一時VCの作成に失敗", "error", err)
			return
		}
		if err := s.GuildMemberMove(e.GuildID, e.UserID, &newChannel.ID); err != nil {
			h.Log.Error("Failed to move member to new channel", "error", err)
		}
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
					h.Log.Error("一時VCの削除に失敗", "error", err)
				}
			}
		}
	}
}
