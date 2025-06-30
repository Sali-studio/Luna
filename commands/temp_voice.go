package commands

import (
	"fmt"
	"luna/logger"
	"time"

	"github.com/bwmarrin/discordgo"
)

// --- 共有変数 ---
var (
	// Key: GuildID, Value: LobbyChannelID
	tempVCLobbyID = make(map[string]string)
	// Key: GuildID, Value: CategoryID
	tempVCCategoryID = make(map[string]string)
	// Key: CreatedChannelID, Value: CreatorUserID
	tempVCCreated = make(map[string]string)
)

func init() {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "temp-vc-setup",
		Description:              "参加すると一時的なVCが作成されるロビーチャンネルを設定します。",
		DefaultMemberPermissions: int64Ptr(discordgo.PermissionManageChannels),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "lobby-channel",
				Description:  "ロビーとして設定するボイスチャンネル",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildVoice},
				Required:     true,
			},
			// ★★★ カテゴリを指定するオプションを追加 ★★★
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "category",
				Description:  "作成されたVCを格納するカテゴリ",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory},
				Required:     true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logger.Info.Println("temp-vc-setup command received")

		options := i.ApplicationCommandData().Options
		lobbyChannel := options[0].ChannelValue(s)
		category := options[1].ChannelValue(s)

		// 設定を保存
		tempVCLobbyID[i.GuildID] = lobbyChannel.ID
		tempVCCategoryID[i.GuildID] = category.ID

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ 一時VC作成ロビーを **%s** に、作成先カテゴリを **%s** に設定しました。", lobbyChannel.Name, category.Name),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	Commands = append(Commands, cmd)
	CommandHandlers[cmd.Name] = handler
}

// ★★★ ここからが新しいイベントハンドラのロジック ★★★

// HandleVoiceStateUpdate はボイスチャンネルの状態変化を処理します
func HandleVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// サーバーのロビー設定を取得
	lobbyID, ok := tempVCLobbyID[v.GuildID]
	if !ok {
		return // このサーバーでは機能がセットアップされていない
	}

	// --- チャンネル作成処理 ---
	// ユーザーがロビーチャンネルに参加したかを確認
	if v.ChannelID == lobbyID {
		handleJoinLobby(s, v)
	}

	// --- チャンネル削除処理 ---
	// ユーザーがVCから退出したか、別のVCに移動したかを確認
	// v.BeforeUpdate は「以前の状態」。nilでなければ、どこかのVCにいたことを示す
	if v.BeforeUpdate != nil {
		// 以前いたチャンネルが、ボットによって作成された一時VCかを確認
		if _, ok := tempVCCreated[v.BeforeUpdate.ChannelID]; ok {
			handleLeaveTempVC(s, v.BeforeUpdate.ChannelID)
		}
	}
}

// handleJoinLobby はユーザーがロビーに参加した際の処理
func handleJoinLobby(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	categoryID := tempVCCategoryID[v.GuildID]
	user, _ := s.User(v.UserID)

	logger.Info.Printf("User %s joined lobby, creating a temp channel.", user.Username)

	// 新しいボイスチャンネルを作成
	channel, err := s.GuildChannelCreateComplex(v.GuildID, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("%sの部屋", user.Username),
		Type:     discordgo.ChannelTypeGuildVoice,
		ParentID: categoryID,
		Topic:    fmt.Sprintf("このチャンネルは%sによって作成されました。全員が退出すると自動的に削除されます。", user.Username),
		Bitrate:  64000, // ビットレート
	})
	if err != nil {
		logger.Error.Printf("Failed to create temp VC: %v", err)
		return
	}

	// 作成したチャンネルの情報を記録
	tempVCCreated[channel.ID] = v.UserID

	// ユーザーを作成したチャンネルに移動
	err = s.GuildMemberMove(v.GuildID, v.UserID, &channel.ID)
	if err != nil {
		logger.Error.Printf("Failed to move user to temp VC: %v", err)
		// 移動に失敗した場合でも、チャンネルが残り続けないように削除処理を入れる
		time.Sleep(5 * time.Second) // 念のため少し待つ
		s.ChannelDelete(channel.ID)
	}
}

// handleLeaveTempVC はユーザーが一時VCから退出した際の処理
func handleLeaveTempVC(s *discordgo.Session, channelID string) {
	// チャンネル情報を取得して、現在の人数を確認
	channel, err := s.Channel(channelID)
	if err != nil {
		return // チャンネルがすでにない場合は何もしない
	}

	// Discord APIの仕様上、Membersフィールドは使えないため、サーバー全体のVoiceStateから探す
	guild, _ := s.State.Guild(channel.GuildID)
	memberCount := 0
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == channelID {
			memberCount++
		}
	}

	// チャンネルに誰もいなくなったら、チャンネルを削除
	if memberCount == 0 {
		logger.Info.Printf("Temp channel %s is empty, deleting.", channel.Name)
		_, err := s.ChannelDelete(channelID)
		if err == nil {
			// チャンネル情報を記録から削除
			delete(tempVCCreated, channelID)
		}
	}
}
