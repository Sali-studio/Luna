package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func HandleExecuteKick(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	userID := parts[1]
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	targetUser, _ := s.User(userID)

	err := s.GuildMemberDeleteWithReason(i.GuildID, userID, reason)
	if err != nil {
		// エラー処理
		return
	}
	response := fmt.Sprintf("✅ **%s** を理由「%s」でサーバーから追放しました。", targetUser.Username, reason)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: response},
	})
}

func HandleExecuteBan(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	userID := parts[1]
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	targetUser, _ := s.User(userID)

	err := s.GuildBanCreateWithReason(i.GuildID, userID, reason, 0)
	if err != nil {
		// エラー処理
		return
	}
	response := fmt.Sprintf("✅ **%s** を理由「%s」でサーバーからBANしました。", targetUser.Username, reason)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: response},
	})
}

func HandleExecuteTimeout(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	userID := parts[1]
	durationStr := parts[2]
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	targetUser, _ := s.User(userID)

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// エラー処理
		return
	}
	until := time.Now().Add(duration)

	_, err = s.GuildMemberEdit(i.GuildID, userID, &discordgo.GuildMemberParams{
		CommunicationDisabledUntil: &until,
	}, discordgo.WithAuditLogReason(reason))
	if err != nil {
		// エラー処理
		return
	}
	response := fmt.Sprintf("✅ **%s** を **%s** の期間タイムアウトさせました。 (理由: %s)", targetUser.Username, durationStr, reason)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: response},
	})
}
