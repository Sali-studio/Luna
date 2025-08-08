package events

import (
	"fmt"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

type RoleHandler struct {
	Log   interfaces.Logger
	Store interfaces.DataStore
}

func NewRoleHandler(log interfaces.Logger, store interfaces.DataStore) *RoleHandler {
	return &RoleHandler{Log: log, Store: store}
}

func (h *RoleHandler) Register(s *discordgo.Session) {
	s.AddHandler(h.OnRoleCreate)
	s.AddHandler(h.OnRoleDelete)
	s.AddHandler(h.OnRoleUpdate)
}

func (h *RoleHandler) OnRoleCreate(s *discordgo.Session, e *discordgo.GuildRoleCreate) {
	executorID := GetExecutor(s, e.GuildID, e.Role.ID, discordgo.AuditLogActionRoleCreate, h.Log)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	embed := &discordgo.MessageEmbed{
		Title: "✨ ロール作成",
				Color:       0x77b255, // Green
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ロール", Value: fmt.Sprintf("<@&%s> (%s)", e.Role.ID, e.Role.Name), Inline: true},
			{Name: "実行者", Value: executorMention, Inline: true},
		},
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}

func (h *RoleHandler) OnRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) {
	executorID := GetExecutor(s, e.GuildID, e.RoleID, discordgo.AuditLogActionRoleDelete, h.Log)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	// We don't have the role name after it's deleted, so we just use the ID.
	embed := &discordgo.MessageEmbed{
		Title: "🗑️ ロール削除",
		Color: ColorRed,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ロールID", Value: e.RoleID, Inline: true},
			{Name: "実行者", Value: executorMention, Inline: true},
		},
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}

func (h *RoleHandler) OnRoleUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) {
	before, err := s.State.Role(e.GuildID, e.Role.ID)
	if err != nil {
		// If we can't get the before state, we can't compare.
		return
	}

	// For now, we only log name changes.
	if e.Role.Name == before.Name {
		return
	}

	executorID := GetExecutor(s, e.GuildID, e.Role.ID, discordgo.AuditLogActionRoleUpdate, h.Log)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	embed := &discordgo.MessageEmbed{
		Title: "✏️ ロール更新",
		Color: ColorBlue,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ロール", Value: fmt.Sprintf("<@&%s>", e.Role.ID), Inline: false},
			{Name: "変更内容", Value: "名前の変更", Inline: true},
			{Name: "実行者", Value: executorMention, Inline: true},
			{Name: "変更前", Value: before.Name, Inline: false},
			{Name: "変更後", Value: e.Role.Name, Inline: false},
		},
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}
