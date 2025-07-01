package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type ScheduleCommand struct {
	Scheduler *cron.Cron
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

	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := specParser.Parse(cronSpec); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ 無効なCron形式です: `%v`", err), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	entryID, err := c.Scheduler.AddFunc(cronSpec, func() {
		if _, err := s.ChannelMessageSend(channel.ID, message); err != nil {
			logger.Error.Printf("予約メッセージの送信に失敗 (Channel: %s): %v", channel.ID, err)
		}
	})
	if err != nil {
		logger.Error.Printf("スケジューラへのタスク追加に失敗: %v", err)
		return
	}

	logger.Info.Printf("新しいタスクをスケジュールしました (ID: %d, Spec: '%s')", entryID, cronSpec)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("✅ メッセージを予約しました。\n- **時間:** `%s`\n- **チャンネル:** <#%s>", cronSpec, channel.ID), Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *ScheduleCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ScheduleCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *ScheduleCommand) GetComponentIDs() []string                                            { return []string{} }
