package commands

import (
	"luna/interfaces"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AppContext provides dependencies to commands.
type AppContext struct {
	Log       interfaces.Logger
	Store     interfaces.DataStore
	Scheduler interfaces.Scheduler
	StartTime time.Time
	Player    interfaces.MusicPlayer
}

// RegisterCommands initializes and returns all command handlers.
func RegisterCommands(log interfaces.Logger, db interfaces.DataStore, scheduler interfaces.Scheduler, player interfaces.MusicPlayer, session *discordgo.Session, startTime time.Time) (map[string]interfaces.CommandHandler, map[string]interfaces.CommandHandler, []*discordgo.ApplicationCommand) {
	commandHandlers := make(map[string]interfaces.CommandHandler)
	componentHandlers := make(map[string]interfaces.CommandHandler)
	registeredCommands := make([]*discordgo.ApplicationCommand, 0)

	appCtx := &AppContext{
		Log:       log,
		Store:     db,
		Scheduler: scheduler,
		Player:    player,
		StartTime: startTime,
	}

	// To add a new command, simply add it to this list.
	commands := []interfaces.CommandHandler{
		&ConfigCommand{Store: appCtx.Store, Log: appCtx.Log},
		&DashboardCommand{Store: appCtx.Store, Scheduler: appCtx.Scheduler, Log: appCtx.Log},
		&TicketCommand{Store: appCtx.Store, Log: appCtx.Log},
		&PingCommand{StartTime: appCtx.StartTime, Store: appCtx.Store},
		&AskCommand{Log: appCtx.Log},
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
		&QuizCommand{Log: appCtx.Log, Store: appCtx.Store},
		&DescribeImageCommand{Log: appCtx.Log},
		&OcrCommand{Log: appCtx.Log},
		&ProfileCommand{Log: appCtx.Log, Store: appCtx.Store},
		&WordCountCommand{Store: appCtx.Store, Log: appCtx.Log},
		&WordRankingCommand{Store: appCtx.Store, Log: appCtx.Log},
		&WordConfigCommand{Store: appCtx.Store, Log: appCtx.Log},
		&RouletteCommand{Log: appCtx.Log},
		&JoinCommand{Player: appCtx.Player, Log: appCtx.Log},
		&PlayCommand{Player: appCtx.Player, Log: appCtx.Log},
		&StopCommand{Player: appCtx.Player, Log: appCtx.Log},
		&SkipCommand{Player: appCtx.Player, Log: appCtx.Log},
		&QueueCommand{Player: appCtx.Player, Log: appCtx.Log},
		&LeaveCommand{Player: appCtx.Player, Log: appCtx.Log},
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
		NewQuizBetCommand(appCtx.Store, appCtx.Log),
		NewBlackjackCommand(appCtx.Store, appCtx.Log),
		NewHiLowCommand(appCtx.Store, appCtx.Log),
		NewFishCommand(appCtx.Store, appCtx.Log),
		NewExchangeCommand(appCtx.Store, appCtx.Log),
		// NewShopCommand(appCtx.Store, appCtx.Log),
	}

	for _, cmd := range commands {
		commandDef := cmd.GetCommandDef()
		commandHandlers[commandDef.Name] = cmd
		registeredCommands = append(registeredCommands, commandDef)

		// Register component handlers
		for _, id := range cmd.GetComponentIDs() {
			componentHandlers[id] = cmd
		}
	}

	return commandHandlers, componentHandlers, registeredCommands
}
