package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

var scheduler *cron.Cron

func init() {
	// 秒単位の指定も可能なスケジューラを作成
	scheduler = cron.New(cron.WithSeconds())
	scheduler.Start()

	cmd := &discordgo.ApplicationCommand{
		Name:                     "schedule",
		Description:              "指定した日時にメッセージを送信する予約をします。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageMessages),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "datetime",
				Description: "送信日時 (例: 2025-07-02 15:30:00)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "送信するメッセージの内容",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "送信先のチャンネル (任意、未指定の場合はこのチャンネル)",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Required:     false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("schedule command received")

		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		datetimeStr := optionMap["datetime"].StringValue()
		message := optionMap["message"].StringValue()

		targetChannelID := i.ChannelID // デフォルトはコマンド実行チャンネル
		if opt, ok := optionMap["channel"]; ok {
			targetChannelID = opt.ChannelValue(s).ID
		}

		// ユーザーが入力した日時をパース(解釈)する
		loc, _ := time.LoadLocation("Asia/Tokyo") // タイムゾーンを日本時間に設定
		t, err := time.ParseInLocation("2006-01-02 15:04:05", datetimeStr, loc)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "❌ 日時の形式が正しくありません。(例: `2025-07-02 15:30:00`)", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		// cron形式のスケジュール文字列を生成 (秒 時 分 日 月 曜日)
		cronSpec := fmt.Sprintf("%d %d %d %d %d *", t.Second(), t.Minute(), t.Hour(), t.Day(), t.Month())

		// スケジューラにタスクを追加
		_, err = scheduler.AddFunc(cronSpec, func() {
			s.ChannelMessageSend(targetChannelID, message)
			logger.Info.Printf("Sent scheduled message to channel %s", targetChannelID)
		})

		if err != nil {
			logger.Error.Printf("Failed to add schedule: %v", err)
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ メッセージの送信予約をしました！\n**送信日時:** <t:%d:F>\n**送信先:** <#%s>", t.Unix(), targetChannelID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}
