package events

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

type ChannelHandler struct {
	Log   interfaces.Logger
	Store interfaces.DataStore
}

func NewChannelHandler(log interfaces.Logger, store interfaces.DataStore) *ChannelHandler {
	return &ChannelHandler{Log: log, Store: store}
}

func (h *ChannelHandler) Register(s *discordgo.Session) {
	s.AddHandler(h.onChannelCreate)
	s.AddHandler(h.onChannelDelete)
	s.AddHandler(h.onChannelUpdate)
}

func (h *ChannelHandler) onChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) {
	executorID := GetExecutor(s, e.GuildID, e.ID, discordgo.AuditLogActionChannelCreate, h.Log)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	embed := &discordgo.MessageEmbed{
		Title: "📬 チャンネル作成",
		Color: ColorGreen,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s> (%s)", e.ID, e.Name), Inline: true},
			{Name: "種類", Value: ChannelTypeToString(e.Type), Inline: true},
			{Name: "実行者", Value: executorMention, Inline: true},
		},
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}

func (h *ChannelHandler) onChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	executorID := GetExecutor(s, e.GuildID, e.ID, discordgo.AuditLogActionChannelDelete, h.Log)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	embed := &discordgo.MessageEmbed{
		Title: "🗑️ チャンネル削除",
		Color: ColorRed,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル名", Value: e.Name, Inline: true},
			{Name: "種類", Value: ChannelTypeToString(e.Type), Inline: true},
			{Name: "実行者", Value: executorMention, Inline: true},
		},
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}

func (h *ChannelHandler) onChannelUpdate(s *discordgo.Session, e *discordgo.ChannelUpdate) {
	if e.BeforeUpdate == nil {
		return
	}

	// We only care about name changes for now
	if e.BeforeUpdate.Name == e.Name {
		return
	}

	executorID := GetExecutor(s, e.GuildID, e.ID, discordgo.AuditLogActionChannelUpdate, h.Log)
	executorMention := "不明"
	if executorID != "" {
		executorMention = fmt.Sprintf("<@%s>", executorID)
	}

	embed := &discordgo.MessageEmbed{
		Title: "✏️ チャンネル更新",
		Color: ColorBlue,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ID), Inline: false},
			{Name: "変更内容", Value: "名前の変更", Inline: true},
			{Name: "実行者", Value: executorMention, Inline: true},
			{Name: "変更前", Value: e.BeforeUpdate.Name, Inline: false},
			{Name: "変更後", Value: e.Name, Inline: false},
		},
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}
