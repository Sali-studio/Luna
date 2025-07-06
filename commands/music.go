package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// MusicCommand は音楽再生関連のコマンドを処理します
type MusicCommand struct{}

func (c *MusicCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "music",
		Description: "音楽を再生します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "play",
				Description: "指定した曲を再生します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "query", Description: "YouTubeのURLまたは検索ワード", Required: true},
				},
			},
			{
				Name:        "skip",
				Description: "現在再生中の曲をスキップします",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "stop",
				Description: "音楽を停止し、BotがVCから切断します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *MusicCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subcommand := i.ApplicationCommandData().Options[0]
	switch subcommand.Name {
	case "play":
		c.handlePlay(s, i)
	case "skip":
		c.handleSkip(s, i)
	case "stop":
		c.handleStop(s, i)
	}
}

// Pythonサーバーにリクエストを送信するヘルパー関数
func sendMusicPlayerRequest(endpoint string, data map[string]interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	// Pythonサーバーのポートはplayer.pyと合わせる
	url := fmt.Sprintf("http://localhost:5002%s", endpoint)
	return http.Post(url, "application/json", bytes.NewBuffer(jsonData))
}

func (c *MusicCommand) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ユーザーがいるボイスチャンネルを取得
	vs, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ まずボイスチャンネルに参加してください。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	query := i.ApplicationCommandData().Options[0].Options[0].StringValue()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("🎵 `%s` をキューに追加しました。", query)},
	})

	// Pythonサーバーに再生リクエストを送信
	reqData := map[string]interface{}{
		"guild_id":   i.GuildID,
		"channel_id": vs.ChannelID,
		"query":      query,
	}
	sendMusicPlayerRequest("/play", reqData)
}

func (c *MusicCommand) handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	reqData := map[string]interface{}{"guild_id": i.GuildID}
	_, err := sendMusicPlayerRequest("/skip", reqData)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ スキップに失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "⏩ 曲をスキップしました。"},
	})
}

func (c *MusicCommand) handleStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	reqData := map[string]interface{}{"guild_id": i.GuildID}
	_, err := sendMusicPlayerRequest("/stop", reqData)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ 停止に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "⏹️ 再生を停止し、VCから切断しました。"},
	})
}

// --- 未使用のインターフェースメソッド ---
func (c *MusicCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *MusicCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *MusicCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *MusicCommand) GetCategory() string                                                  { return "音楽" }
