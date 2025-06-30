package commands

import "github.com/bwmarrin/discordgo"

var Commands = []*discordgo.ApplicationCommand{}
var CommandHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))

// この関数をここに一ヶ所だけ定義
func int64Ptr(i int64) *int64 {
	return &i
}
