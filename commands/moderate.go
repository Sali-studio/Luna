package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "moderate",
		Description: "ユーザーに対する管理操作を行います。",
		Options: []*discordgo.ApplicationCommandOption{
			// Kickサブコマンド
			{
				Name:        "kick",
				Description: "ユーザーをサーバーから追放します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "追放するユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "追放する理由", Required: false},
				},
			},
			// Banサブコマンド
			{
				Name:        "ban",
				Description: "ユーザーをサーバーからBANします。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "BANするユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "BANする理由", Required: false},
				},
			},
			// Timeoutサブコマンド
			{
				Name:        "timeout",
				Description: "ユーザーをタイムアウトさせます。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "タイムアウトさせるユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "duration", Description: "期間 (例: 5m, 1h, 3d)", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "タイムアウトさせる理由", Required: false},
				},
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// 実行されたサブコマンドによって処理を分岐
		switch i.ApplicationCommandData().Options[0].Name {
		case "kick":
			handleShowKickModal(s, i)
		case "ban":
			handleShowBanModal(s, i)
		case "timeout":
			handleShowTimeoutModal(s, i)
		}
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// 確認モーダルを表示する関数群
func handleShowKickModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ユーザーIDなどの情報をモーダルのIDに埋め込むことで、後の処理で使えるようにする
	userID := i.ApplicationCommandData().Options[0].Options[0].UserValue(s).ID
	reason := ""
	if len(i.ApplicationCommandData().Options[0].Options) > 1 {
		reason = i.ApplicationCommandData().Options[0].Options[1].StringValue()
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("moderate_kick_confirm:%s", userID),
			Title:    "Kick実行確認",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "reason",
							Label:       "理由（変更可能）",
							Style:       discordgo.TextInputParagraph,
							Value:       reason,
							Placeholder: "ユーザーをサーバーから追放する理由を入力してください。",
							Required:    true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show kick modal: %v", err)
	}
}

// (BanとTimeoutのモーダル表示関数も同様に作成)
func handleShowBanModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.ApplicationCommandData().Options[0].Options[0].UserValue(s).ID
	reason := ""
	if len(i.ApplicationCommandData().Options[0].Options) > 1 {
		reason = i.ApplicationCommandData().Options[0].Options[1].StringValue()
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("moderate_ban_confirm:%s", userID),
			Title:    "BAN実行確認",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "reason",
							Label:       "理由（変更可能）",
							Style:       discordgo.TextInputParagraph,
							Value:       reason,
							Placeholder: "ユーザーをサーバーからBANする理由を入力してください。",
							Required:    true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show ban modal: %v", err)
	}
}

func handleShowTimeoutModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.ApplicationCommandData().Options[0].Options[0].UserValue(s).ID
	duration := i.ApplicationCommandData().Options[0].Options[1].StringValue()
	reason := ""
	if len(i.ApplicationCommandData().Options[0].Options) > 2 {
		reason = i.ApplicationCommandData().Options[0].Options[2].StringValue()
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("moderate_timeout_confirm:%s:%s", userID, duration),
			Title:    "Timeout実行確認",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "reason",
							Label:       "理由（変更可能）",
							Style:       discordgo.TextInputParagraph,
							Value:       reason,
							Placeholder: "ユーザーをタイムアウトさせる理由を入力してください。",
							Required:    true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to show timeout modal: %v", err)
	}
}
