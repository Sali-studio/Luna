package handlers

import (
	"luna/handlers/events"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// EventHandler は、すべてのイベントハンドラをまとめる構造体です。
// 各イベントの具体的なロジックは handlers/events ディレクトリに委譲します。
type EventHandler struct {
	log               interfaces.Logger
	db                interfaces.DataStore
	commandHandlers   map[string]interfaces.CommandHandler
	componentHandlers map[string]interfaces.CommandHandler

	// Individual handlers
	messageHandler *events.MessageHandler
	channelHandler *events.ChannelHandler
	roleHandler    *events.RoleHandler
	voiceHandler   *events.VoiceHandler
	memberHandler  *events.MemberEventHandler
}

// NewEventHandler は、すべてのイベントハンドラを初期化してラップする新しいEventHandlerを返します。
func NewEventHandler(log interfaces.Logger, db interfaces.DataStore, commandHandlers, componentHandlers map[string]interfaces.CommandHandler) *EventHandler {
	return &EventHandler{
		log:               log,
		db:                db,
		commandHandlers:   commandHandlers,
		componentHandlers: componentHandlers,
		messageHandler:    events.NewMessageHandler(log, db),
		channelHandler:    events.NewChannelHandler(log, db),
		roleHandler:       events.NewRoleHandler(log, db),
		voiceHandler:      events.NewVoiceHandler(log, db),
		memberHandler:     events.NewMemberEventHandler(db, log),
	}
}

// OnReady は、Botの準備ができたときに呼び出されます。
func (h *EventHandler) OnReady(s *discordgo.Session, r *discordgo.Ready) {
	events.OnReady(s, r, h.log)
}

// OnInteractionCreate は、インタラクション（コマンド、ボタンなど）が作成されたときに呼び出されます。
func (h *EventHandler) OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	events.OnInteractionCreate(s, i, h.commandHandlers, h.componentHandlers, h.log)
}

// OnMessageCreate は、メッセージが作成されたときに呼び出されます。
func (h *EventHandler) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	h.messageHandler.OnMessageCreate(s, m)
}

// OnMessageDelete は、メッセージが削除されたときに呼び出されます。
func (h *EventHandler) OnMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	h.messageHandler.OnMessageDelete(s, m)
}

// OnGuildMemberAdd は、メンバーがギルドに参加したときに呼び出されます。
func (h *EventHandler) OnGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	h.memberHandler.OnGuildMemberAdd(s, m)
}

// OnGuildMemberRemove は、メンバーがギルドから退出または追放されたときに呼び出されます。
func (h *EventHandler) OnGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	events.OnGuildMemberRemove(s, m, h.log, h.db)
}

// OnVoiceStateUpdate は、ボイスステートが更新されたときに呼び出されます。
func (h *EventHandler) OnVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	h.voiceHandler.OnVoiceStateUpdate(s, v)
}

// OnChannelCreate は、チャンネルが作成されたときに呼び出されます。
func (h *EventHandler) OnChannelCreate(s *discordgo.Session, c *discordgo.ChannelCreate) {
	h.channelHandler.OnChannelCreate(s, c)
}

// OnChannelDelete は、チャンネルが削除されたときに呼び出されます。
func (h *EventHandler) OnChannelDelete(s *discordgo.Session, c *discordgo.ChannelDelete) {
	h.channelHandler.OnChannelDelete(s, c)
}

// OnGuildRoleCreate は、ロールが作成されたときに呼び出されます。
func (h *EventHandler) OnGuildRoleCreate(s *discordgo.Session, r *discordgo.GuildRoleCreate) {
	h.roleHandler.OnRoleCreate(s, r)
}

// OnGuildRoleDelete は、ロールが削除されたときに呼び出されます。
func (h *EventHandler) OnGuildRoleDelete(s *discordgo.Session, r *discordgo.GuildRoleDelete) {
	h.roleHandler.OnRoleDelete(s, r)
}