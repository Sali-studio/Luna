// commands/command.go
package commands

import "github.com/bwmarrin/discordgo"

// CommandHandler は、すべてのスラッシュコマンドが実装すべきインターフェースです。
type CommandHandler interface {
	GetCommandDef() *discordgo.ApplicationCommand
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
	GetComponentIDs() []string
	GetCategory() string
}
