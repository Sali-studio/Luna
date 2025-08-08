package commands

import (
	"fmt"
	"luna/interfaces"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	FishingCost int64 = 10 // Cost to fish once
)

// Fish represents an item that can be caught.
type Fish struct {
	Name   string
	Payout int64
	Rarity string
	Weight int
	Emoji  string
}

// fishTable holds all possible items that can be caught.
var fishTable = []Fish{
	{Name: "藻", Payout: 0, Rarity: "コモン", Weight: 30, Emoji: "🌿"},
	{Name: "長靴", Payout: 1, Rarity: "コモン", Weight: 20, Emoji: "👢"},
	{Name: "小アジ", Payout: 5, Rarity: "コモン", Weight: 25, Emoji: "🐟"},
	{Name: "普通のバス", Payout: 15, Rarity: "アンコモン", Weight: 15, Emoji: "🐠"},
	{Name: "大きなタイ", Payout: 50, Rarity: "レア", Weight: 8, Emoji: "🐡"},
	{Name: "巨大なマグロ", Payout: 100, Rarity: "エピック", Weight: 2, Emoji: "🦑"},
	{Name: "宝箱", Payout: 500, Rarity: "レジェンダリー", Weight: 1, Emoji: "💎"},
}

// FishCommand handles the /fish command.
type FishCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

// NewFishCommand creates a new FishCommand.
func NewFishCommand(store interfaces.DataStore, log interfaces.Logger) *FishCommand {
	return &FishCommand{
		Store: store,
		Log:   log,
	}
}

func (c *FishCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "fish",
		Description: "チップを払って釣りをします。何が釣れるかな？",
	}
}

func (c *FishCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	guildID := i.GuildID

	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for fish", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。")
		return
	}

	if casinoData.Chips < FishingCost {
		sendErrorResponse(s, i, fmt.Sprintf("チップが足りません！釣るには %d チップ必要です。", FishingCost))
		return
	}

	// Deduct cost first
	casinoData.Chips -= FishingCost

	// Perform weighted random selection
	rand.Seed(time.Now().UnixNano())
	totalWeight := 0
	for _, fish := range fishTable {
		totalWeight += fish.Weight
	}
	randomNum := rand.Intn(totalWeight)

	var caughtFish Fish
	currentWeight := 0
	for _, fish := range fishTable {
		currentWeight += fish.Weight
		if randomNum < currentWeight {
			caughtFish = fish
			break
		}
	}

	// Add payout
	casinoData.Chips += caughtFish.Payout

	// Update database
	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data after fishing", "error", err)
		sendErrorResponse(s, i, "結果の保存中にエラーが発生しました。")
		return
	}

	profit := caughtFish.Payout - FishingCost

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎣 %sは釣りをした！", i.Member.User.Username),
		Description: fmt.Sprintf("**%s %s** を釣り上げた！", caughtFish.Emoji, caughtFish.Name),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "結果",
				Value:  fmt.Sprintf("獲得: `%d` チップ\n収支: `%+d` チップ", caughtFish.Payout, profit),
				Inline: true,
			},
			{
				Name:   "所持チップ",
				Value:  fmt.Sprintf("**%d** チップ", casinoData.Chips),
				Inline: true,
			},
		},
		Color:  0x45b3e0, // Water blue
		Footer: &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("レアリティ: %s", caughtFish.Rarity)},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *FishCommand) GetCategory() string {
	return "カジノ"
}

func (c *FishCommand) GetComponentIDs() []string {
	return nil
}

func (c *FishCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}

func (c *FishCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {}
