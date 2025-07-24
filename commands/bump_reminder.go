package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

// BumpReminderCommand は /bump-reminder コマンドを処理します。
type BumpReminderCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

// GetCommandDef はコマンドの定義を返します。
func (c *BumpReminderCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "bump-reminder",
		Description: "Bump通知のリマインダーを設定します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "enable",
				Description: "指定したチャンネルでBumpリマインダーを有効にします。",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionChannel,
						Name:        "channel",
						Description: "監視するチャンネル",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "disable",
				Description: "Bumpリマインダーを無効にします。",
			},
		},
	}
}

// Handle はコマンドの実行を処理します。
func (c *BumpReminderCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Options[0].Name {
	case "enable":
		c.handleEnable(s, i)
	case "disable":
		c.handleDisable(s, i)
	}
}

func (c *BumpReminderCommand) handleEnable(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel := i.ApplicationCommandData().Options[0].Options[0].ChannelValue(s)

	config := storage.BumpReminderConfig{
		Enabled:   true,
		ChannelID: channel.ID,
	}

	if err := c.Store.SaveConfig(i.GuildID, "bump_reminder_config", &config); err != nil {
		c.Log.Error("Failed to save bump reminder config", "error", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "設定の保存に失敗しました。",
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<#%s> でBumpリマインダーを有効にしました。", channel.ID),
		},
	})
}

func (c *BumpReminderCommand) handleDisable(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := storage.BumpReminderConfig{
		Enabled:   false,
		ChannelID: "",
	}

	if err := c.Store.SaveConfig(i.GuildID, "bump_reminder_config", &config); err != nil {
		c.Log.Error("Failed to save bump reminder config", "error", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "設定の保存に失敗しました。",
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Bumpリマインダーを無効にしました。",
		},
	})
}

// GetComponentIDs はこのコマンドが処理するComponentのIDを返します。
func (c *BumpReminderCommand) GetComponentIDs() []string {
	return nil
}

// HandleComponent はComponentの実行を処理します。
func (c *BumpReminderCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}

// HandleModal はModalの実行を処理します。
func (c *BumpReminderCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}

// GetCategory はコマンドのカテゴリを返します。
func (c *BumpReminderCommand) GetCategory() string {
	return "設定"
}
