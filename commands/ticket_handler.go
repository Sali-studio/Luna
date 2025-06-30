package commands

import (
	"fmt"
	"luna/logger"

	"github.com/bwmarrin/discordgo"
)

// HandleTicketCreation はチケット作成ボタンが押されたときの処理を行います
func HandleTicketCreation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ボタンを押したユーザーの情報を取得
	user := i.Member.User
	// スタッフロールのIDを取得
	staffRoleID := ticketStaffRoleID[i.GuildID]

	// チャンネルの権限設定を作成
	permissionOverwrites := []*discordgo.PermissionOverwrite{
		// @everyone にはチャンネルを非表示にする
		{
			ID:   i.GuildID,
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel,
		},
		// チケットを作成した本人には表示する
		{
			ID:    user.ID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
		},
		// スタッフロールにも表示する
		{
			ID:    staffRoleID,
			Type:  discordgo.PermissionOverwriteTypeRole,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
		},
		// ボット自身にも表示する
		{
			ID:    s.State.User.ID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
		},
	}

	// 新しいプライベートチャンネルを作成
	ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
		Name:                 fmt.Sprintf("ticket-%s", user.Username),
		Type:                 discordgo.ChannelTypeGuildText,
		PermissionOverwrites: permissionOverwrites,
	})
	if err != nil {
		logger.Error.Printf("Failed to create ticket channel: %v", err)
		return
	}

	// ボタンを押したことに対する応答（Ephemeralで本人にだけ見える）
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ チケットを作成しました: <#%s>", ch.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// 作成されたチャンネルにメッセージを送信
	// TODO: ここにチケットを閉じるボタンなどを追加すると、より高機能になる
	s.ChannelMessageSend(ch.ID, fmt.Sprintf("ようこそ <@%s> さん。\n<@&%s> が対応しますので、ご用件をお書きください。", user.ID, staffRoleID))
}
