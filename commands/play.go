package commands

import (
	"fmt"
	"luna/interfaces"
	"luna/player"

	"github.com/bwmarrin/discordgo"
)

// PlayCommand は音楽を再生するコマンドです。
type PlayCommand struct {
	Player interfaces.MusicPlayer
	Log    interfaces.Logger
}

func (c *PlayCommand) GetCommandDef() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "play",
		Description: "指定されたURLの音楽を再生します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "url",
				Description: "再生したい音楽のURL (YouTubeなど)",
				Required:    true,
			},
		},
	}
}

func (c *PlayCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	url := options[0].StringValue()

	// ボットがボイスチャンネルに接続しているか確認
	gp := c.Player.GetGuildPlayer(i.GuildID)
	if gp.VoiceConnection == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ボットがボイスチャンネルに接続していません。`/join` コマンドで接続してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Song構造体を作成 (タイトルと作者は仮)
	song := &player.Song{
		URL:    url,
		Title:  "不明なタイトル", // TODO: URLからタイトルを取得するロジックを追加
		Author: "不明な作者",   // TODO: URLから作者を取得するロジックを追加
	}

	// 再生キューに追加
	err := c.Player.Play(i.GuildID, song)
	if err != nil {
		c.Log.Error("Failed to play music", "error", err, "guildID", i.GuildID, "url", url)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("音楽の再生に失敗しました: %s", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 成功メッセージをEmbedで送信
	embed := &discordgo.MessageEmbed{
		Title:       "🎵 再生キューに追加しました！",
		Description: fmt.Sprintf("**[%s](%s)** を再生キューに追加しました。", song.Title, song.URL),
		Color:       0x3498db, // Blue
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *PlayCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {}
func (c *PlayCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)     {}
func (c *PlayCommand) GetComponentIDs() []string                                            { return []string{} }
func (c *PlayCommand) GetCategory() string                                                  { return "音楽" }
