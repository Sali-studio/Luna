package player

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"luna/interfaces"
	"net/http"
	"os"
	"os/exec"
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
	Encoder         *dca.EncodeSession
	Stream          *dca.StreamingSession
	Queue           []*Song
	NowPlaying      *Song // Add NowPlaying field
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
func (p *Player) GetGuildPlayer(guildID string) interface{} {
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
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
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
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if gp.VoiceConnection != nil {
		gp.VoiceConnection.Speaking(false)
		gp.VoiceConnection.Disconnect()
		gp.VoiceConnection = nil
	}
	if gp.Stream != nil {
		gp.Stream.SetPaused(true)
		// gp.Stream.Cleanup() // StreamingSessionにはCleanupがない
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
func (p *Player) Play(guildID string, url, title, author string) error {
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if gp.VoiceConnection == nil {
		return fmt.Errorf("ボイスチャンネルに接続していません。先に接続してください。")
	}

	song := &Song{URL: url, Title: title, Author: author}
	gp.Queue = append(gp.Queue, song)

	if !gp.Playing {
		gp.Playing = true
		go p.playNextSong(guildID)
	}
	return nil
}

// playNextSong はキューから次の曲を再生します。
func (p *Player) playNextSong(guildID string) {
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
	for {
		gp.mu.Lock()
		if len(gp.Queue) == 0 {
			gp.Playing = false
			gp.NowPlaying = nil // Clear NowPlaying
			gp.mu.Unlock()
			break // キューが空になったら再生を停止
		}
		song := gp.Queue[0]
		gp.Queue = gp.Queue[1:] // キューから削除
		gp.NowPlaying = song    // Set NowPlaying
		gp.mu.Unlock()

		p.Log.Info("再生開始", "guildID", guildID, "title", song.Title, "url", song.URL)

		options := dca.StdEncodeOptions
		options.RawOutput = true
		options.Bitrate = 96 // 音質設定
		options.Application = "lowdelay"

		// yt-dlp を使用してオーディオストリームのURLとメタデータを取得
		streamURL, title, author, err := p.GetAudioStreamURL(song.URL)
		if err != nil {
			p.Log.Error("Failed to get audio stream URL from yt-dlp", "error", err, "url", song.URL)
			continue
		}
		p.Log.Info("yt-dlp stream URL", "url", streamURL)
		song.Title = title
		song.Author = author

		// ストリームを一時ファイルにダウンロード
		tempFile, err := ioutil.TempFile("", "audio-*.dca")
		if err != nil {
			p.Log.Error("Failed to create temp file", "error", err)
			continue
		}
		defer os.Remove(tempFile.Name()) // 関数終了時に一時ファイルを削除

		resp, err := http.Get(streamURL)
		if err != nil {
			p.Log.Error("Failed to download audio stream", "error", err, "url", streamURL)
			continue
		}
		defer resp.Body.Close()

		if _, err := io.Copy(tempFile, resp.Body); err != nil {
			p.Log.Error("Failed to write audio to temp file", "error", err)
			continue
		}
		tempFile.Close()

		encodeSession, err := dca.EncodeFile(tempFile.Name(), options)
		if err != nil {
			p.Log.Error("音声のエンコードに失敗しました", "error", err, "file", tempFile.Name())
			continue
		}
		defer encodeSession.Cleanup()

		gp.Encoder = encodeSession
		errChan := make(chan error)

		if gp.VoiceConnection == nil || !gp.VoiceConnection.Ready {
			p.Log.Error("Voice connection is not ready before starting stream", "guildID", guildID)
			continue // Skip to next song if voice connection is not ready
		}

		p.Log.Info("Starting DCA stream", "guildID", guildID)
		gp.Stream = dca.NewStream(encodeSession, gp.VoiceConnection, errChan)
		p.Log.Info("DCA stream started", "guildID", guildID)

		// 再生終了を待つ
		select {
		case <-gp.Quit:
			p.Log.Info("再生停止シグナルを受信しました", "guildID", guildID)
			return // 停止シグナルを受け取ったら終了
		case err := <-errChan:
			if err != nil && err != io.EOF {
				p.Log.Error("ストリームエラー", "error", err, "guildID", guildID)
			}
			p.Log.Info("曲の再生が終了しました", "guildID", guildID, "title", song.Title)
		}
	}
}

// GetAudioStreamURL はyt-dlpを使用してオーディオストリームのURLとメタデータを取得します。
func (p *Player) GetAudioStreamURL(url string) (streamURL, title, author string, err error) {
	cmd := exec.Command("yt-dlp", "-f", "bestaudio[ext=webm]/bestaudio", "--dump-json", url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", "", "", fmt.Errorf("yt-dlpの実行に失敗しました: %w\n%s", err, stderr.String())
	}

	var result struct {
		URL      string `json:"url"`
		Title    string `json:"title"`
		Uploader string `json:"uploader"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return "", "", "", fmt.Errorf("yt-dlpの出力のパースに失敗しました: %w\n%s", err, stdout.String())
	}

	if result.URL == "" {
		return "", "", "", fmt.Errorf("yt-dlpからオーディオストリームのURLを取得できませんでした: %s", stderr.String())
	}

	return result.URL, result.Title, result.Uploader, nil
}

// Stop は現在の再生を停止し、キューをクリアします。
func (p *Player) Stop(guildID string) {
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if gp.Stream != nil {
		gp.Stream.SetPaused(true)
		// gp.Stream.Cleanup() // StreamingSessionにはCleanupがない
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
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	// 現在再生中のストリームを停止
	if gp.Stream != nil {
		gp.Stream.SetPaused(true)
		// gp.Stream.Cleanup() // StreamingSessionにはCleanupがない
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
func (p *Player) GetQueue(guildID string) []struct{ URL, Title, Author string } {
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
	gp.mu.Lock()
	defer gp.mu.Unlock()
	// キューのコピーを返す（外部からの変更を防ぐため）
	queueCopy := make([]struct{ URL, Title, Author string }, len(gp.Queue))
	for i, song := range gp.Queue {
		queueCopy[i] = struct{ URL, Title, Author string }{URL: song.URL, Title: song.Title, Author: song.Author}
	}
	return queueCopy
}

// NowPlaying は現在再生中の曲を取得します。
func (p *Player) NowPlaying(guildID string) *struct{ URL, Title, Author string } {
	gp := p.GetGuildPlayer(guildID).(*GuildPlayer)
	gp.mu.Lock()
	defer gp.mu.Unlock()

	if gp.NowPlaying != nil && gp.Playing {
		return &struct{ URL, Title, Author string }{URL: gp.NowPlaying.URL, Title: gp.NowPlaying.Title, Author: gp.NowPlaying.Author}
	}
	return nil
}
