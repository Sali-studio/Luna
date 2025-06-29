package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

// init関数はパッケージがインポートされたときに自動的に実行されます
func init() {
	// コマンドの定義
	cmd := &discordgo.ApplicationCommand{
		Name:        "avatar",
		Description: "ユーザーのアバターまたはサーバープロフィールアイコンを表示します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "アバターを表示するユーザー",
				Required:    false, // falseにすると、このオプションは任意になります
			},
		},
	}

	// コマンドが実行されたときの処理
	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("avatar command received")

		var targetUser *discordgo.User
		options := i.ApplicationCommandData().Options

		// "user"オプションが指定されているかチェック
		if len(options) > 0 {
			targetUser = options[0].UserValue(s)
		} else {
			// 指定されていない場合は、コマンドを実行したユーザー自身を対象とする
			targetUser = i.Member.User
		}

		// サーバー固有のアバターを優先して取得するため、Memberオブジェクトを取得
		// (discordgoのMember.AvatarURLは、サーバー固有のアバターがない場合、自動で通常のアバターを返してくれます)
		avatarURL := i.Member.User.AvatarURL("1024") // "1024"は画像のサイズ
		if len(options) > 0 {
			// 他のユーザーが指定された場合は、そのユーザーのMember情報を取得する
			member, err := s.GuildMember(i.GuildID, targetUser.ID)
			if err != nil {
				logger.Error.Printf("Failed to get member info: %v", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "ユーザー情報の取得に失敗しました。"},
				})
				return
			}
			avatarURL = member.AvatarURL("1024")
		}

		// Embed（埋め込み）メッセージを作成
		embed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("%s のアバター", targetUser.Username),
			Color: 0x0099ff, // 埋め込みの左側の線の色
			Image: &discordgo.MessageEmbedImage{
				URL: avatarURL,
			},
		}

		// 作成したEmbedで応答
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
		if err != nil {
			logger.Error.Printf("Error responding to avatar command: %v", err)
		}
	}

	// グローバルな変数にコマンド定義とハンドラを追加
	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
