// commands/describe_image_context.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"luna/interfaces"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type DescribeImageContextCommand struct {
	Log interfaces.Logger
}

func (c *DescribeImageContextCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name: "この画像を説明して", // メッセージコンテキストメニューに表示される名前
		Type: discordgo.MessageApplicationCommand,
	}
}

func (c *DescribeImageContextCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	targetMessage := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID]

	var imageURL string
	if len(targetMessage.Attachments) > 0 && len(targetMessage.Attachments[0].ContentType) > 5 && targetMessage.Attachments[0].ContentType[0:5] == "image" {
		imageURL = targetMessage.Attachments[0].URL
	} else if len(targetMessage.Embeds) > 0 && targetMessage.Embeds[0].Image != nil {
		imageURL = targetMessage.Embeds[0].Image.URL
	} else {
		content := "エラー: 対象のメッセージに画像が見つかりませんでした。"
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			c.Log.Error("Failed to send error response", "error", err)
		}
		return
	}

	// 共通関数を呼び出す
	SendDescribeRequest(s, i, imageURL, c.Log)
}

func (c *DescribeImageContextCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *DescribeImageContextCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *DescribeImageContextCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *DescribeImageContextCommand) GetCategory() string                                                  { return "AI" }
