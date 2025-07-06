package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

const musicServerURL = "http://localhost:8080" // Node.jsサーバーのアドレス

type MusicCommand struct{}

func (c *MusicCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "music",
		Description: "音楽を再生します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "play",
				Description: "曲を再生またはキューに追加します",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "query", Description: "曲名またはYouTubeのURL", Required: true},
				},
			},
			{
				Name:        "skip",
				Description: "現在の曲をスキップします",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "stop",
				Description: "再生を停止し、キューをクリアします",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *MusicCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subCommand := i.ApplicationCommandData().Options[0]
	switch subCommand.Name {
	case "play":
		c.handlePlay(s, i, subCommand.Options)
	case "skip":
		c.handleSkip(s, i)
	case "stop":
		c.handleStop(s, i)
	}
}

// handlePlay は /music play コマンドを処理します
func (c *MusicCommand) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// ユーザーがボイスチャンネルにいるか確認
	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		return
	}
	vs, err := findUserVoiceState(s, guild, i.Member.User.ID)
	if err != nil || vs == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "🔊 まずボイスチャンネルに参加してください。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	query := options[0].StringValue()

	// Node.jsサーバーに送るデータ
	payload := map[string]string{
		"guildId":   i.GuildID,
		"channelId": i.ChannelID, // ★修正: コマンドが実行されたテキストチャンネルのIDを渡す
		"userId":    i.Member.User.ID,
		"query":     query,
	}
	jsonPayload, _ := json.Marshal(payload)

	// Node.jsサーバーにリクエストを送信
	_, err = http.Post(fmt.Sprintf("%s/play", musicServerURL), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ 音楽プレーヤーサーバーへの接続に失敗しました。", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// レスポンスの内容に応じて応答
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("リクエストを送信しました: `%s`", query)},
	})
}

func (c *MusicCommand) handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	payload := map[string]string{"guildId": i.GuildID}
	jsonPayload, _ := json.Marshal(payload) // ★修正：jsonPayloadを定義
	http.Post(fmt.Sprintf("%s/skip", musicServerURL), "application/json", bytes.NewBuffer(jsonPayload))
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "⏭️ スキップリクエストを送信しました。"}})
}

func (c *MusicCommand) handleStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	payload := map[string]string{"guildId": i.GuildID}
	jsonPayload, _ := json.Marshal(payload) // ★修正：jsonPayloadを定義
	http.Post(fmt.Sprintf("%s/stop", musicServerURL), "application/json", bytes.NewBuffer(jsonPayload))
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "⏹️ 停止リクエストを送信しました。"}})
}

// ユーザーのボイスステートを見つけるヘルパー関数
func findUserVoiceState(s *discordgo.Session, g *discordgo.Guild, userID string) (*discordgo.VoiceState, error) {
	for _, vs := range g.VoiceStates {
		if vs.UserID == userID {
			return vs, nil
		}
	}
	return nil, fmt.Errorf("ボイスチャンネルにユーザーが見つかりません")
}

func (c *MusicCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *MusicCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *MusicCommand) GetComponentIDs() []string {
	return []string{}
}
func (c *MusicCommand) GetCategory() string {
	return "音楽"
}
