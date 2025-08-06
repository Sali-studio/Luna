package commands

import (
	"fmt"
	"luna/interfaces"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// LeaderboardCommand handles the /leaderboard command.
type LeaderboardCommand struct {
	Store interfaces.DataStore
	Log   interfaces.Logger
}

func (c *LeaderboardCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "leaderboard",
		Description: "サーバー内のチップ所持数ランキングを表示します。",
	}
}

func (c *LeaderboardCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	leaderboard, err := c.Store.GetChipLeaderboard(i.GuildID, 10)
	if err != nil {
		c.Log.Error("Failed to get leaderboard data", "error", err)
		sendErrorResponse(s, i, "リーダーボードの取得に失敗しました。")
		return
	}

	if len(leaderboard) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🏆 チップランキング",
			Description: "まだ誰もカジノで遊んでいないようです！",
			Color:       0x95a5a6, // Gray
		}
		sendEmbedResponse(s, i, embed)
		return
	}

	var description strings.Builder
	for idx, data := range leaderboard {
		var medal string
		switch idx {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("%2d.", idx+1)
		}
		description.WriteString(fmt.Sprintf("%s <@%s> - **%d** チップ\n", medal, data.UserID, data.Chips))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏆 チップランキング",
		Description: description.String(),
		Color:       0xffd700, // Gold
	}
	sendEmbedResponse(s, i, embed)
}

func (c *LeaderboardCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *LeaderboardCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *LeaderboardCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *LeaderboardCommand) GetCategory() string                                                  { return "カジノ" }
