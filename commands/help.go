package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct {
	// 将来的にコマンドリストを動的に生成する場合、ここに全コマンドの定義を持つマップを渡すことができます
}

func (c *HelpCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Botのコマンド一覧を表示します",
	}
}

func (c *HelpCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// NOTE: 本来は main.go から全コマンドのリストを受け取り、
	//       それを基にヘルプメッセージを動的に生成するのが理想的です。
	//       ここでは簡単化のため、手動でリストを作成します。
	commandsList := []struct {
		Name        string
		Description string
	}{
		{"/ping", "Botの応答速度をテストします。"},
		{"/help", "このヘルプメッセージを表示します。"},
		{"/user-info", "ユーザーの情報を表示します。"},
		{"/weather", "指定した都市の天気を表示します。"},
		{"/calc", "数式を計算します。"},
		{"/poll", "投票を作成します。"},
		{"/embed", "埋め込みメッセージを作成します。"},
		{"/translate", "テキストを翻訳します。"},
		{"/schedule", "メッセージを予約投稿します。"},
		{"/ticket-setup", "チケットパネルを設置します。"},
		{"/reaction-role-setup", "リアクションロールを設定します。"},
		{"/config", "サーバー固有の設定を行います。"},
	}

	var builder strings.Builder
	builder.WriteString("## 🌙 Luna Bot コマンド一覧\n\n")
	for _, cmd := range commandsList {
		builder.WriteString(
			fmt.Sprintf("**`%s`**\n%s\n\n", cmd.Name, cmd.Description),
		)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "ヘルプ",
		Description: builder.String(),
		Color:       0x7289da, // Discord Blue
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Luna Bot",
		},
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
