package commands

import (
	"fmt"
	"luna/interfaces"
	"time"

	"github.com/bwmarrin/discordgo"
)

// DailyCommand handles the /daily command.
type DailyCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *DailyCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "daily",
		Description: "1日1回、2000チップを受け取ります。",
	}
}

func (c *DailyCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	guildID := i.GuildID

	casinoData, err := c.Store.GetCasinoData(guildID, userID)
	if err != nil {
		c.Log.Error("Failed to get casino data for daily command", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。後でもう一度お試しください。")
		return
	}

	// Check if the user is eligible for the daily reward
	if casinoData.LastDaily.Valid && time.Since(casinoData.LastDaily.Time) < 24*time.Hour {
		remaining := (24 * time.Hour) - time.Since(casinoData.LastDaily.Time)
		embed := &discordgo.MessageEmbed{
			Title:       "⏰ また後で！",
			Description: fmt.Sprintf("次のデイリーチップが受け取れるまで、あと **%s** です。", formatDuration(remaining)),
			Color:       0xf1c40f, // Yellow
		}
		sendEmbedResponse(s, i, embed)
		return
	}

	// Grant the daily chips
	casinoData.Chips += 2000
	casinoData.LastDaily.Time = time.Now()
	casinoData.LastDaily.Valid = true

	if err := c.Store.UpdateCasinoData(casinoData); err != nil {
		c.Log.Error("Failed to update casino data for daily command", "error", err)
		sendErrorResponse(s, i, "エラーが発生しました。後でもう一度お試しください。")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎉 デイリーチップ！",
		Description: fmt.Sprintf("**2000** チップを受け取りました！\n現在のあなたのチップ: **%%d**", casinoData.Chips),
		Color:       0xffd700, // Gold
	}
	sendEmbedResponse(s, i, embed)
}

// Helper function to format duration nicely
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d時間 %02d分 %02d秒", h, m, s)
}

func (c *DailyCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *DailyCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *DailyCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *DailyCommand) GetCategory() string                                                  { return "カジノ" }
