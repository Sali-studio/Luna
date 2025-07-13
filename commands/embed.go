package commands

import (
	"luna/interfaces"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

const EmbedModalCustomID = "embed_modal"

type EmbedCommand struct {
	Log interfaces.Logger
}

func (c *EmbedCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "embed",
		Description:              "カスタマイズされた埋め込みメッセージを作成します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageMessages),
	}
}

func (c *EmbedCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: EmbedModalCustomID,
			Title:    "Embedを作成",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "title", Label: "タイトル", Style: discordgo.TextInputShort, Placeholder: "Embedのタイトルを入力", Required: false, MaxLength: 256},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "description", Label: "説明文", Style: discordgo.TextInputParagraph, Placeholder: "Embedの本文を入力。Markdownが使えます。", Required: true},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "color", Label: "色 (16進数カラーコード)", Style: discordgo.TextInputShort, Placeholder: "例: 7289da (Discord Blue)", Required: false, MinLength: 6, MaxLength: 6},
				}},
			},
		},
	})
	if err != nil {
		c.Log.Error("Embedモーダルの表示に失敗", "error", err)
	}
}

func (c *EmbedCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	title := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	description := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	colorStr := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	color := 0x7289da // デフォルトカラー
	if colorStr != "" {
		if c, err := strconv.ParseInt(colorStr, 16, 32); err == nil {
			color = int(c)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Author:      &discordgo.MessageEmbedAuthor{Name: i.Member.User.Username, IconURL: i.Member.User.AvatarURL("")},
	}

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	}); err != nil {
		c.Log.Error("Failed to send embed response", "error", err)
	}
}

func (c *EmbedCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *EmbedCommand) GetComponentIDs() []string                                            { return []string{EmbedModalCustomID} }
func (c *EmbedCommand) GetCategory() string {
	return "ユーティリティ"
}
