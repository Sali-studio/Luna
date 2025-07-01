package handlers

import "github.com/bwmarrin/discordgo"

// CommandHandler は、Discordのスラッシュコマンドを処理するためのインターフェースです。
type CommandHandler interface {
	// GetCommandDef は、Discordに登録するためのコマンド定義を返します。
	GetCommandDef() *discordgo.ApplicationCommand

	// Handle は、コマンドが実行されたときの処理を担います。
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate)

	// HandleComponent は、そのコマンドに関連するボタンや選択メニューが操作されたときの処理を担います。
	// このコマンドに関連するComponentがない場合は、nilを返すか、何もしない関数を実装します。
	HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate)

	// HandleModal は、そのコマンドに関連するモーダルが送信されたときの処理を担います。
	// このコマンドに関連するModalがない場合は、nilを返すか、何もしない関数を実装します。
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
}
