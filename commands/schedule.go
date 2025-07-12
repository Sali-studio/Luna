package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type ScheduleCommand struct {
	Scheduler interfaces.Scheduler
	Store     interfaces.DataStore
	Log       interfaces.Logger
}

func (c *ScheduleCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "schedule",
		Description:              "指定した時間にメッセージを投稿予約します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageMessages),
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "cron", Description: "時間をCron形式で指定 (分 時 日 月 曜日)", Required: true},
			{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "投稿先のチャンネル", ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText}, Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "投稿するメッセージ内容", Required: true},
		},
	}
}

func (c *ScheduleCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	cronSpec := options[0].StringValue()
	channel := options[1].ChannelValue(s)
	message := options[2].StringValue()

	if _, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(cronSpec); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ 無効なCron形式です: `%v`", err), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	schedule := storage.Schedule{
		GuildID:   i.GuildID,
		ChannelID: channel.ID,
		CronSpec:  cronSpec,
		Message:   message,
	}
	if err := c.Store.SaveSchedule(schedule); err != nil {
		c.Log.Error("スケジュールのDB保存に失敗", "error", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "スケジュールの保存に失敗しました。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}

	c.Scheduler.AddFunc(cronSpec, func() {
		if _, err := s.ChannelMessageSend(channel.ID, message); err != nil {
			c.Log.Error("予約メッセージの送信に失敗", "error", err, "channelID", channel.ID)
		}
	})

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("✅ メッセージを予約しました。\n- **時間:** `%s`\n- **チャンネル:** <#%s>", cronSpec, channel.ID), Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		c.Log.Error("Failed to send response", "error", err)
	}
}

func (c *ScheduleCommand) LoadAndRegisterSchedules(s *discordgo.Session) {
	schedules, err := c.Store.GetAllSchedules()
	if err != nil {
		c.Log.Error("DBからのスケジュール読み込みに失敗", "error", err)
		return
	}
	for _, sc := range schedules {
		currentSchedule := sc
		c.Scheduler.AddFunc(currentSchedule.CronSpec, func() {
			s.ChannelMessageSend(currentSchedule.ChannelID, currentSchedule.Message)
		})
	}
	c.Log.Info("DBからスケジュールを登録しました", "count", len(schedules))
}

func (c *ScheduleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ScheduleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ScheduleCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *ScheduleCommand) GetCategory() string {
	return "管理"
}
