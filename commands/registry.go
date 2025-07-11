package commands

import (
	"os"
	"time"

	"luna/storage"

	"github.com/robfig/cron/v3"
)

// AppContext provides dependencies to commands.
type AppContext struct {
	Store     *storage.DBStore
	Scheduler *cron.Cron
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
		&AIGameCommand{},
		// To add a new command, simply add it to this list.
	}
}
