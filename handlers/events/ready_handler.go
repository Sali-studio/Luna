package events

import (
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// OnReady は、Botの準備ができたときに呼び出され、ステータスを設定します。
func OnReady(s *discordgo.Session, r *discordgo.Ready, log interfaces.Logger) {
	log.Info("Bot is ready! Logged in as: ", r.User.String())
	s.UpdateGameStatus(0, "/help | Luna")
}
