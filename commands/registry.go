package commands

import (
	"os"
	"time"

	"luna/bot"
	"luna/logger"
)

// AppContext provides dependencies to commands.
type AppContext struct {
	Log       logger.Logger
	Store     bot.DataStore
	Scheduler bot.Scheduler
	StartTime time.Time
}

// RegisterAllCommands initializes and returns all command handlers.
func RegisterAllCommands(ctx *AppContext, allCommands map[string]CommandHandler) []CommandHandler {
	return []CommandHandler{
		&ConfigCommand{Store: ctx.Store},
		&DashboardCommand{Store: ctx.Store, Scheduler: ctx.Scheduler},
		&ReactionRoleCommand{Store: ctx.Store},
		&ScheduleCommand{Scheduler: ctx.Scheduler, Store: ctx.Store},
		&TicketCommand{Store: ctx.Store},
		&PingCommand{StartTime: ctx.StartTime, Store: ctx.Store},
		&AskCommand{},
		&AvatarCommand{},
		&CalculatorCommand{},
		&EmbedCommand{},
		&ModerateCommand{},
		&PokemonCalculatorCommand{},
		&PollCommand{},
		&PowerConverterCommand{},
		&TranslateCommand{},
		&UserInfoCommand{},
		&WeatherCommand{APIKey: os.Getenv("WEATHER_API_KEY")},
		&HelpCommand{AllCommands: allCommands},
		&ImagineCommand{},
		&MusicCommand{},
		&QuizCommand{},
		// To add a new command, simply add it to this list.
	}
}
