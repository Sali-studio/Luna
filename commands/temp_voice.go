package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

// HandleExecuteTempVCSetup は一時VCのセットアップを実行します
func HandleExecuteTempVCSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⏳ 一時VC機能のセットアップを開始します...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Error.Printf("Failed to defer temp vc setup response: %v", err)
		return
	}

	config := Config.GetGuildConfig(i.GuildID)

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

	config.TempVC.LobbyID = lobby.ID
	config.TempVC.CategoryID = category.ID
	Config.SaveGuildConfig(i.GuildID, config)

	content := "✅ セットアップが完了しました！\n新しく作成された「➕ Join to Create」チャンネルに参加すると一時的なVCが作成されます。"
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		logger.Error.Printf("Failed to edit interaction response for temp vc setup: %v", err)
	}
}

// HandleVoiceStateUpdate はボイスチャンネルの状態変化を処理します
func HandleVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	config := Config.GetGuildConfig(v.GuildID)
	lobbyID := config.TempVC.LobbyID
	if lobbyID == "" {
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
	config := Config.GetGuildConfig(v.GuildID)
	categoryID := config.TempVC.CategoryID
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
