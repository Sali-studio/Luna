package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"luna/logger"
	"net/http"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca/v2"
)

// 各サーバーの音楽再生状態を管理
var musicSessions = make(map[string]*MusicSession)
var musicMutex = &sync.Mutex{}

// MusicSession は、1つのサーバーでの音楽再生セッションを表す
type MusicSession struct {
	GuildID         string
	VoiceConnection *discordgo.VoiceConnection
	Queue           []Song
	NowPlaying      *Song
	IsPlaying       bool
	Mutex           sync.Mutex
	EncodeSession   *dca.EncodeSession // エンコードセッションを直接保持
}

// Song は再生する曲の情報を表す
type Song struct {
	Title     string
	StreamURL string
	Query     string
	Requester *discordgo.User
}

type MusicCommand struct{}

func (c *MusicCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "music",
		Description: "音楽を再生します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "play",
				Description: "指定した曲を再生またはキューに追加します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "query", Description: "YouTubeのURLまたは検索ワード", Required: true},
				},
			},
			{
				Name:        "skip",
				Description: "現在再生中の曲をスキップします",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "stop",
				Description: "音楽を停止し、BotがVCから切断します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "queue",
				Description: "再生待ちの曲一覧を表示します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *MusicCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subcommand := i.ApplicationCommandData().Options[0]
	switch subcommand.Name {
	case "play":
		c.handlePlay(s, i)
	case "skip":
		c.handleSkip(s, i)
	case "stop":
		c.handleStop(s, i)
	case "queue":
		c.handleQueue(s, i)
	}
}

func getOrCreateSession(guildID string) *MusicSession {
	musicMutex.Lock()
	defer musicMutex.Unlock()

	if session, ok := musicSessions[guildID]; ok {
		return session
	}

	musicSessions[guildID] = &MusicSession{
		GuildID: guildID,
		Queue:   make([]Song, 0),
	}
	return musicSessions[guildID]
}

func (c *MusicCommand) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})

	query := i.ApplicationCommandData().Options[0].Options[0].StringValue()
	session := getOrCreateSession(i.GuildID)

	vs, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &[]string{"❌ まずボイスチャンネルに参加してください。"}[0]})
		return
	}

	reqData := map[string]interface{}{"query": query}
	jsonData, _ := json.Marshal(reqData)
	resp, err := http.Post("http://localhost:5002/get-stream-url", "application/json", bytes.NewBuffer(jsonData))

	if err != nil || resp.StatusCode != http.StatusOK {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &[]string{"❌ 曲情報の取得に失敗しました。"}[0]})
		logger.Error("Failed to get stream url", "error", err)
		return
	}

	var songInfo struct {
		StreamURL string `json:"stream_url"`
		Title     string `json:"title"`
	}
	json.NewDecoder(resp.Body).Decode(&songInfo)
	resp.Body.Close()

	song := Song{
		Title:     songInfo.Title,
		StreamURL: songInfo.StreamURL,
		Query:     query,
		Requester: i.Member.User,
	}

	session.Mutex.Lock()
	session.Queue = append(session.Queue, song)
	queueLen := len(session.Queue)
	isPlaying := session.IsPlaying
	session.Mutex.Unlock()

	if isPlaying {
		content := fmt.Sprintf("🎵 **%s** をキューの%d番目に追加しました。", song.Title, queueLen)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
	} else {
		vc, err := s.ChannelVoiceJoin(i.GuildID, vs.ChannelID, false, true)
		if err != nil {
			logger.Error("Failed to join voice channel", "error", err)
			return
		}
		session.VoiceConnection = vc
		go playMusic(session) // 再生ループを開始
		content := fmt.Sprintf("▶️ **%s** の再生を開始します。", song.Title)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
	}
}

func (c *MusicCommand) handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	session, ok := musicSessions[i.GuildID]
	if !ok || !session.IsPlaying {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "❌ 再生中の曲がありません。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	if session.EncodeSession != nil {
		session.EncodeSession.Stop()
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "⏩ スキップしました。"}})
}

