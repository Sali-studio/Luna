package commands

import (
	"fmt"
	"luna/logger"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	modalKickPrefix    = "moderate_kick_confirm:"
	modalBanPrefix     = "moderate_ban_confirm:"
	modalTimeoutPrefix = "moderate_timeout_confirm:"
)

type ModerateCommand struct{}

func (c *ModerateCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "moderate",
		Description:              "ユーザーに対する管理操作を行います。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionKickMembers | discordgo.PermissionBanMembers | discordgo.PermissionModerateMembers),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "kick",
				Description: "ユーザーをサーバーから追放します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "追放するユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "追放する理由", Required: false},
				},
			},
			{
				Name:        "ban",
				Description: "ユーザーをサーバーからBANします。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "BANするユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "BANする理由", Required: false},
				},
			},
			{
				Name:        "timeout",
				Description: "ユーザーをタイムアウトさせます。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "タイムアウトさせるユーザー", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "duration", Description: "期間 (例: 5m, 1h, 3d)", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "タイムアウトさせる理由", Required: false},
				},
			},
		},
	}
}

// Handle はサブコマンドに応じて適切なモーダルを表示します
func (c *ModerateCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subcommand := i.ApplicationCommandData().Options[0]
	switch subcommand.Name {
	case "kick":
		c.showKickModal(s, i, subcommand.Options)
	case "ban":
		c.showBanModal(s, i, subcommand.Options)
	case "timeout":
		c.showTimeoutModal(s, i, subcommand.Options)
	}
}

// HandleModal は送信されたモーダルに応じて処理を実行します
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

// --- モーダル表示処理 ---
func (c *ModerateCommand) showKickModal(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandOption) {
	userID := options[0].UserValue(s).ID
	reason := ""
	if len(options) > 1 {
		reason = options[1].StringValue()
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalKickPrefix + userID,
			Title:    "Kick実行確認",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "reason", Label: "理由（変更可能）", Style: discordgo.TextInputParagraph, Value: reason, Required: true},
				}},
			},
		},
	})
	if err != nil {
		logger.Error.Printf("Kickモーダルの表示に失敗: %v", err)
	}
}

// (showBanModal, showTimeoutModal も同様に実装)

// --- 実行処理 ---
func (c *ModerateCommand) executeKick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := strings.TrimPrefix(i.ModalSubmitData().CustomID, modalKickPrefix)
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	err := s.GuildMemberDeleteWithReason(i.GuildID, userID, reason)
	if err != nil {
		logger.Error.Printf("Kickの実行に失敗: %v", err)
		// ... エラーレスポンス ...
		return
	}
	response := fmt.Sprintf("✅ ユーザーを理由「%s」でサーバーから追放しました。", reason)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: response, Flags: discordgo.MessageFlagsEphemeral},
	})
}

// (executeBan, executeTimeout も同様に実装)

func (c *ModerateCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}

// showBanModal, showTimeoutModal, executeBan, executeTimeout の中身は長くなるため省略しますが、
// executeKick と同様のロジックで実装してください。
