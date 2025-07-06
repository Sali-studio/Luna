package commands

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"luna/logger"
	"net/http"
	"os/exec"
	"sync"

	"github.com/bwmarrin/discordgo"
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
	Stop            chan bool // 再生停止を通知するチャンネル
	IsPlaying       bool
	Mutex           sync.Mutex
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

// getOrCreateSession は、サーバーのセッションを取得または新規作成する
func getOrCreateSession(guildID string) *MusicSession {
	musicMutex.Lock()
	defer musicMutex.Unlock()

	if session, ok := musicSessions[guildID]; ok {
		return session
	}

	musicSessions[guildID] = &MusicSession{
		GuildID: guildID,
		Queue:   make([]Song, 0),
		Stop:    make(chan bool),
	}
	return musicSessions[guildID]
}

// --- コマンドハンドラ ---

func (c *MusicCommand) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})

	query := i.ApplicationCommandData().Options[0].Options[0].StringValue()
	session := getOrCreateSession(i.GuildID)

	// ユーザーがVCにいるか確認
	vs, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &[]string{"❌ まずボイスチャンネルに参加してください。"}[0]})
		return
	}

	// Pythonサーバーに問い合わせてStream URLを取得
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
		// BotをVCに接続
		vc, err := s.ChannelVoiceJoin(i.GuildID, vs.ChannelID, false, true)
		if err != nil {
			logger.Error("Failed to join voice channel", "error", err)
			return
		}
		session.VoiceConnection = vc
		go playMusic(s, session) // 再生ループを開始
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

	session.Stop <- true
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "⏩ スキップしました。"}})
}

func (c *MusicCommand) handleStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	session, ok := musicSessions[i.GuildID]
	if !ok {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "❌ BotはVCに参加していません。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	session.Mutex.Lock()
	session.Queue = make([]Song, 0) // キューをクリア
	if session.IsPlaying {
		session.Stop <- true
	}
	session.Mutex.Unlock()

	session.VoiceConnection.Disconnect()

	musicMutex.Lock()
	delete(musicSessions, i.GuildID)
	musicMutex.Unlock()

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
			if i > 9 { // 表示上限
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

// playMusic は、キューから曲を取り出して再生するループ
func playMusic(s *discordgo.Session, session *MusicSession) {
	for {
		session.Mutex.Lock()
		if len(session.Queue) == 0 {
			session.IsPlaying = false
			session.VoiceConnection.Disconnect()
			musicMutex.Lock()
			delete(musicSessions, session.GuildID)
			musicMutex.Unlock()
			session.Mutex.Unlock()
			return // キューが空なら終了
		}

		song := session.Queue[0]
		session.Queue = session.Queue[1:]
		session.NowPlaying = &song
		session.IsPlaying = true
		session.Mutex.Unlock()

		// ffmpegを使って音声ストリームをエンコード
		ffmpeg := exec.Command("ffmpeg", "-i", song.StreamURL, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
		stdout, err := ffmpeg.StdoutPipe()
		if err != nil {
			logger.Error("ffmpeg stdout pipe error", "error", err)
			continue
		}

		if err := ffmpeg.Start(); err != nil {
			logger.Error("ffmpeg start error", "error", err)
			continue
		}

		session.VoiceConnection.Speaking(true)

	streamLoop:
		for {
			select {
			case <-session.Stop:
				break streamLoop
			default:
				opus, err := readOpus(stdout)
				if err != nil {
					if err != io.EOF {
						logger.Error("readOpus error", "error", err)
					}
					break streamLoop
				}
				session.VoiceConnection.OpusSend <- opus
			}
		}

		session.VoiceConnection.Speaking(false)
		ffmpeg.Process.Kill()
	}
}

// readOpus はio.Readerから1フレーム分のOpusデータを読み込む
func readOpus(r io.Reader) ([]byte, error) {
	var opusLen int16
	err := binary.Read(r, binary.LittleEndian, &opusLen)
	if err != nil {
		return nil, err
	}

	opus := make([]byte, opusLen)
	err = binary.Read(r, binary.LittleEndian, &opus)
	return opus, err
}

// --- 未使用のインターフェースメソッド ---
func (c *MusicCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *MusicCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *MusicCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *MusicCommand) GetCategory() string                                                  { return "音楽" }
