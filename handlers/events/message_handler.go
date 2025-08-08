package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

// 定数を定義
const (
	ConfigKeyLog       = "log_config"
	AuditLogTimeWindow = 5 * time.Second
	ColorBlue          = 0x3498db
	ColorRed           = 0xe74c3c
	ColorGray          = 0x95a5a6
)

type TextRequest struct {
	Prompt string `json:"prompt"`
}
type TextResponse struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}

type MessageHandler struct {
	Log   interfaces.Logger
	Store interfaces.DataStore
}

func NewMessageHandler(log interfaces.Logger, store interfaces.DataStore) *MessageHandler {
	return &MessageHandler{Log: log, Store: store}
}

func (h *MessageHandler) Register(s *discordgo.Session) {
	s.AddHandler(h.onMessageCreate)
	s.AddHandler(h.onMessageUpdate)
	s.AddHandler(h.onMessageDelete)
}

func (h *MessageHandler) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// --- Word Count ---
	go func() {
		// サーバーでカウント対象に設定されている単語リストを取得
		countableWords, err := h.Store.GetCountableWords(m.GuildID)
		if err != nil {
			h.Log.Error("Failed to get countable words", "error", err, "guildID", m.GuildID)
			return
		}

		// メッセージがカウント対象の単語のいずれかと一致するかチェック
		for _, word := range countableWords {
			// TODO: 将来的には部分一致なども設定できるようにする
			if m.Content == word {
				if err := h.Store.IncrementWordCount(m.GuildID, m.Author.ID, word); err != nil {
					h.Log.Error("Failed to increment word count", "error", err, "guildID", m.GuildID, "userID", m.Author.ID, "word", word)
				}
				// 一致したらループを抜ける（複数の単語に一致してもカウントは1回）
				break
			}
		}
	}()
	// --- End Word Count ---

	if err := h.Store.CreateMessageCache(m.ID, m.Content, m.Author.ID); err != nil {
		h.Log.Error("Failed to cache message in DB", "error", err)
	}

	isMentioned := false
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			isMentioned = true
			break
		}
	}
	if isMentioned {
		go func() {
			if err := s.ChannelTyping(m.ChannelID); err != nil {
				h.Log.Warn("Failed to send typing indicator", "error", err)
			}
			messages, err := s.ChannelMessages(m.ChannelID, 15, m.ID, "", "")
			if err != nil {
				h.Log.Error("会話履歴の取得に失敗", "error", err)
				return
			}
			var history string
			for i := len(messages) - 1; i >= 0; i-- {
				msg := messages[i]
				history += fmt.Sprintf("%s: %s\n", msg.Author.Username, msg.Content)
			}
			persona := "あなたは「Luna Assistant」という名前の、高性能で親切なAIアシスタントです。過去の会話の文脈を理解し、自然な対話を行ってください。一人称は「私」を使い、常にフレンドリーで丁寧な言葉遣いを心がけてください。"
			prompt := fmt.Sprintf("システムインストラクション（あなたの役割）: %s\n\n以下の会話履歴の続きとして、あなたの次の発言を生成してください。\n\n[会話履歴]\n%s\nLuna Assistant:", persona, history)
			reqData := TextRequest{Prompt: prompt}
			reqJson, _ := json.Marshal(reqData)

			aiServerURL := os.Getenv("PYTHON_AI_SERVER_URL")
			if aiServerURL == "" {
				aiServerURL = "http://localhost:5001/generate-text" // Fallback to default
				h.Log.Warn("PYTHON_AI_SERVER_URL environment variable not set. Using default: " + aiServerURL)
			}

			resp, err := http.Post(aiServerURL, "application/json", bytes.NewBuffer(reqJson))
			if err != nil {
				if _, err := s.ChannelMessageSend(m.ChannelID, "すみません、AIサーバーへの接続に失敗したようです…。"); err != nil {
					h.Log.Error("Failed to send error message", "error", err)
				}
				h.Log.Error("Failed to connect to AI server", "error", err, "url", aiServerURL)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var textResp TextResponse
			if err := json.Unmarshal(body, &textResp); err != nil {
				h.Log.Error("Failed to unmarshal AI response", "error", err)
				return
			}
			if textResp.Error != "" || resp.StatusCode != http.StatusOK {
				if _, err := s.ChannelMessageSend(m.ChannelID, "すみません、AIからの応答取得に失敗しました…。"); err != nil {
					h.Log.Error("Failed to send error message", "error", err)
				}
				h.Log.Error("AI server returned an error or non-OK status", "status", resp.StatusCode, "response_error", textResp.Error)
				return
			}
			if _, err := s.ChannelMessageSend(m.ChannelID, textResp.Text); err != nil {
				h.Log.Error("Failed to send AI response", "error", err)
			}
		}()
	}
}

