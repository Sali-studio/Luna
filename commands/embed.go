package commands

import (
	"luna/logger"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

const (
	EmbedModalCustomID = "embed_creator_modal"
	EmbedTitleID       = "embed_title"
	EmbedDescriptionID = "embed_description"
	EmbedColorID       = "embed_color"
)

type EmbedCommand struct{}

func (c *EmbedCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "embed",
		Description: "カスタマイズされた埋め込みメッセージを作成します",
	}
}

// Handle はモーダルを表示する役割を担います
func (c *EmbedCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: EmbedModalCustomID,
			Title:    "Embedを作成",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    EmbedTitleID,
							Label:       "タイトル",
							Style:       discordgo.TextInputShort,
							Placeholder: "Embedのタイトルを入力",
							Required:    true,
							MaxLength:   256,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    EmbedDescriptionID,
							Label:       "説明文",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Embedの本文を入力。Markdownが使えます。",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    EmbedColorID,
							Label:       "色 (16進数カラーコード)",
							Style:       discordgo.TextInputShort,
							Placeholder: "例: 7289da (Discord Blue)",
							Required:    false,
							MinLength:   6,
							MaxLength:   6,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Embedモーダルの表示に失敗: %v", err)
	}
}

// HandleModal はモーダルが送信された後の処理です
func (c *EmbedCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	// どのモーダルに対する応答かを確認
	if data.CustomID != EmbedModalCustomID {
		return
	}

	title := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	description := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	colorStr := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	color := 0x7289da // デフォルトカラー
	if colorStr != "" {
		c, err := strconv.ParseInt(colorStr, 16, 32)
		if err == nil {
			color = int(c)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		logger.Error.Printf("Embedの送信に失敗: %v", err)
	}
}

func (c *EmbedCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
