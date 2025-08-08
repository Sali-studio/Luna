package commands

import (
	"luna/ai"
	"luna/interfaces"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AppContext provides dependencies to commands.
type AppContext struct {
	Log       interfaces.Logger
	Store     interfaces.DataStore
	Scheduler interfaces.Scheduler
	AI        *ai.Client
	StartTime time.Time
}

// RegisterCommands initializes and returns all command handlers.
func RegisterCommands(log interfaces.Logger, db interfaces.DataStore, scheduler interfaces.Scheduler, aiClient *ai.Client, session *discordgo.Session, startTime time.Time) (map[string]interfaces.CommandHandler, map[string]interfaces.CommandHandler, []*discordgo.ApplicationCommand, *StockCommand) {
	commandHandlers := make(map[string]interfaces.CommandHandler)
	componentHandlers := make(map[string]interfaces.CommandHandler)
	registeredCommands := make([]*discordgo.ApplicationCommand, 0)

	appCtx := &AppContext{
		Log:       log,
		Store:     db,
		Scheduler: scheduler,
		AI:        aiClient,
		StartTime: startTime,
	}

	stockCmd := NewStockCommand(appCtx.Store, appCtx.Log)

	// To add a new command, simply add it to this list.
	commands := []interfaces.CommandHandler{
		&ConfigCommand{Store: appCtx.Store, Log: appCtx.Log},
		&TicketCommand{Store: appCtx.Store, Log: appCtx.Log},
		&PingCommand{StartTime: appCtx.StartTime, Store: appCtx.Store},
		&AskCommand{Log: appCtx.Log, AI: appCtx.AI},
		&AvatarCommand{},
		&CalculatorCommand{Log: appCtx.Log},
		&EmbedCommand{Log: appCtx.Log},
		&ModerateCommand{Log: appCtx.Log},
		&PokemonCalculatorCommand{Log: appCtx.Log},
		&PollCommand{Log: appCtx.Log},
		&PowerConverterCommand{Log: appCtx.Log},
		&TranslateCommand{Log: appCtx.Log},
		&UserInfoCommand{Log: appCtx.Log},
		&HelpCommand{AllCommands: commandHandlers},
		&ImagineCommand{Log: appCtx.Log},
		&DescribeImageCommand{Log: appCtx.Log},
		&OcrCommand{Log: appCtx.Log},
		&ProfileCommand{Log: appCtx.Log, Store: appCtx.Store},
		&WordCountCommand{Store: appCtx.Store, Log: appCtx.Log},
		&WordRankingCommand{Store: appCtx.Store, Log: appCtx.Log},
		&WordConfigCommand{Store: appCtx.Store, Log: appCtx.Log},
		&RouletteCommand{Log: appCtx.Log},
		&WTBRCommand{Log: appCtx.Log},
		&AutoRoleCommand{Store: appCtx.Store, Log: appCtx.Log},
		// Casino Commands
		&DailyCommand{Store: appCtx.Store, Log: appCtx.Log},
		&BalanceCommand{Store: appCtx.Store, Log: appCtx.Log},
		&SlotsCommand{Store: appCtx.Store, Log: appCtx.Log},
		&LeaderboardCommand{Store: appCtx.Store, Log: appCtx.Log},
		&CoinflipCommand{Store: appCtx.Store, Log: appCtx.Log},
		&PayCommand{Store: appCtx.Store, Log: appCtx.Log},
		NewHorseRaceCommand(appCtx.Store, appCtx.Log),
		NewQuizCommand(appCtx.Store, appCtx.Log),
		NewBlackjackCommand(appCtx.Store, appCtx.Log),
		NewHiLowCommand(appCtx.Store, appCtx.Log),
		NewFishCommand(appCtx.Store, appCtx.Log),
		NewExchangeCommand(appCtx.Store, appCtx.Log),
		stockCmd,
		// NewShopCommand(appCtx.Store, appCtx.Log),
	}

	for _, cmd := range commands {
		cmdHandler := cmd // ローカル変数にコピー
		commandDef := cmdHandler.GetCommandDef()
		commandHandlers[commandDef.Name] = cmdHandler
		registeredCommands = append(registeredCommands, commandDef)

		// Register component handlers
		for _, id := range cmdHandler.GetComponentIDs() {
			componentHandlers[id] = cmdHandler
		}
	}

	// ラッパーハンドラーを作成して、元のハンドラーをラップする
	for name, handler := range commandHandlers {
		originalHandler := handler // クロージャのためにコピー
		wrappedHandler := &CommandUsageWrapper{
			CommandHandler: originalHandler,
			Store:          db,
		}
		commandHandlers[name] = wrappedHandler
	}

	return commandHandlers, componentHandlers, registeredCommands, stockCmd
}

// CommandUsageWrapper は、コマンドの実行をラップして使用状況を記録します。
type CommandUsageWrapper struct {
	interfaces.CommandHandler
	Store interfaces.DataStore
}

// Handle は、元のハンドラを呼び出す前に使用状況を記録します。
func (w *CommandUsageWrapper) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	category := w.CommandHandler.GetCategory()
	if category != "" && category != "管理" { // 管理コマンドは経済に影響を与えない
		w.Store.IncrementCommandUsage(category)
	}
	w.CommandHandler.Handle(s, i)
}