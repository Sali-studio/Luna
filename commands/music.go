package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca/v2"
)

var (
	musicSessions = make(map[string]*MusicSession)
	musicMutex    sync.Mutex
)

type MusicSession struct {
	GuildID         string
	VoiceConnection *discordgo.VoiceConnection
	Queue           []Song
	NowPlaying      *Song
	IsPlaying       bool
	EncodeSession   *dca.EncodeSession
	Mutex           sync.Mutex
	Log             interfaces.Logger
}

type Song struct {
	Title     string `json:"title"`
	StreamURL string `json:"stream_url"`
	Query     string
}

type MusicCommand struct {
	Log interfaces.Logger
}

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

func (c *MusicCommand) getOrCreateSession(guildID string) *MusicSession {
	musicMutex.Lock()
	defer musicMutex.Unlock()

	if session, ok := musicSessions[guildID]; ok {
		return session
	}

	musicSessions[guildID] = &MusicSession{
		GuildID: guildID,
		Queue:   make([]Song, 0),
		Log:     c.Log,
	}
	return musicSessions[guildID]
}

func (c *MusicCommand) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})

	query := i.ApplicationCommandData().Options[0].Options[0].StringValue()
	session := c.getOrCreateSession(i.GuildID)

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
	}

	session.Mutex.Lock()
	session.Queue = append(session.Queue, song)
	isPlaying := session.IsPlaying
	session.Mutex.Unlock()

	if isPlaying {
		content := fmt.Sprintf("🎵 **%s** をキューに追加しました。", song.Title)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
	} else {
		vc, err := s.ChannelVoiceJoin(i.GuildID, vs.ChannelID, false, true)
		if err != nil {
			c.Log.Error("Failed to join voice channel", "error", err)
			return
		}
		session.VoiceConnection = vc
		go playMusic(session)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &[]string{"▶️ 再生を開始します..."}[0]})
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

	embed := &discordgo.MessageEmbed{Title: "🎵 再生キュー", Color: 0x5865F2}
	session.Mutex.Lock()
	defer session.Mutex.Unlock()

	if session.NowPlaying != nil {
		embed.Description = fmt.Sprintf("**現在再生中:**\n[%s](%s)\n\n", session.NowPlaying.Title, session.NowPlaying.Query)
	}

	if len(session.Queue) > 0 {
		var queueText string
		for i, song := range session.Queue {
			if i > 9 {
				queueText += fmt.Sprintf("\n...他%d曲", len(session.Queue)-10)
				break
			}
			queueText += fmt.Sprintf("**%d.** %s\n", i+1, song.Title)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "再生待ち", Value: queueText})
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

// playMusicは音楽再生のメインループ
func playMusic(session *MusicSession) {
	defer func() {
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
			return
		}
		song := session.Queue[0]
		session.Queue = session.Queue[1:]
		session.NowPlaying = &song
		session.IsPlaying = true
		session.Mutex.Unlock()
		opts := dca.StdEncodeOptions
		opts.RawOutput = true
		opts.Bitrate = 96
		opts.Application = "lowdelay"

		encodingSession, err := dca.EncodeFile(song.StreamURL, opts)
		if err != nil {
			session.Log.Error("エンコードセッションの作成に失敗しました。", "error", err)
			continue
		}
		session.EncodeSession = encodingSession
		defer encodingSession.Cleanup()

		session.VoiceConnection.Speaking(true)

		// 音声フレームを読み出し、送信するループ
		for {
			frame, err := encodingSession.OpusFrame()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					session.Log.Error("Opusフレームの読み取りに失敗しました。", "error", err)
				}
				break
			}

			// Discordのチャンネルに送信
			session.VoiceConnection.OpusSend <- frame
		}

		session.VoiceConnection.Speaking(false)

	}
}

func (c *MusicCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *MusicCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *MusicCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *MusicCommand) GetCategory() string                                                  { return "音楽" }