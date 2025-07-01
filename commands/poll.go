package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PollCommand struct{}

func (c *PollCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "poll",
		Description: "投票を作成します",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "question", Description: "投票の質問内容", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "options", Description: "選択肢をカンマ(,)で区切って入力 (最大10個)", Required: true},
		},
	}
}

func (c *PollCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	question := data.Options[0].StringValue()
	optionsStr := data.Options[1].StringValue()
	options := strings.Split(optionsStr, ",")

	for i, opt := range options {
		options[i] = strings.TrimSpace(opt)
	}

	if len(options) < 2 || len(options) > 10 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ 選択肢は2個以上、10個以下で指定してください。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	var description strings.Builder
	emojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	for i, opt := range options {
		description.WriteString(fmt.Sprintf("%s %s\n\n", emojis[i], opt))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📊 %s", question),
		Description: description.String(),
		Color:       0x40E0D0,
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("%s によって作成されました", i.Member.User.Username)},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	if err != nil {
		logger.Error.Printf("投票の送信に失敗: %v", err)
		return
	}

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		logger.Error.Printf("投票メッセージの取得に失敗: %v", err)
		return
	}

	for i := 0; i < len(options); i++ {
		s.MessageReactionAdd(msg.ChannelID, msg.ID, emojis[i])
	}
}

func (c *PollCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PollCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *PollCommand) GetComponentIDs() []string                                            { return []string{} }
