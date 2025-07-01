package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"luna/commands"
	"luna/handlers"
	"luna/logger"
	"luna/storage"

	"github.com/bwmarrin/discordgo"
)

var (
	// コマンド名とハンドラのマップ
	commandHandlers map[string]handlers.CommandHandler
	// コンポーネントのCustomIDとハンドラのマップ
	componentHandlers map[string]handlers.CommandHandler
)

func main() {
	logger.Init()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		logger.Fatal.Println("環境変数 'DISCORD_BOT_TOKEN' が設定されていません。")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal.Printf("Discordセッションの作成中にエラー: %v", err)
	}

	// --- 依存関係の初期化 ---
	configStore, err := storage.NewConfigStore("config.json")
	if err != nil {
		logger.Fatal.Fatalf("設定ストアの初期化に失敗: %v", err)
	}

	// --- ハンドラの初期化 ---
	commandHandlers = make(map[string]handlers.CommandHandler)
	componentHandlers = make(map[string]handlers.CommandHandler)

	// 各コマンドを初期化し、必要な依存を注入する
	// ここに全てのコマンドを追加していく
	registerCommand(&commands.PingCommand{})
	registerCommand(&commands.HelpCommand{})
	registerCommand(&commands.ConfigCommand{Store: configStore})
	registerCommand(&commands.TicketCommand{Store: configStore})
	// ... 全てのコマンドを同様に登録 ...

	// --- イベントハンドラの設定 ---
	dg.AddHandler(interactionCreate)

	// --- Botの起動 ---
	err = dg.Open()
	if err != nil {
		logger.Fatal.Printf("Discordへの接続中にエラー: %v", err)
	}
	defer dg.Close()

	logger.Info.Println("Botが起動しました。コマンドを登録します...")

	// 登録するコマンドのリストを作成
	registeredCommands := make([]*discordgo.ApplicationCommand, 0, len(commandHandlers))
	for _, handler := range commandHandlers {
		registeredCommands = append(registeredCommands, handler.GetCommandDef())
	}

	// コマンドを一括登録
	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", registeredCommands)
	if err != nil {
		logger.Fatal.Printf("コマンドの登録に失敗しました: %v", err)
	}

	logger.Info.Println("コマンドの登録が完了しました。Ctrl+Cで終了します。")

	// 終了シグナルを待機
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

// registerCommand はコマンドをハンドラマップに登録するヘルパー関数
func registerCommand(cmd handlers.CommandHandler) {
	def := cmd.GetCommandDef()
	commandHandlers[def.Name] = cmd

	// チケットボタンのような、コマンドに紐づくコンポーネントを登録
	// ここでは例としてCustomIDのプレフィックスで判定
	if strings.HasPrefix(commands.CreateTicketButtonID, def.Name) {
		componentHandlers[commands.CreateTicketButtonID] = cmd
	}
	// embedのモーダルのように、他のコンポーネントもここで登録していく
}

// interactionCreate はすべてのインタラクションを処理する中央ハブです
func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h.Handle(s, i)
		}
	case discordgo.InteractionMessageComponent:
		if h, ok := componentHandlers[i.MessageComponentData().CustomID]; ok {
			h.HandleComponent(s, i)
		}
	case discordgo.InteractionModalSubmit:
		// ModalのIDもcomponentHandlersと同様の仕組みで管理できます
		if h, ok := componentHandlers[i.ModalSubmitData().CustomID]; ok {
			h.HandleModal(s, i)
		}
	}
}
