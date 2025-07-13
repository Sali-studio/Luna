package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

type WelcomeCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *WelcomeCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "welcome-setup",
		Description:              "新メンバー参加時のウェルカムメッセージを設定します",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageGuild),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "enable",
				Description: "ウェルカムメッセージ機能を有効にするか",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "メッセージを送信するチャンネル",
				Required:     true,
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "送信するメッセージ。{user}でメンション、{server}でサーバー名に置換されます。",
				Required:    true,
			},
		},
	}
}

func (c *WelcomeCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	enable := options[0].BoolValue()
	channel := options[1].ChannelValue(s)
	message := options[2].StringValue()

	config := storage.WelcomeConfig{
		Enabled:   enable,
		ChannelID: channel.ID,
		Message:   message,
	}

	if err := c.Store.SaveConfig(i.GuildID, "welcome_config", config); err != nil {
		c.Log.Error("ウェルカムメッセージ設定の保存に失敗", "error", err, "guildID", i.GuildID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ 設定の保存に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	var response string
	if enable {
		response = fmt.Sprintf("✅ ウェルカムメッセージを <#%s> に設定しました。", channel.ID)
	} else {
		response = "✅ ウェルカムメッセージを無効にしました。"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: response, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *WelcomeCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *WelcomeCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *WelcomeCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *WelcomeCommand) GetCategory() string                                                  { return "管理" }
