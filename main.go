package main

import (
	"luna/commands"
	"luna/handlers"
	"luna/logger"
	"luna/storage"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

var (
	// コマンド名をキーとして、対応するハンドラを保持します
	commandHandlers map[string]handlers.CommandHandler

	// ボタンやモーダルのCustomIDをキーとして、対応するハンドラを保持します
	componentHandlers map[string]handlers.CommandHandler
)

func main() {
	// 1. 初期化処理
	logger.Init()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		logger.Fatal.Println("環境変数 'DISCORD_BOT_TOKEN' が設定されていません。")
		return
	}

	// 天気APIキーも環境変数から読み込みます
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		logger.Warning.Println("環境変数 'WEATHER_API_KEY' が設定されていません。Weatherコマンドは無効になります。")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal.Printf("Discordセッションの作成中にエラー: %v", err)
	}

	// 2. 依存関係のセットアップ
	// 設定ファイルを管理するストアを初期化
	configStore, err := storage.NewConfigStore("config.json")
	if err != nil {
		logger.Fatal.Fatalf("設定ストアの初期化に失敗: %v", err)
	}

	// スケジュールタスク（cron）を管理するスケジューラを初期化
	scheduler := cron.New()

	// 3. ハンドラの登録
	// 各ハンドラを格納するためのマップを初期化
	commandHandlers = make(map[string]handlers.CommandHandler)
	componentHandlers = make(map[string]handlers.CommandHandler)

	// --- ここに全てのコマンドを登録していきます ---
	// 引数が必要ないコマンド
	registerCommand(&commands.PingCommand{})
	registerCommand(&commands.HelpCommand{})
	registerCommand(&commands.UserInfoCommand{})
	registerCommand(&commands.CalculatorCommand{})
	registerCommand(&commands.PollCommand{})
	registerCommand(&commands.EmbedCommand{})     // モーダルを持つ
	registerCommand(&commands.TranslateCommand{}) // モーダルを持つ

	// 依存を注入する必要があるコマンド
	registerCommand(&commands.ConfigCommand{Store: configStore})
	registerCommand(&commands.TicketCommand{Store: configStore})
	registerCommand(&commands.ReactionRoleCommand{Store: configStore})
	registerCommand(&commands.ScheduleCommand{Scheduler: scheduler, Store: configStore})
	registerCommand(&commands.WeatherCommand{APIKey: weatherAPIKey})
	registerCommand(&commands.AskCommand{}) // askコマンドも登録

	// 4. Discordイベントハンドラの設定
	// すべてのインタラクション（コマンド、ボタン、モーダル）はこの関数に集約されます
	dg.AddHandler(interactionCreate)
	// リアクションロールのためのリアクションイベント
	dg.AddHandler(messageReactionAdd)
	dg.AddHandler(messageReactionRemove)

	// Botがサーバーに参加したときにログを出す
	dg.AddHandler(func(s *discordgo.Session, event *discordgo.GuildCreate) {
		logger.Info.Printf("サーバーに接続しました: %s (ID: %s)", event.Guild.Name, event.Guild.ID)
	})

	// 5. Botの起動
	err = dg.Open()
	if err != nil {
		logger.Fatal.Printf("Discordへの接続中にエラー: %v", err)
	}
	defer dg.Close()

	// スケジューラを開始
	scheduler.Start()
	defer scheduler.Stop()

	logger.Info.Println("Botが起動しました。スラッシュコマンドをDiscordに登録します...")

	// 登録するコマンドの定義リストを作成
	registeredCommands := make([]*discordgo.ApplicationCommand, 0, len(commandHandlers))
	for _, handler := range commandHandlers {
		registeredCommands = append(registeredCommands, handler.GetCommandDef())
	}

	// コマンドを一括で上書き登録
	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", registeredCommands)
	if err != nil {
		logger.Fatal.Printf("コマンドの登録に失敗しました: %v", err)
	}

	logger.Info.Println("コマンドの登録が完了しました。Ctrl+Cで終了します。")

	// 6. 終了シグナルを待機
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	logger.Info.Println("Botをシャットダウンします...")
}

// registerCommand はコマンドを各種ハンドラマップに登録するヘルパー関数です
func registerCommand(cmd handlers.CommandHandler) {
	def := cmd.GetCommandDef()
	commandHandlers[def.Name] = cmd

	// コマンドに紐づくコンポーネントやモーダルがあれば、それらのIDも登録します。
	// これにより、どのボタンがどのコマンドに属しているかを判別できます。
	switch def.Name {
	case "ticket-setup":
		componentHandlers[commands.CreateTicketButtonID] = cmd
		componentHandlers[commands.CloseTicketButtonID] = cmd // 閉じるボタンも追加
	case "embed":
		componentHandlers[commands.EmbedModalCustomID] = cmd
	case "translate":
		componentHandlers[commands.TranslateModalCustomID] = cmd
	}
}

// interactionCreate はすべてのインタラクションを処理する中央ハブです
func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			// コマンドが実行された場合
			h.Handle(s, i)
		}
	case discordgo.InteractionMessageComponent:
		if h, ok := componentHandlers[i.MessageComponentData().CustomID]; ok {
			// ボタンなどが押された場合
			h.HandleComponent(s, i)
		}
	case discordgo.InteractionModalSubmit:
		if h, ok := componentHandlers[i.ModalSubmitData().CustomID]; ok {
			// モーダルが送信された場合
			h.HandleModal(s, i)
		}
	}
}

// messageReactionAdd はリアクションロール機能のためのハンドラです
func messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// 自分自身のリアクションは無視
	if r.UserID == s.State.User.ID {
		return
	}

	// ReactionRoleCommandのハンドラを取得して処理を委譲
	// この部分は少しトリッキーですが、一つの方法です
	if cmd, ok := commandHandlers["reaction-role-setup"]; ok {
		// 型アサーションで具体的な型に変換
		if rrCmd, ok := cmd.(*commands.ReactionRoleCommand); ok {
			rrCmd.HandleReactionAdd(s, r)
		}
	}
}

// messageReactionRemove はリアクションロール機能のためのハンドラです
func messageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}

	if cmd, ok := commandHandlers["reaction-role-setup"]; ok {
		if rrCmd, ok := cmd.(*commands.ReactionRoleCommand); ok {
			rrCmd.HandleReactionRemove(s, r)
		}
	}
}
