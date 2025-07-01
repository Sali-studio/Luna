package handlers

import "github.com/bwmarrin/discordgo"

// CommandHandler はすべてのスラッシュコマンドが実装すべきインターフェースです。
type CommandHandler interface {
	GetCommandDef() *discordgo.ApplicationCommand
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
	GetComponentIDs() []string // コマンドが反応するComponentのIDプレフィックスを返す
}