func (h *MessageHandler) OnMessageUpdate(s *discordgo.Session, e *discordgo.MessageUpdate) {
	if e.Author == nil || e.Author.Bot {
		return
	}
	cachedMsg, err := h.Store.GetMessageCache(e.ID)
	var embed *discordgo.MessageEmbed
	if err != nil || cachedMsg == nil {
		embed = &discordgo.MessageEmbed{
			Title: "✏️ メッセージ編集 (編集前は内容不明)", Color: ColorBlue,
			Author: &discordgo.MessageEmbedAuthor{Name: e.Author.String(), IconURL: e.Author.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "投稿者", Value: e.Author.Mention(), Inline: true},
				{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
				{Name: "メッセージ", Value: fmt.Sprintf("[リンク](https://discord.com/channels/%s/%s/%s)", e.GuildID, e.ChannelID, e.ID), Inline: true},
				{Name: "編集後", Value: "```\n" + e.Content + "\n```", Inline: false},
			},
		}
	} else {
		if e.Content == cachedMsg.Content {
			return
		}
		embed = &discordgo.MessageEmbed{
			Title: "✏️ メッセージ編集", Color: ColorBlue,
			Author: &discordgo.MessageEmbedAuthor{Name: e.Author.String(), IconURL: e.Author.AvatarURL("")},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "投稿者", Value: e.Author.Mention(), Inline: true},
				{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
				{Name: "メッセージ", Value: fmt.Sprintf("[リンク](https://discord.com/channels/%s/%s/%s)", e.GuildID, e.ChannelID, e.ID), Inline: true},
				{Name: "編集前", Value: "```\n" + cachedMsg.Content + "\n```", Inline: false},
				{Name: "編集後", Value: "```\n" + e.Content + "\n```", Inline: false},
			},
		}
	}
	if err := h.Store.CreateMessageCache(e.ID, e.Content, e.Author.ID); err != nil {
		h.Log.Error("Failed to create message cache", "error", err)
	}
	h.sendLog(s, e.GuildID, embed)
}

func (h *MessageHandler) OnMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	cachedMsg, err := h.Store.GetMessageCache(e.ID)
	if err != nil || cachedMsg == nil {
		// We don't have info, so just log the ID
		embed := &discordgo.MessageEmbed{
			Title:       "🗑️ メッセージ削除 (内容不明)",
			Description: fmt.Sprintf("<#%s> でメッセージが削除されました。", e.ChannelID),
			Color:       ColorGray,
			Fields:      []*discordgo.MessageEmbedField{{Name: "メッセージID", Value: e.ID}},
		}
		SendLog(s, e.GuildID, h.Store, h.Log, embed)
		return
	}

	author, err := s.User(cachedMsg.AuthorID)
	if err != nil {
		author = &discordgo.User{Username: "不明なユーザー", ID: cachedMsg.AuthorID}
	}

	deleterMention := "不明"
	// GetExecutorForMessageDelete is a special function to find who deleted the message
	deleterID := GetMessageDeleteExecutor(s, e.GuildID, cachedMsg.AuthorID, e.ChannelID, h.Log)
	if deleterID != "" {
		if deleterID == author.ID {
			deleterMention = "本人"
		} else {
			deleterMention = fmt.Sprintf("<@%s>", deleterID)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🗑️ メッセージ削除",
		Color:  ColorRed,
		Author: &discordgo.MessageEmbedAuthor{Name: author.String(), IconURL: author.AvatarURL("")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "投稿者", Value: author.Mention(), Inline: true},
			{Name: "削除者", Value: deleterMention, Inline: true},
			{Name: "チャンネル", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
			{Name: "内容", Value: "```\n" + cachedMsg.Content + "\n```", Inline: false},
		},
	}
	SendLog(s, e.GuildID, h.Store, h.Log, embed)
}

// sendLog and getExecutor are helper functions that might be used by other handlers as well.
// For now, we keep them here, but they could be moved to a shared utility package if needed.

func (h *MessageHandler) sendLog(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	var logConfig storage.LogConfig
	if err := h.Store.GetConfig(guildID, ConfigKeyLog, &logConfig); err != nil {
		h.Log.Error("Failed to get log config from DB", "error", err, "guildID", guildID)
		return
	}
	if logConfig.ChannelID == "" {
		return
	}

	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, _ = s.Guild(guildID)
	}
	if guild != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: guild.Name}
	}
	embed.Timestamp = time.Now().Format(time.RFC3339)
	if _, err := s.ChannelMessageSendEmbed(logConfig.ChannelID, embed); err != nil {
		h.Log.Error("Failed to send log embed", "error", err, "channelID", logConfig.ChannelID)
	}
}
