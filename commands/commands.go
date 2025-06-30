package commands

import "github.com/bwmarrin/discordgo"

// --- コマンド登録用の変数 ---
var Commands = []*discordgo.ApplicationCommand{}
var CommandHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))

// --- 機能ごとの共有変数 ---
var (
	// Ticket機能用
	ticketStaffRoleID = make(map[string]string)
	ticketCategoryID  = make(map[string]string)
	ticketCounter     = make(map[string]int)
	// ログ機能用
	logChannelID = make(map[string]string)
	// 一時VC機能用
	tempVCLobbyID    = make(map[string]string)
	tempVCCategoryID = make(map[string]string)
	tempVCCreated    = make(map[string]string)
)

// --- 共有ヘルパー関数 ---
func int64Ptr(i int64) *int64 {
	return &i
}
