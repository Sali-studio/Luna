package commands

import (
	"bytes"
	"fmt"
	"io"
	"luna/logger"
	"os/exec"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "play",
		Description: "指定されたYouTubeのURLを再生します（VCにいない場合は自動で参加します）",
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

		vc, ok := VoiceConnections[i.GuildID]
		if !ok {
			logger.Info.Println("Bot is not in a voice channel. Attempting to join.")
			guild, err := s.State.Guild(i.GuildID)
			if err != nil {
				logger.Error.Printf("Failed to get guild: %v", err)
				return
			}

			var voiceChannelID string
			for _, vs := range guild.VoiceStates {
				if vs.UserID == i.Member.User.ID {
					voiceChannelID = vs.ChannelID
					break
				}
			}

			if voiceChannelID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "ボイスチャンネルに参加してからコマンドを実行してください。",
					},
				})
				return
			}

			newVc, err := s.ChannelVoiceJoin(i.GuildID, voiceChannelID, false, true)
			if err != nil {
				logger.Error.Printf("Failed to join voice channel: %v", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "ボイスチャンネルへの接続に失敗しました。",
					},
				})
				return
			}
			VoiceConnections[i.GuildID] = newVc
			vc = newVc
			logger.Info.Printf("Joined voice channel: %s", voiceChannelID)
		}

		url := i.ApplicationCommandData().Options[0].StringValue()

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "🎵 再生準備中です..."},
		})

		go playYoutube(s, i, vc, url)
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

func playYoutube(s *discordgo.Session, i *discordgo.InteractionCreate, vc *discordgo.VoiceConnection, url string) {
	var stderrBuf bytes.Buffer

	sendError := func(msg string, err error) {
		logger.Error.Printf("%s: %v\n--- Stderr ---\n%s", msg, err, stderrBuf.String())
		content := fmt.Sprintf("❌ 再生に失敗しました: %s", msg)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	}

	// ★★★ ここからコマンド実行方法を全面的に変更 ★★★
	// 実行するコマンド全体を一つの文字列として作成
	// "URL" のように、URLをダブルクォートで囲むのが安定動作のコツ
	commandString := fmt.Sprintf(`yt-dlp --no-playlist --quiet --no-warnings -f bestaudio -o - "%s" | ffmpeg -i pipe:0 -f s16le -ar 48000 -ac 2 -loglevel error pipe:1`, url)

	// cmd.exe を経由してコマンドを実行
	cmd := exec.Command("cmd", "/C", commandString)
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendError("コマンドパイプ作成エラー", err)
		return
	}

	if err := cmd.Start(); err != nil {
		sendError("コマンドの起動エラー", err)
		return
	}

	content := fmt.Sprintf("🎶 再生を開始します: `%s`", url)
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	vc.Speaking(true)
	defer vc.Speaking(false)

	time.Sleep(250 * time.Millisecond)

	for {
		opusPacket := make([]byte, 3840)
		_, err := io.ReadFull(stdout, opusPacket)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			logger.Info.Println("再生が終了しました。")
			break
		}
		if err != nil {
			sendError("音声データの読み込みエラー", err)
			break
		}
		vc.OpusSend <- opusPacket
	}

	if err := cmd.Wait(); err != nil {
		sendError("コマンド実行時エラー", err)
	}
}