func (c *MusicCommand) handleStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	session, ok := musicSessions[i.GuildID]
	if !ok {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "❌ BotはVCに参加していません。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	session.Mutex.Lock()
	session.Queue = make([]Song, 0)
	if session.IsPlaying && session.EncodeSession != nil {
		session.EncodeSession.Stop()
	}
	session.Mutex.Unlock()

	time.Sleep(250 * time.Millisecond)
	session.VoiceConnection.Disconnect()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "⏹️ 再生を停止し、切断しました。"}})
}

func (c *MusicCommand) handleQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	session, ok := musicSessions[i.GuildID]
	if !ok || (session.NowPlaying == nil && len(session.Queue) == 0) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "キューは空です。"}})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎵 再生キュー",
		Color: 0x5865F2,
	}

	session.Mutex.Lock()
	defer session.Mutex.Unlock()

	if session.NowPlaying != nil {
		embed.Description = fmt.Sprintf("**現在再生中:**\n[%s](%s) | `リクエスト: %s`\n\n", session.NowPlaying.Title, session.NowPlaying.Query, session.NowPlaying.Requester.Username)
	}

	if len(session.Queue) > 0 {
		var queueText string
		for i, song := range session.Queue {
			if i > 9 {
				queueText += fmt.Sprintf("\n...他%d曲", len(session.Queue)-10)
				break
			}
			queueText += fmt.Sprintf("**%d.** [%s](%s) | `リクエスト: %s`\n", i+1, song.Title, song.Query, song.Requester.Username)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "再生待ち",
			Value: queueText,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

// playMusicは音楽再生のメインループ
func playMusic(session *MusicSession) {
	// この関数が終了したら（キューが空になったら）、VCから切断しセッション情報を削除する
	defer func() {
		logger.Info("キューが空になったため再生を終了し、VCから切断します。")
		session.VoiceConnection.Disconnect()
		musicMutex.Lock()
		delete(musicSessions, session.GuildID)
		musicMutex.Unlock()
	}()

	for {
		session.Mutex.Lock()
		if len(session.Queue) == 0 {
			session.IsPlaying = false
			session.NowPlaying = nil
			session.Mutex.Unlock()
			logger.Info("キューが空になりました。再生ループを終了します。")
			return // キューが空になったらループを終了
		}

		song := session.Queue[0]
		session.Queue = session.Queue[1:]
		session.NowPlaying = &song
		session.IsPlaying = true
		logger.Info("次の曲の再生を開始します。", "title", song.Title)
		session.Mutex.Unlock()

		opts := &dca.EncodeOptions{
			Volume:        256,
			Channels:      2,
			FrameRate:     48000,
			FrameDuration: 20,
			Bitrate:       96,
			Application:   dca.AudioApplicationLowDelay,
			RawOutput:     true,
		}

		encodingSession, err := dca.EncodeFile(song.StreamURL, opts)
		if err != nil {
			logger.Error("DCAエンコードセッションの作成に失敗しました。", "error", err, "title", song.Title)
			continue // 次の曲へ
		}

		session.EncodeSession = encodingSession // スキップ/ストップ用にセッションを保持

		// ボイスチャンネルに送信を開始
		session.VoiceConnection.Speaking(true)
		logger.Info("音声のストリーミングを開始します。")

		for {
			opus, err := encodingSession.OpusFrame()
			if err != nil {
				// EOFは曲の正常な終了
				if !errors.Is(err, io.EOF) {
					logger.Error("Opusフレームの読み込み中にエラーが発生しました。", "error", err)
				} else {
					logger.Info("曲が正常に終了しました(EOF)。")
				}
				break // 曲の終わりかエラー
			}
			session.VoiceConnection.OpusSend <- opus
		}

		session.VoiceConnection.Speaking(false)
		logger.Info("音声のストリーミングが完了しました。")

		// 曲が終わるごとに必ずCleanupを呼び出す
		encodingSession.Cleanup()
		session.EncodeSession = nil // 参照をクリア
		logger.Info("エンコードセッションをクリーンアップしました。")
	}
}

func (c *MusicCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *MusicCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *MusicCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *MusicCommand) GetCategory() string                                                  { return "音楽" }
