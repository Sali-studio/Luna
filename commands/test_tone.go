package commands

import (
	"io"
	"luna/logger"
	"os/exec"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "test-tone",
		Description: "ffmpegから直接テスト音を再生し、音声送信機能を診断します",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("test-tone command received")

		vc, ok := VoiceConnections[i.GuildID]
		if !ok {
			// ボットがVCにいなければ、自動で参加させる
			guild, _ := s.State.Guild(i.GuildID)
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
					Data: &discordgo.InteractionResponseData{Content: "先にボイスチャンネルに参加してください。"},
				})
				return
			}
			newVc, err := s.ChannelVoiceJoin(i.GuildID, voiceChannelID, false, true)
			if err != nil {
				logger.Error.Printf("Failed to join voice channel for test-tone: %v", err)
				return
			}
			VoiceConnections[i.GuildID] = newVc
			vc = newVc
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "🔔 テスト音の再生を試みます..."},
		})

		go playTestTone(vc)
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

func playTestTone(vc *discordgo.VoiceConnection) {
	logger.Info.Println("Attempting to play test tone...")

	// ffmpegに10秒間のサイン波を生成させるコマンド
	ffmpeg := exec.Command("ffmpeg", "-f", "lavfi", "-i", "sine=frequency=440:duration=10", "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")

	ffmpegOutput, err := ffmpeg.StdoutPipe()
	if err != nil {
		logger.Error.Printf("ffmpeg.StdoutPipe() for test-tone failed: %v", err)
		return
	}

	if err := ffmpeg.Start(); err != nil {
		logger.Error.Printf("ffmpeg.Start() for test-tone failed: %v", err)
		return
	}

	vc.Speaking(true)
	defer vc.Speaking(false)

	time.Sleep(250 * time.Millisecond)

	logger.Info.Println("Sending test tone data...")
	for {
		opusPacket := make([]byte, 3840)
		_, err := io.ReadFull(ffmpegOutput, opusPacket)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			logger.Info.Println("Test tone finished.")
			break
		}
		if err != nil {
			logger.Error.Printf("Failed to read from ffmpeg for test-tone: %v", err)
			break
		}
		vc.OpusSend <- opusPacket
	}

	if err := ffmpeg.Wait(); err != nil {
		logger.Error.Printf("ffmpeg.Wait() for test-tone failed: %v", err)
	}
}
