package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

// HandleVoiceStateUpdate はボイスチャンネルの状態変化を処理します
func HandleVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	config := Config.GetGuildConfig(v.GuildID)
	lobbyID := config.TempVC.LobbyID
	if lobbyID == "" {
		return
	}

	// ユーザーがロビーチャンネルに参加したかを確認
	if v.ChannelID == lobbyID {
		handleJoinLobby(s, v)
	}

	// ユーザーがVCから退出したか、別のVCに移動したかを確認
	if v.BeforeUpdate != nil {
		if _, ok := tempVCCreated[v.BeforeUpdate.ChannelID]; ok {
			handleLeaveTempVC(s, v.BeforeUpdate.ChannelID)
		}
	}
}

// handleJoinLobby はユーザーがロビーに参加した際の処理
func handleJoinLobby(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	config := Config.GetGuildConfig(v.GuildID)
	categoryID := config.TempVC.CategoryID
	user, _ := s.User(v.UserID)

	logger.Info.Printf("User %s joined lobby, creating a temp channel.", user.Username)

	channel, err := s.GuildChannelCreateComplex(v.GuildID, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("%sの部屋", user.Username),
		Type:     discordgo.ChannelTypeGuildVoice,
		ParentID: categoryID,
		Topic:    fmt.Sprintf("このチャンネルは%sによって作成されました。全員が退出すると自動的に削除されます。", user.Username),
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

// handleLeaveTempVC はユーザーが一時VCから退出した際の処理
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
