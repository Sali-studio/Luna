package commands

import "github.com/bwmarrin/discordgo"

// コマンド登録用の変数
var Commands = []*discordgo.ApplicationCommand{}
var CommandHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))

// チケット機能用の共有変数
var (
	ticketStaffRoleID = make(map[string]string)
	ticketCategoryID  = make(map[string]string)
	ticketCounter     = make(map[string]int)
)

// 共有ヘルパー関数
func int64Ptr(i int64) *int64 {
	return &i
}
