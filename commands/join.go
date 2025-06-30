package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "join",
		Description: "ボットをボイスチャンネルに参加させます",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("join command received")

		// コマンドが実行されたサーバー(Guild)を取得
		guild, err := s.State.Guild(i.GuildID)
		if err != nil {
			logger.Error.Printf("Failed to get guild: %v", err)
			return
		}

		// コマンドを実行したユーザーがどのボイスチャンネルにいるか探す
		var voiceChannelID string
		for _, vs := range guild.VoiceStates {
			if vs.UserID == i.Member.User.ID {
				voiceChannelID = vs.ChannelID
				break
			}
		}

		// ユーザーがボイスチャンネルにいなかった場合
		if voiceChannelID == "" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ボイスチャンネルに参加してからコマンドを実行してください。",
				},
			})
			return
		}

		// ボイスチャンネルに接続
		vc, err := s.ChannelVoiceJoin(i.GuildID, voiceChannelID, false, true)
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

		// 接続情報をマップに保存
		VoiceConnections[i.GuildID] = vc

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<#%s> に参加しました。", voiceChannelID),
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
