package commands

import (
	"fmt"
	"luna/interfaces"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	modalKickPrefix    = "moderate_kick_confirm:"
	modalBanPrefix     = "moderate_ban_confirm:"
	modalTimeoutPrefix = "moderate_timeout_confirm:"
)

type ModerateCommand struct {
	Log interfaces.Logger
}

func (c *ModerateCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "moderate",
		Description:              "ユーザーに対する管理操作を行います。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionKickMembers | discordgo.PermissionBanMembers | discordgo.PermissionModerateMembers),
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "kick", Description: "ユーザーをサーバーから追放します。", Type: discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "追放するユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "追放する理由", Required: false},
				},
			},
			{Name: "ban", Description: "ユーザーをサーバーからBANします。", Type: discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "BANするユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "BANする理由", Required: false},
				},
			},
			{Name: "timeout", Description: "ユーザーをタイムアウトさせます。", Type: discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "タイムアウトさせるユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "duration", Description: "期間 (例: 5m, 1h, 3d)", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "タイムアウトさせる理由", Required: false},
				},
			},
		},
	}
}

func (c *ModerateCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subcommand := i.ApplicationCommandData().Options[0]
	switch subcommand.Name {
	case "kick":
		c.showKickModal(s, i)
	case "ban":
		c.showBanModal(s, i)
	case "timeout":
		c.showTimeoutModal(s, i)
	}
}

func (c *ModerateCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	switch {
	case strings.HasPrefix(customID, modalKickPrefix):
		c.executeKick(s, i)
	case strings.HasPrefix(customID, modalBanPrefix):
		c.executeBan(s, i)
	case strings.HasPrefix(customID, modalTimeoutPrefix):
		c.executeTimeout(s, i)
	}
}

func (c *ModerateCommand) showKickModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	userID := options[0].UserValue(s).ID
	reason := ""
	if len(options) > 1 {
		reason = options[1].StringValue()
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalKickPrefix + userID, Title: "Kick実行確認",
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{CustomID: "reason", Label: "理由（変更可能）", Style: discordgo.TextInputParagraph, Value: reason, Required: true},
			}}},
		},
	})
}

func (c *ModerateCommand) executeKick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := strings.TrimPrefix(i.ModalSubmitData().CustomID, modalKickPrefix)
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	err := s.GuildMemberDeleteWithReason(i.GuildID, userID, reason)
	if err != nil {
		c.Log.Error("Kickの実行に失敗", "error", err, "guildID", i.GuildID, "userID", userID)
		return
	}
	response := fmt.Sprintf("✅ ユーザー <@%s> を理由「%s」でサーバーから追放しました。", userID, reason)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: response, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to send kick response", "error", err)
	}
}

func (c *ModerateCommand) showBanModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	userID := options[0].UserValue(s).ID
	reason := ""
	if len(options) > 1 {
		reason = options[1].StringValue()
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalBanPrefix + userID, Title: "BAN実行確認",
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{CustomID: "reason", Label: "理由（変更可能）", Style: discordgo.TextInputParagraph, Value: reason, Required: true},
			}}},
		},
	})
}

func (c *ModerateCommand) executeBan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := strings.TrimPrefix(i.ModalSubmitData().CustomID, modalBanPrefix)
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	err := s.GuildBanCreateWithReason(i.GuildID, userID, reason, 0)
	if err != nil {
		c.Log.Error("BANの実行に失敗", "error", err, "guildID", i.GuildID, "userID", userID)
		return
	}
	response := fmt.Sprintf("✅ ユーザー <@%s> を理由「%s」でサーバーからBANしました。", userID, reason)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: response, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to send ban response", "error", err)
	}
}

func (c *ModerateCommand) showTimeoutModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options
	userID := options[0].UserValue(s).ID
	durationStr := options[1].StringValue()
	reason := ""
	if len(options) > 2 {
		reason = options[2].StringValue()
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s%s:%s", modalTimeoutPrefix, userID, durationStr), Title: "Timeout実行確認",
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{CustomID: "reason", Label: "理由（変更可能）", Style: discordgo.TextInputParagraph, Value: reason, Required: true},
			}}},
		},
	}); err != nil {
		c.Log.Error("Failed to show timeout modal", "error", err)
	}
}

func (c *ModerateCommand) executeTimeout(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	parts := strings.Split(strings.TrimPrefix(customID, modalTimeoutPrefix), ":")
	userID, durationStr := parts[0], parts[1]
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.Log.Error("期間の解析に失敗", "error", err, "durationStr", durationStr)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "期間の形式が正しくあり��せん。(例: 5m, 1h, 3d)", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}
	until := time.Now().Add(duration)

	err = s.GuildMemberTimeout(i.GuildID, userID, &until)
	if err != nil {
		c.Log.Error("Timeoutの実行に失敗", "error", err, "guildID", i.GuildID, "userID", userID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "タイムアウトの実行に失敗しました。BOTの権限が不足している可能性があります。", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}
	response := fmt.Sprintf("✅ ユーザー <@%s> を理由「%s」で %s までタイムアウトしました。", userID, reason, until.Format("2006/01/02 15:04"))
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: response, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		c.Log.Error("Failed to send timeout response", "error", err)
	}
}

func (c *ModerateCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *ModerateCommand) GetComponentIDs() []string {
	return []string{modalKickPrefix, modalBanPrefix, modalTimeoutPrefix}
}
func (c *ModerateCommand) GetCategory() string { return "管理" }
