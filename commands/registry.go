package commands

import (
	"os"
	"time"

	"luna/interfaces"
)

// AppContext provides dependencies to commands.
type AppContext struct {
	Log       interfaces.Logger
	Store     interfaces.DataStore
	Scheduler interfaces.Scheduler
	StartTime time.Time
}

// RegisterAllCommands initializes and returns all command handlers.
func RegisterAllCommands(ctx *AppContext, allCommands map[string]interfaces.CommandHandler) []interfaces.CommandHandler {
	return []interfaces.CommandHandler{
		&ConfigCommand{Store: ctx.Store, Log: ctx.Log},
		&DashboardCommand{Store: ctx.Store, Scheduler: ctx.Scheduler, Log: ctx.Log},
		&ReactionRoleCommand{Store: ctx.Store, Log: ctx.Log},
		&ScheduleCommand{Scheduler: ctx.Scheduler, Store: ctx.Store, Log: ctx.Log},
		&TicketCommand{Store: ctx.Store, Log: ctx.Log},
		&PingCommand{StartTime: ctx.StartTime, Store: ctx.Store},
		&AskCommand{Log: ctx.Log},
		&AvatarCommand{},
		&CalculatorCommand{Log: ctx.Log},
		&EmbedCommand{Log: ctx.Log},
		&ModerateCommand{Log: ctx.Log},
		&PokemonCalculatorCommand{Log: ctx.Log},
		&PollCommand{Log: ctx.Log},
		&PowerConverterCommand{Log: ctx.Log},
		&TranslateCommand{Log: ctx.Log},
		&UserInfoCommand{Log: ctx.Log},
		&WeatherCommand{APIKey: os.Getenv("WEATHER_API_KEY"), Log: ctx.Log},
		&HelpCommand{AllCommands: allCommands},
		&ImagineCommand{Log: ctx.Log},
		&MusicCommand{Log: ctx.Log},
		&QuizCommand{Log: ctx.Log},
		// To add a new command, simply add it to this list.
	}
}
