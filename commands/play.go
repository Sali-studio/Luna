package commands

import (
	"io"
	"luna/logger"
	"os/exec"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "play",
		Description: "指定されたYouTubeのURLを再生します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "url",
				Description: "再生するYouTube動画のURL",
				Required:    true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("play command received")

		// ボットがVCに参加しているか確認
		vc, ok := VoiceConnections[i.GuildID]
		if !ok {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "先に/joinコマンドでボットをVCに参加させてください。"},
			})
			return
		}

		// URLを取得
		url := i.ApplicationCommandData().Options[0].StringValue()

		// まずはコマンドを受け付けたことをユーザーに知らせる
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "🎵 再生準備中です..."},
		})

		// playYoutube関数をゴルーチンで非同期に実行
		go playYoutube(vc, url)
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// playYoutube は指定されたURLの音声を再生する関数
func playYoutube(vc *discordgo.VoiceConnection, url string) {
	// yt-dlp と ffmpeg をパイプで繋いで実行するコマンドを設定
	ytdlp := exec.Command("yt-dlp", "-f", "bestaudio", "-o", "-", url)
	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")

	// yt-dlpの標準出力をffmpegの標準入力に接続
	r, w := io.Pipe()
	ytdlp.Stdout = w
	ffmpeg.Stdin = r

	// ffmpegの標準出力を取得
	stdout, err := ffmpeg.StdoutPipe()
	if err != nil {
		logger.Error.Printf("ffmpeg.StdoutPipe() error: %v", err)
		return
	}

	// コマンドを開始
	err = ytdlp.Start()
	if err != nil {
		logger.Error.Printf("ytdlp.Start() error: %v", err)
		return
	}
	err = ffmpeg.Start()
	if err != nil {
		logger.Error.Printf("ffmpeg.Start() error: %v", err)
		return
	}

	// VCのスピーカーをオンにする
	vc.Speaking(true)
	defer vc.Speaking(false) // 関数終了時にオフにする

	// Opusパケットを送信するためのループ
	for {
		// Opusの1フレームは20ms = 960サンプル * 2ch * 2byte = 3840 byte
		opusPacket := make([]byte, 3840)

		// stdoutからopusPacketに直接読み込む
		_, err := io.ReadFull(stdout, opusPacket)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			logger.Info.Println("再生が終了しました。")
			break
		}
		if err != nil {
			logger.Error.Printf("io.ReadFull() error: %v", err)
			break
		}

		// VCのOpus送信チャネルにデータを送る
		vc.OpusSend <- opusPacket
	}

	// プロセスを終了
	ytdlp.Process.Kill()
	ffmpeg.Process.Kill()
}
