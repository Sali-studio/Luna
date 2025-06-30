package commands

import "github.com/bwmarrin/discordgo"

// すべてのコマンド定義を格納するスライス
var Commands = []*discordgo.ApplicationCommand{}

// すべてのコマンドハンドラを格納するマップ
var CommandHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))

// サーバー(Guild)ごとのボイス接続を保持するためのマップ
var VoiceConnections = make(map[string]*discordgo.VoiceConnection)
