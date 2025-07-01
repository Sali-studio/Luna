package commands

import (
	"fmt"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type ScheduleCommand struct {
	Scheduler *cron.Cron
	Store     *storage.ConfigStore // 将来の永続化のためにStoreも持たせておく
}

func (c *ScheduleCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "schedule",
		Description: "指定した時間にメッセージを投稿予約します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "cron",
				Description: "時間をCron形式で指定 (分 時 日 月 曜日)",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "投稿先のチャンネル",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Required:     true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "投稿するメッセージ内容",
				Required:    true,
			},
		},
	}
}

func (c *ScheduleCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	cronSpec := options[0].StringValue()
	channel := options[1].ChannelValue(s)
	message := options[2].StringValue()

	// cron式が正しいかパースしてみる
	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := specParser.Parse(cronSpec)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ 無効なCron形式です: `%s`\nエラー: `%v`", cronSpec, err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// スケジューラにタスクを追加
	entryID, err := c.Scheduler.AddFunc(cronSpec, func() {
		_, err := s.ChannelMessageSend(channel.ID, message)
		if err != nil {
			logger.Error.Printf("予約メッセージの送信に失敗 (Channel: %s): %v", channel.ID, err)
		}
	})
	if err != nil {
		// ...エラー処理...
		return
	}

	logger.Info.Printf("新しいタスクをスケジュールしました (ID: %d, Spec: '%s')", entryID, cronSpec)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ メッセージを予約しました。\n- **時間:** `%s`\n- **チャンネル:** <#%s>", cronSpec, channel.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *ScheduleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ScheduleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
