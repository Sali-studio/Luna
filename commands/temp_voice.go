package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "temp-vc-setup",
		Description:              "一時的なボイスチャンネルを作成する機能を自動でセットアップします。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageChannels),
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("temp-vc-setup command received")

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⏳ セットアップを開始します...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		category, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
			Name: "🎤 Temp Voice Channels",
			Type: discordgo.ChannelTypeGuildCategory,
		})
		if err != nil {
			logger.Error.Printf("Failed to create temp VC category: %v", err)
			return
		}

		lobby, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
			Name:     "➕ Join to Create",
			Type:     discordgo.ChannelTypeGuildVoice,
			ParentID: category.ID,
		})
		if err != nil {
			logger.Error.Printf("Failed to create lobby channel: %v", err)
			s.ChannelDelete(category.ID)
			return
		}

		tempVCLobbyID[i.GuildID] = lobby.ID
		tempVCCategoryID[i.GuildID] = category.ID

		content := "✅ セットアップが完了しました！\n新しく作成された「➕ Join to Create」チャンネルに参加すると一時的なVCが作成されます。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

func HandleVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	lobbyID, ok := tempVCLobbyID[v.GuildID]
	if !ok {
		return
	}
	if v.ChannelID == lobbyID {
		handleJoinLobby(s, v)
	}
	if v.BeforeUpdate != nil {
		if _, ok := tempVCCreated[v.BeforeUpdate.ChannelID]; ok {
			handleLeaveTempVC(s, v.BeforeUpdate.ChannelID)
		}
	}
}

func handleJoinLobby(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	categoryID := tempVCCategoryID[v.GuildID]
	user, _ := s.User(v.UserID)

	logger.Info.Printf("User %s joined lobby, creating a temp channel.", user.Username)

	channel, err := s.GuildChannelCreateComplex(v.GuildID, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("%sの部屋", user.Username),
		Type:     discordgo.ChannelTypeGuildVoice,
		ParentID: categoryID,
		Topic:    fmt.Sprintf("このチャンネルは%sによって作成されました。", user.Username),
		Bitrate:  64000,
	})
	if err != nil {
		logger.Error.Printf("Failed to create temp VC: %v", err)
		return
	}

	tempVCCreated[channel.ID] = v.UserID

	err = s.GuildMemberMove(v.GuildID, v.UserID, &channel.ID)
	if err != nil {
		logger.Error.Printf("Failed to move user to temp VC: %v", err)
		time.Sleep(5 * time.Second)
		s.ChannelDelete(channel.ID)
	}
}

func handleLeaveTempVC(s *discordgo.Session, channelID string) {
	channel, err := s.Channel(channelID)
	if err != nil {
		return
	}

	guild, _ := s.State.Guild(channel.GuildID)
	memberCount := 0
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == channelID {
			memberCount++
		}
	}

	if memberCount == 0 {
		logger.Info.Printf("Temp channel %s is empty, deleting.", channel.Name)
		_, err := s.ChannelDelete(channelID)
		if err == nil {
			delete(tempVCCreated, channelID)
		}
	}
}
