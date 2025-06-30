package commands

import (
	"luna/logger"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:        "embed",
		Description: "Embed作成ビルダーを起動します。",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("embed command received, showing modal")

		// Embed作成用のモーダルを表示する
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "embed_creation_modal",
				Title:    "Embed 作成ビルダー",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "title",
								Label:       "タイトル",
								Style:       discordgo.TextInputShort,
								Placeholder: "Embedのタイトルを入力",
								Required:    true,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "description",
								Label:       "説明文 (メインのテキスト)",
								Style:       discordgo.TextInputParagraph,
								Placeholder: "ここにメインの文章を入力します。\n改行も可能です。",
								Required:    true,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "color",
								Label:       "色 (16進数カラーコード)",
								Style:       discordgo.TextInputShort,
								Placeholder: "例: #3498DB (未入力の場合はDiscordのデフォルト色)",
								Required:    false,
								MaxLength:   7,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "footer",
								Label:       "フッター (一番下のテキスト)",
								Style:       discordgo.TextInputShort,
								Placeholder: "例: サーバーからのお知らせ",
								Required:    false,
							},
						},
					},
				},
			},
		})
		if err != nil {
			logger.Error.Printf("Failed to show embed modal: %v", err)
		}
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// HandleEmbedCreation はモーダルから送信されたデータに基づいてEmbedを作成します
func HandleEmbedCreation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	title := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	description := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	colorStr := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	footer := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Timestamp:   time.Now().Format(time.RFC3339),
		Author: &discordgo.MessageEmbedAuthor{
			Name:    i.Member.User.Username,
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	if footer != "" {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: footer,
		}
	}

	if colorStr != "" {
		colorStr = strings.TrimPrefix(colorStr, "#")
		colorInt, err := strconv.ParseInt(colorStr, 16, 32)
		if err == nil {
			embed.Color = int(colorInt)
		}
	}

	// 作成したEmbedを送信
	s.ChannelMessageSendEmbed(i.ChannelID, embed)

	// モーダルへの応答（Ephemeralで本人にだけ見える）
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Embedを作成しました。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
