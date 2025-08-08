package commands

import (
	"fmt"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

const (
	PpcToChipsRate = 10
)

// ExchangeCommand handles the /exchange command.
type ExchangeCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

// NewExchangeCommand creates a new ExchangeCommand.
func NewExchangeCommand(store interfaces.DataStore, log interfaces.Logger) *ExchangeCommand {
	return &ExchangeCommand{
		Store: store,
		Log:   log,
	}
}

func (c *ExchangeCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "exchange",
		Description: "PepeCoinとチップを両替します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "ppc_to_chips",
				Description: "PepeCoinをチップに両替します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "amount",
						Description: "両替するPepeCoinの額",
						Required:    true,
						MinValue:    &[]float64{1}[0],
					},
				},
			},
			{
				Name:        "chips_to_ppc",
				Description: "チップをPepeCoinに両替します。",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "amount",
						Description: "両替するチップの額",
						Required:    true,
						MinValue:    &[]float64{10}[0], // Must be at least 10 to get 1 PPC
					},
				},
			},
		},
	}
}

func (c *ExchangeCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subCommand := i.ApplicationCommandData().Options[0]
	switch subCommand.Name {
	case "ppc_to_chips":
		c.handlePpcToChips(s, i)
	case "chips_to_ppc":
		c.handleChipsToPpc(s, i)
	}
}

func (c *ExchangeCommand) handlePpcToChips(s *discordgo.Session, i *discordgo.InteractionCreate) {
	amount := i.ApplicationCommandData().Options[0].Options[0].IntValue()
	userID := i.Member.User.ID
	guildID := i.GuildID

	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for exchange", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if casinoData.PepeCoinBalance < amount {
		sendErrorResponse(s, i, fmt.Sprintf("PepeCoinが足りません！\n現在のPPC: `%d`", casinoData.PepeCoinBalance))
		return
	}

	casinoData.PepeCoinBalance -= amount
	chipsToReceive := amount * PpcToChipsRate
	casinoData.Chips += chipsToReceive

	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data for exchange", "error", err)
		sendErrorResponse(s, i, "両替処理中にエラーが発生しました。")
		return
	}

	response := fmt.Sprintf("🐸 **%d PPC** を 💰 **%d チップ** に両替しました。", amount, chipsToReceive)
	sendSuccessResponse(s, i, response)
}

func (c *ExchangeCommand) handleChipsToPpc(s *discordgo.Session, i *discordgo.InteractionCreate) {
	amount := i.ApplicationCommandData().Options[0].Options[0].IntValue()
	userID := i.Member.User.ID
	guildID := i.GuildID

	if amount%PpcToChipsRate != 0 {
		sendErrorResponse(s, i, fmt.Sprintf("チップは%dの倍数で入力してください。", PpcToChipsRate))
		return
	}

	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for exchange", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if casinoData.Chips < amount {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！\n現在のチップ: `%d`", casinoData.Chips))
		return
	}

	casinoData.Chips -= amount
	ppcToReceive := amount / PpcToChipsRate
	casinoData.PepeCoinBalance += ppcToReceive

	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data for exchange", "error", err)
		sendErrorResponse(s, i, "両替処理中にエラーが発生しました。")
		return
	}

	response := fmt.Sprintf("💰 **%d チップ** を 🐸 **%d PPC** に両替しました。", amount, ppcToReceive)
	sendSuccessResponse(s, i, response)
}

func (c *ExchangeCommand) GetCategory() string {
	return "経済"
}

func (c *ExchangeCommand) GetComponentIDs() []string {
	return nil
}

func (c *ExchangeCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}

func (c *ExchangeCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
