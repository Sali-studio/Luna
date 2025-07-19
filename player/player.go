package player

import (
	"fmt"
	"io"
	"luna/interfaces"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

// Song は再生する曲の情報を保持します。
type Song struct {
	URL    string
	Title  string
	Author string
}

// GuildPlayer は各サーバー（Guild）ごとのプレイヤーの状態を保持します。
type GuildPlayer struct {
	VoiceConnection *discordgo.VoiceConnection
	Encoder         *dca.Encoder
	Stream          *dca.StreamingSession
	Queue           []*Song
	Playing         bool
	Quit            chan bool
	mu              sync.Mutex
}

// Player はすべてのサーバーのプレイヤーを管理します。
type Player struct {
	Session *discordgo.Session
	Log     interfaces.Logger
	Store   interfaces.DataStore
	Guilds  map[string]*GuildPlayer
	mu      sync.Mutex
}

// NewPlayer は新しいPlayerインスタンスを作成します。
func NewPlayer(s *discordgo.Session, log interfaces.Logger, store interfaces.DataStore) *Player {
	return &Player{
		Session: s,
		Log:     log,
		Store:   store,
		Guilds:  make(map[string]*GuildPlayer),
	}
}

// GetGuildPlayer は指定されたサーバーのGuildPlayerを取得します。存在しない場合は作成します。
func (p *Player) GetGuildPlayer(guildID string) *GuildPlayer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if gp, ok := p.Guilds[guildID]; ok {
		return gp
	}
	gp := &GuildPlayer{
		Queue: make([]*Song, 0),
		Quit:  make(chan bool),
	}
	p.Guilds[guildID] = gp
	return gp
}

// JoinVC はボイスチャンネルに接続します。
func (p *Player) JoinVC(guildID, channelID string) error {
	gp := p.GetGuildPlayer(guildID)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	var err error
	gp.VoiceConnection, err = p.Session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return fmt.Errorf("ボイスチャンネルへの接続に失敗しました: %w", err)
	}

	gp.VoiceConnection.Speaking(true)
	return nil
}

// LeaveVC はボイスチャンネルから切断します。
func (p *Player) LeaveVC(guildID string) {
	gp := p.GetGuildPlayer(guildID)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if gp.VoiceConnection != nil {
		gp.VoiceConnection.Speaking(false)
		gp.VoiceConnection.Disconnect()
		gp.VoiceConnection = nil
	}
	if gp.Stream != nil {
		gp.Stream.SetPaused(true)
		gp.Stream.Cleanup()
		gp.Stream = nil
	}
	if gp.Encoder != nil {
		gp.Encoder.Cleanup()
		gp.Encoder = nil
	}
	gp.Playing = false
	// キューをクリア
	gp.Queue = make([]*Song, 0)
	// 再生中の場合は停止シグナルを送る
	select {
	case gp.Quit <- true:
	default:
	}
}

// Play はURLから音声を再生します。
func (p *Player) Play(guildID string, song *Song) error {
	gp := p.GetGuildPlayer(guildID)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if gp.VoiceConnection == nil {
		return fmt.Errorf("ボイスチャンネルに接続していません。先に接続してください。")
	}

	gp.Queue = append(gp.Queue, song)

	if !gp.Playing {
		gp.Playing = true
		go p.playNextSong(guildID)
	}
	return nil
}

// playNextSong はキューから次の曲を再生します。
func (p *Player) playNextSong(guildID string) {
	gp := p.GetGuildPlayer(guildID)
	for {
		gp.mu.Lock()
		if len(gp.Queue) == 0 {
			gp.Playing = false
			gp.mu.Unlock()
			break // キューが空になったら再生を停止
		}
		song := gp.Queue[0]
		gp.Queue = gp.Queue[1:] // キューから削除
		gp.mu.Unlock()

		p.Log.Info("再生開始", "guildID", guildID, "title", song.Title, "url", song.URL)

		options := dca.StdEncodeOptions
		options.RawOutput = true
		options.Bitrate = 96 // 音質設定
		options.Application = "lowdelay"

		// YouTubeからのストリームを取得
		// TODO: ここにYouTubeダウンロードロジックを実装
		// 現状はダミーとして、dcaのサンプルにあるような直接URLを渡す形
		// 実際にはyt-dlpなどを利用してオーディオストリームを取得する必要がある
		encodeSession, err := dca.EncodeFile(song.URL, options)
		if err != nil {
			p.Log.Error("音声のエンコードに失敗しました", "error", err, "url", song.URL)
			continue
		}
		defer encodeSession.Cleanup()

		gp.Encoder = encodeSession
		gp.Stream = dca.NewStream(encodeSession)

		// 音声データをDiscordに送信
		for {
			select {
			case <-gp.Quit:
				p.Log.Info("再生停止シグナルを受信しました", "guildID", guildID)
				return // 停止シグナルを受け取ったら終了
			case err := <-gp.Stream.Error():
				if err != nil && err != io.EOF {
					p.Log.Error("ストリームエラー", "error", err, "guildID", guildID)
				}
				return // エラーまたはEOFで終了
			case <-gp.Stream.Done():
				p.Log.Info("曲の再生が終了しました", "guildID", guildID, "title", song.Title)
				return // 曲の終了で終了
			case pcm := <-gp.Stream.PCM:
				if gp.VoiceConnection != nil && gp.VoiceConnection.Ready {
					gp.VoiceConnection.OpusSend <- pcm
				}
			}
		}
	}
}

// Stop は現在の再生を停止し、キューをクリアします。
func (p *Player) Stop(guildID string) {
	gp := p.GetGuildPlayer(guildID)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if gp.Stream != nil {
		gp.Stream.SetPaused(true)
		gp.Stream.Cleanup()
		gp.Stream = nil
	}
	if gp.Encoder != nil {
		gp.Encoder.Cleanup()
		gp.Encoder = nil
	}
	gp.Playing = false
	gp.Queue = make([]*Song, 0) // キューをクリア
	select {
	case gp.Quit <- true:
	default:
	}
}

// Skip は現在の曲をスキップし、次の曲を再生します。
func (p *Player) Skip(guildID string) {
	gp := p.GetGuildPlayer(guildID)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	// 現在再生中のストリームを停止
	if gp.Stream != nil {
		gp.Stream.SetPaused(true)
		gp.Stream.Cleanup()
		gp.Stream = nil
	}
	if gp.Encoder != nil {
		gp.Encoder.Cleanup()
		gp.Encoder = nil
	}
	// 再生中の場合は停止シグナルを送る
	select {
	case gp.Quit <- true:
	default:
	}

	// 次の曲を再生
	if len(gp.Queue) > 0 {
		go p.playNextSong(guildID)
	} else {
		gp.Playing = false
	}
}

// GetQueue は現在の再生キューを取得します。
func (p *Player) GetQueue(guildID string) []*Song {
	gp := p.GetGuildPlayer(guildID)
	gp.mu.Lock()
	defer gp.mu.Unlock()
	// キューのコピーを返す（外部からの変更を防ぐため）
	queueCopy := make([]*Song, len(gp.Queue))
	copy(queueCopy, gp.Queue)
	return queueCopy
}

// NowPlaying は現在再生中の曲を取得します。
func (p *Player) NowPlaying(guildID string) *Song {
	gp := p.GetGuildPlayer(guildID)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if len(gp.Queue) > 0 && gp.Playing {
		// キューの先頭が現在再生中の曲とみなす
		return gp.Queue[0]
	}
	return nil
}
