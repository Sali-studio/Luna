package commands

import (
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "leave",
		Description: "ボットをボイスチャンネルから退出させます",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("leave command received")

		// マップからこのサーバーの接続情報を取得
		vc, ok := VoiceConnections[i.GuildID]
		if !ok {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ボットはどのボイスチャンネルにも参加していません。",
				},
			})
			return
		}

		// ボイスチャンネルから切断
		err := vc.Disconnect()
		if err != nil {
			logger.Error.Printf("Failed to disconnect from voice channel: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ボイスチャンネルからの退出に失敗しました。",
				},
			})
			return
		}

		// マップから接続情報を削除
		delete(VoiceConnections, i.GuildID)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ボイスチャンネルから退出しました。",
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
