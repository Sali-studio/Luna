package commands

import (
	"fmt"
	"luna/storage"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type BumpCommand struct {
	Store     *storage.ConfigStore
	Scheduler *cron.Cron
}

// (GetCommandDef, Handle, ... を実装)
// このコマンドは /config bump-setup のサブコマンドとして実装するのがより綺麗です。
// ここでは独立したコマンドとしてリファクタリングします。

func (c *BumpCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{Name: "bump-reminder", Description: "BUMPリマインダーを設定します"}
}

func (c *BumpCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := c.Store.GetGuildConfig(i.GuildID)
	if !config.Bump.Reminder || config.Bump.ChannelID == "" || config.Bump.RoleID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "BUMPリマインダーが設定されていません。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// 2時間後にリマインドするタスクを登録
	// 注意: この方法はBot再起動でタスクが消えます
	time.AfterFunc(2*time.Hour, func() {
		s.ChannelMessageSend(config.Bump.ChannelID, fmt.Sprintf("<@&%s> BUMPの時間です！ `/bump`", config.Bump.RoleID))
	})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "BUMPを確認しました。2時間後にリマインドします。", Flags: discordgo.MessageFlagsEphemeral},
	})
}
func (c *BumpCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *BumpCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *BumpCommand) GetComponentIDs() []string                                            { return []string{} }
