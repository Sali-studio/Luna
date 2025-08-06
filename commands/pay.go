package commands

import (
	"fmt"
	"luna/interfaces"

	"github.com/bwmarrin/discordgo"
)

// PayCommand handles the /pay command.
type PayCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *PayCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "pay",
		Description: "他のユーザーにチップを送金します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "送金する相手",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "送金するチップの額",
				Required:    true,
				MinValue:    &[]float64{1}[0],
			},
		},
	}
}

func (c *PayCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	recipient := i.ApplicationCommandData().Options[0].UserValue(s)
	amount := i.ApplicationCommandData().Options[1].IntValue()
	senderID := i.Member.User.ID
	guildID := i.GuildID

	if recipient.ID == senderID {
		sendErrorResponse(s, i, "自分自身にチップを送ることはできません。")
		return
	}

	senderData, err := c.Store.GetCasinoData(guildID, senderID)
	if err != nil {
		c.Log.Error("Failed to get sender casino data for pay", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if senderData.Chips < amount {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！現在の所持チップ: %d", senderData.Chips))
		return
	}

	recipientData, err := c.Store.GetCasinoData(guildID, recipient.ID)
	if err != nil {
		c.Log.Error("Failed to get recipient casino data for pay", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	// Perform the transaction
	senderData.Chips -= amount
	recipientData.Chips += amount

	if err := c.Store.UpdateCasinoData(senderData); err != nil {
		c.Log.Error("Failed to update sender data after pay", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}
	if err := c.Store.UpdateCasinoData(recipientData); err != nil {
		c.Log.Error("Failed to update recipient data after pay", "error", err)
		// Attempt to revert the sender's transaction
		senderData.Chips += amount
		c.Store.UpdateCasinoData(senderData)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "💸 送金完了",
		Description: fmt.Sprintf("**%s** に **%d** チップを送金しました。", recipient.Username, amount),
		Color:       0x2ecc71, // Green
		Fields: []*discordgo.MessageEmbedField{
			{Name: "あなたの現在のチップ", Value: fmt.Sprintf("%d", senderData.Chips), Inline: true},
			{Name: fmt.Sprintf("%sの現在のチップ", recipient.Username), Value: fmt.Sprintf("%d", recipientData.Chips), Inline: true},
		},
	}
	sendEmbedResponse(s, i, embed)
}

func (c *PayCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PayCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *PayCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *PayCommand) GetCategory() string                                                  { return "カジノ" }
