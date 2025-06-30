package commands

import (
	"luna/logger"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func init() {
	// Embed作成コマンドの定義
	cmd := &discordgo.ApplicationCommand{
		Name:        "embed",
		Description: "指定した内容でEmbedメッセージを作成します",
		Options: []*discordgo.ApplicationCommandOption{
			// 各オプションを定義。すべて任意項目(Required: false)とする
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "title",
				Description: "Embedのタイトル",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "description",
				Description: "Embedの説明文",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "color",
				Description: "左側の線の色 (#RRGGBB形式)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "author",
				Description: "作成者名",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "footer",
				Description: "フッターに表示するテキスト",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "image_url",
				Description: "メイン画像のURL",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "thumbnail_url",
				Description: "サムネイル画像のURL",
				Required:    false,
			},
		},
	}

	// コマンドのハンドラ
	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("embed command received")

		// オプションをマップに変換して使いやすくする
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
		for _, opt := range i.ApplicationCommandData().Options {
			optionMap[opt.Name] = opt
		}

		// 空のEmbedを作成
		embed := &discordgo.MessageEmbed{}

		// 各オプションが指定されていれば、Embedの対応するフィールドに設定
		if opt, ok := optionMap["title"]; ok {
			embed.Title = opt.StringValue()
		}
		if opt, ok := optionMap["description"]; ok {
			embed.Description = opt.StringValue()
		}
		if opt, ok := optionMap["author"]; ok {
			embed.Author = &discordgo.MessageEmbedAuthor{Name: opt.StringValue()}
		}
		if opt, ok := optionMap["footer"]; ok {
			embed.Footer = &discordgo.MessageEmbedFooter{Text: opt.StringValue()}
		}
		if opt, ok := optionMap["image_url"]; ok {
			embed.Image = &discordgo.MessageEmbedImage{URL: opt.StringValue()}
		}
		if opt, ok := optionMap["thumbnail_url"]; ok {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: opt.StringValue()}
		}
		if opt, ok := optionMap["color"]; ok {
			// カラーコードの文字列(#RRGGBB)を数値に変換
			colorStr := strings.TrimPrefix(opt.StringValue(), "#")
			colorInt, err := strconv.ParseInt(colorStr, 16, 32)
			if err == nil {
				embed.Color = int(colorInt)
			} else {
				logger.Warning.Printf("Invalid color code provided: %s", opt.StringValue())
			}
		}

		// Embedを送信して応答
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
		if err != nil {
			logger.Error.Printf("Error responding to embed command: %v", err)
		}
	}

	// コマンドとハンドラを登録
	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
