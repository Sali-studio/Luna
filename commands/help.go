package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.comcom/bwmarrin/discordgo"
)

type HelpCommand struct{}

func (c *HelpCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Botのコマンド一覧を表示します",
	}
}

func (c *HelpCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 本来はmain.goから全コマンドのリストを受け取り動的に生成するのが理想
	commandsList := []struct{ Name, Description string }{
		{"/ping", "Botの応答速度をテストします。"},
		{"/help", "このヘルプメッセージを表示します。"},
		{"/ask", "AIに質問します。"},
		{"/config", "サーバーの各種設定を行います。"},
		{"/ticket-setup", "チケットパネルを設置します。"},
		{"/reaction-role-setup", "リアクションロールを設定します。"},
		// ... 他の主要なコマンドを追加 ...
	}

	var builder strings.Builder
	builder.WriteString("## 🌙 Luna Bot コマンド一覧\n\n")
	for _, cmd := range commandsList {
		builder.WriteString(fmt.Sprintf("**`%s`**\n%s\n\n", cmd.Name, cmd.Description))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "ヘルプ",
		Description: builder.String(),
		Color:       0x7289da,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Luna Bot"},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		logger.Error.Printf("helpコマンドへの応答中にエラー: %v", err)
	}
}

func (c *HelpCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *HelpCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *HelpCommand) GetComponentIDs() []string                                            { return []string{} }
